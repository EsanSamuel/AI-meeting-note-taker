package main

import (
	"os"
	"strconv"
	"time"

	"example.com/internal/handlers"
	"example.com/internal/router"
	"example.com/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	storageDir := os.Getenv("RECORDINGS_DIR")
	if storageDir == "" {
		storageDir = "./recordings"
	}

	maxSize := int64(100 * 1024 * 1024)
	if value := os.Getenv("MAX_RECORDING_BYTES"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
			maxSize = parsed
		}
	}

	fileService := services.NewFileService(storageDir, maxSize)
	audioService := services.NewAudioService("", 5*time.Minute)
	transcribeService := services.NewTranscribeService()
	recordingHandler := handlers.NewRecordingHandler(fileService, audioService, transcribeService)

	if err := router.New(recordingHandler).Run(":8080"); err != nil {
		panic(err)
	}
}

// Downloading model command = .\models\download-ggml-model.cmd base.en
// Build whisper command = "cmake -B build", "cmake --build build --config Release"
