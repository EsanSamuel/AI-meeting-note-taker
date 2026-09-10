package main

import (
	//"fmt"
	"os"
	"strconv"
	"time"

	"example.com/internal/handlers"
	//"example.com/internal/llama"
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

	/*go func() {
		message, err := llama.SummarizeText("Hello, can you provide a brief summary of the current state of AI research and its potential future applications?")
		if err != nil {
			fmt.Printf("Error summarizing text: %v\n", err)
			return
		}
		fmt.Printf("LLaMA Summary: %s\n", message)
	}()*/

	fileService := services.NewFileService(storageDir, maxSize)
	audioService := services.NewAudioService("", 5*time.Minute)
	transcribeService := services.NewTranscribeService()
	recordingHandler := handlers.NewRecordingHandler(fileService, audioService, transcribeService)

	segments := []services.MergedSegment{
		{Start: 0, End: 1.72, Speaker: "SPEAKER_01", Text: "Hi, how are you?"},
		{Start: 4.3, End: 6.7, Speaker: "SPEAKER_00", Text: "I'm good, thank you, and you."},
		{Start: 8, End: 10.5, Speaker: "SPEAKER_01", Text: "I'm fine, what are you doing here?"},
		{Start: 11.8, End: 14.8, Speaker: "SPEAKER_00", Text: "I'm just taking a walk. The weather is nice today."},
		{Start: 16.7, End: 19.8, Speaker: "SPEAKER_01", Text: "Yes, it is. Want to join me for coffee?"},
		{Start: 21.9, End: 23.7, Speaker: "SPEAKER_00", Text: "Great idea, let's go."},
	}

	chunks := transcribeService.ChunkTranscript(segments, 15.0)
	for i, chunk := range chunks {
		println("Chunk", i+1)
		println("Start:", chunk.Start)
		println("End:", chunk.End)
		for _, segment := range chunk.Segments {
			println("  Segment Start:", segment.Start)
			println("  Segment End:", segment.End)
			println("  Speaker:", segment.Speaker)
			println("  Text:", segment.Text)
		}
	}

	if err := router.New(recordingHandler).Run(":8080"); err != nil {
		panic(err)
	}
}

// Downloading model command = .\models\download-ggml-model.cmd base.en
// Build whisper command = "cmake -B build", "cmake --build build --config Release"
