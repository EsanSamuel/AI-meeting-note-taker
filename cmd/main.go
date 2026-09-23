package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"example.com/internal/config"
	"example.com/internal/db"
	"example.com/internal/handlers"
	"example.com/internal/llama"
	"example.com/internal/repository"
	"example.com/internal/router"
	"example.com/internal/services"
	dbservices "example.com/internal/services/db"
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

	LOGGER := config.InitLogger()

	db, err := db.InitDatabase(&cfg.Database)
	if err != nil {
		fmt.Printf("error initializing database: %v\n", err)
		return
	}

	// repositories
	meetingRepository := repository.NewMeetingRepository(db)
	transcriptRepository := repository.NewTranscriptRepository(db)
	vectorRepository := repository.NewVectorRepository(db)

	// services
	fileService := services.NewFileService(storageDir, maxSize)
	audioService := services.NewAudioService("", 5*time.Minute)
	llamaService := llama.NewLlamaService()
	transcribeService := services.NewTranscribeService(llamaService, vectorRepository, transcriptRepository)
	meetingService := dbservices.NewMeetingService(meetingRepository)
	transcriptService := dbservices.NewTranscriptService(transcriptRepository)

	// handlers
	recordingHandler := handlers.NewRecordingHandler(fileService, audioService, transcribeService, meetingService, transcriptService, LOGGER)
	meetingHandler := handlers.NewMeetingHandler(meetingService)
	transcriptHandler := handlers.NewTranscriptHandler(transcriptService, transcribeService)

	if err := llamaService.StartEmbeddingServer(); err != nil {
		fmt.Printf("Error starting embedding server %s", err)
		return
	}

	if err := router.New(recordingHandler, meetingHandler, transcriptHandler).Run(serverAddr); err != nil {
		panic(err)
	}
}

// Downloading whisper model command = .\models\download-ggml-model.cmd base.en
// Build whisper command = "cmake -B build", "cmake --build build --config Release"
