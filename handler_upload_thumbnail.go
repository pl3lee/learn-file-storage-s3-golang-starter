package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// implement the upload here
	// This is the same as 10 * 1024 * 1024, which is 10MB
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to parse multipart form", err)
		return
	}

	// get thumbnail image data from the form
	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	// Parse out mime type.
	mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Malformed mime type", nil)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Unsupported file type", nil)
		return
	}

	// extract file extension
	fileExtension := strings.Split(mediaType, "/")[1]

	// gets metadata of video from db
	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video metadata", err)
		return
	}
	// checks if user owns the video
	if videoMetaData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to user", fmt.Errorf("video does not belong to user"))
		return
	}

	randomID := make([]byte, 32)
	_, err = rand.Read(randomID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't generate random ID", err)
		return
	}
	randomIDB64 := base64.RawURLEncoding.EncodeToString(randomID)

	// generate file path
	filePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%v.%v", randomIDB64, fileExtension))

	// create an empty file with the filepath
	fileOnFS, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot create file with filepath", err)
		return
	}
	// remember to close the file!
	defer fileOnFS.Close()

	// copy image to the file
	_, err = io.Copy(fileOnFS, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot copy to file system", err)
		return
	}

	// create base url
	baseURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%v", cfg.port),
		// Path:   fmt.Sprintf("/assets/%v.%v", randomIDB64, fileExtension),
		Path: filePath,
	}
	// Get the formatted URL string
	thumbnailURL := baseURL.String()

	// Create a new struct for the video
	newUpdatedVideo := database.Video{
		ID:                videoID,
		CreatedAt:         videoMetaData.CreatedAt,
		UpdatedAt:         time.Now(),
		ThumbnailURL:      &thumbnailURL,
		VideoURL:          videoMetaData.VideoURL,
		CreateVideoParams: videoMetaData.CreateVideoParams,
	}
	// save in db
	err = cfg.db.UpdateVideo(newUpdatedVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't save thumbnail url to database", err)
		return
	}
	respondWithJSON(w, http.StatusOK, newUpdatedVideo)
}
