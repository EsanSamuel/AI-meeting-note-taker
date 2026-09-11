package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"example.com/internal/config"
	"example.com/internal/handlers"
	"example.com/internal/llama"
	"example.com/internal/router"
	"example.com/internal/services"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	s := cfg.Server
	serverAddr := fmt.Sprintf(":%s", s.Port)

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
	llamaService := llama.NewLlamaService()
	transcribeService := services.NewTranscribeService(llamaService)
	recordingHandler := handlers.NewRecordingHandler(fileService, audioService, transcribeService)

	if err := router.New(recordingHandler).Run(serverAddr); err != nil {
		panic(err)
	}
}

// Downloading model command = .\models\download-ggml-model.cmd base.en
// Build whisper command = "cmake -B build", "cmake --build build --config Release"
