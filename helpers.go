package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	var output bytes.Buffer

	cmd.Stdout = &output
	cmd.Run()

	var aspectRatio AspectRatio

	err := json.Unmarshal(output.Bytes(), &aspectRatio)
	if err != nil {
		return "", err
	}

	if len(aspectRatio.Streams) == 0 {
		return "", errors.New(fmt.Sprintf("Unmarshalling failed for %s", filePath))
	}

	isPortrait := aspectRatio.Streams[0].Height/16 == aspectRatio.Streams[0].Width/9
	isLandScape := aspectRatio.Streams[0].Height/9 == aspectRatio.Streams[0].Width/16

	if isPortrait {
		return "portrait", nil
	} else if isLandScape {
		return "landscape", nil
	} else {
		return "other", nil
	}

}

func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		fmt.Printf("ffmpeg error: %v\nstderr: %s\n", err, stderr.String())
		return "", err
	}

	return outputFilePath, nil
}
