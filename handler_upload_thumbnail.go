package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
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

	// TODO: implement the upload here

	const maxMemory = 10 << 20

	r.ParseMultipartForm(maxMemory)

	uploadFile, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse form file", err)
		return
	}
	defer uploadFile.Close()

	mediaType := header.Header.Get("Content-Type")
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "video not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "error fetching video data", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "no permission", nil)
		return
	}

	filename := video.ID.String() + "." + filepath.Base(mediaType)
	filePath := filepath.Join(cfg.assetsRoot, filename)
	thumbnailURL := fmt.Sprintf("http://localhost:%s/", cfg.port) + filePath
	newFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading thumbnail", err)
		return
	}
	if _, err := io.Copy(newFile, uploadFile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading thumbnail", err)
	}

	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error writing to server", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
