package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server error", err)
		return
	}
	defer file.Close()

	contentHeader := header.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(contentHeader)
	if mediatype != "image/jpeg" && mediatype != "image/png" {
		respondWithError(w, http.StatusForbidden, "MediaType not accepted", nil)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server error", err)
		return
	}

	fileType := strings.Split(mediatype, "/")[1]

	fileName := fmt.Sprintf("%s.%s", videoIDString, fileType)
	filePath := filepath.Join(cfg.assetsRoot, fileName)
	filePointer, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server error", err)
		return
	}
	_, err = io.Copy(filePointer, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server error", err)
		return
	}

	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server error", err)
		return
	}

	thumbnailUrl := fmt.Sprintf("http://localhost:%s/%s", cfg.port, filePath)
	metadata.ThumbnailURL = &thumbnailUrl

	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server error", err)
		return
	}

	respondWithJSON(w, http.StatusOK, metadata)
}
