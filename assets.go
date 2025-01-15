package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(mediaType string) string {
	base := make([]byte, 32)
	_, err := rand.Read(base)
	if err != nil {
		panic("failed to generate random bytes")
	}
	id := base64.RawURLEncoding.EncodeToString(base)

	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", id, ext)
}

func (cfg apiConfig) getObjectURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func getVideoAspectRatio(filePath string) (string, error) {
	// creates a CLI command to be executed later
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	// captures stdout of the command to outBuffer
	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	// run the command
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// extracts video info
	type videoInfo struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	var video videoInfo

	if err := json.Unmarshal(outBuffer.Bytes(), &video); err != nil {
		return "", err
	}
	if len(video.Streams) < 1 {
		return "", fmt.Errorf("Video stream does not exist")
	}
	w, h := float64(video.Streams[0].Width), float64(video.Streams[0].Height)
	ratio := w / h
	target169 := 16.0 / 9.0
	target916 := 9.0 / 16.0
	if math.Abs(ratio-target169) < 0.1 {
		return "16:9", nil
	} else if math.Abs(ratio-target916) < 0.1 {
		return "9:16", nil
	}
	return "other", nil
}
