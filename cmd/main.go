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

	summaries := []string{
		`
The meeting began with a discussion about the current state of the backend API. Samuel reported that the authentication endpoints are working correctly, but the transcript processing endpoint is still experiencing occasional delays when processing longer recordings. John suggested investigating whether the delays are caused by the transcription service or by the database queries used to store transcript segments.
	`,

		`
The team discussed the transcript processing pipeline in more detail. Samuel explained that audio is first converted to WAV format, then passed through the transcription service before the resulting segments are stored in PostgreSQL. The team agreed that processing should happen asynchronously so that users do not have to wait for the entire recording to finish before receiving a response.
	`,

		`
John raised concerns about storing large audio files directly on the application server. He suggested using object storage for audio and video files while keeping metadata and transcript information in PostgreSQL. Samuel agreed and mentioned that this would also make it easier to scale the application across multiple servers.
	`,

		`
The discussion then moved to the AI summarization system. Samuel explained that long transcripts are divided into smaller chunks before being sent to the language model because sending the entire transcript at once can exceed the model context window. Each chunk currently receives its own summary. John pointed out that these individual summaries need to be combined afterward so that the final result represents the entire meeting rather than separate conversations.
	`,

		`
The team discussed what the final meeting summary should contain. They agreed that it should provide an overall overview, describe the major topics discussed, identify decisions, list action items with responsible people when known, and highlight unresolved questions. They also agreed that the final summarizer should remove duplicate information that appears in multiple chunk summaries.
	`,

		`
Samuel demonstrated the speaker identification system. The transcript currently contains identifiers such as SPEAKER_00 and SPEAKER_01. The team discussed allowing users to rename these speakers to real names from the frontend. Samuel proposed keeping a permanent speaker_id while allowing the display name to change so that renaming a speaker would not destroy the underlying speaker identity.
	`,

		`
The team also discussed how speaker renaming should be persisted. The frontend will send all speaker mappings in a single request, such as SPEAKER_00 mapped to Samuel and SPEAKER_01 mapped to John. The backend will update the corresponding transcript segments in PostgreSQL and also update the local transcript JSON file.
	`,

		`
Before ending the meeting, John recommended adding better error handling around transcript processing. The system should detect failed transcription or summarization operations instead of returning a successful response when no transcript records were actually updated. Samuel agreed to improve the repository layer so that database update counts can be checked and zero-row updates can be detected.
	`,
	}

	summary, err := transcribeService.SummaryAllSummaryChunks(summaries)
	fmt.Printf("Summary: %s", summary)

	if err := router.New(recordingHandler, meetingHandler, transcriptHandler).Run(serverAddr); err != nil {
		panic(err)
	}
}

// Downloading whisper model command = .\models\download-ggml-model.cmd base.en
// Build whisper command = "cmake -B build", "cmake --build build --config Release"
