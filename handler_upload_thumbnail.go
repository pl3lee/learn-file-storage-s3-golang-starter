package main

import (
	"fmt"
	"io"
	"net/http"
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
	mediaType := fileHeader.Header.Get("Content-Type")

	// reads image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read file", err)
		return
	}
	// gets metadata of video from db
	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video metadata", err)
		return
	}
	// checks if user owns the video
	if videoMetaData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to user", err)
		return
	}
	// save thumbnail to global map
	videoThumbnails[videoID] = thumbnail{
		data:      imageData,
		mediaType: mediaType,
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", cfg.port, videoID)

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
