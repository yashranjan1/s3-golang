package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing video ID", err)
		return
	}

	authHeader, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error fetching auth token", err)
		return
	}

	userID, err := auth.ValidateJWT(authHeader, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error authenticating user", err)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Video does not exist", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "An error occured while trying to fetch video metadata", err)
		return
	}

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized for this action", err)
		return
	}

	video, headers, err := r.FormFile("video")
	defer r.Body.Close()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "An error occured while fetching video data", err)
		return
	}

	videoHeaderType := headers.Header.Get("Content-Type")

	mimeType, _, err := mime.ParseMediaType(videoHeaderType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while fetching mimetype of file", err)
		return
	} else if mimeType != "video/mp4" {
		respondWithError(w, http.StatusUnprocessableEntity, "Invalid file type", nil)
		return
	}

	filePointer, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Server side error", err)
		return
	}

	defer filePointer.Close()
	defer os.Remove(filePointer.Name())

	_, err = io.Copy(filePointer, video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Creating file on fs failed", err)
		return
	}

	fastStartFilePath, err := processVideoForFastStart(filePointer.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Creating a faststart file failed", err)
		return
	}

	fastStartFile, err := os.Open(fastStartFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Reading the faststart file failed", err)
		return
	}
	defer fastStartFile.Close()

	aspectRatio, err := getVideoAspectRatio(filePointer.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching file aspect ratio", err)
		return
	}

	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	fileName := hex.EncodeToString(randomBytes)

	fullFileName := fmt.Sprintf("%s/%s.%s", aspectRatio, fileName, "mp4")

	filePointer.Seek(0, io.SeekStart)

	params := s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fullFileName),
		Body:        fastStartFile,
		ContentType: aws.String(mimeType),
	}

	_, err = cfg.s3client.PutObject(context.Background(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to upload file", err)
		return
	}

	newVideoURL := fmt.Sprintf("%s/%s", cfg.s3CfDistribution, *params.Key)
	videoData.VideoURL = &newVideoURL
	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}
	respondWithJSON(w, http.StatusAccepted, videoData)
}
