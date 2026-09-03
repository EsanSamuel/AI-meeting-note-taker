package handlers

import (
	"fmt"
	"net/http"
	"os"

	"example.com/internal/diarization"
	"example.com/internal/services"
	"github.com/gin-gonic/gin"
)

/*type AudioExtractor interface {
	ExtractAudio(ctx context.Context, input string) (*services.AudioResult, error)
}*/

type RecordingHandler struct {
	files         services.FileService
	audio         services.AudioService
	transcription services.TranscribeService
}

func NewRecordingHandler(files services.FileService, audio services.AudioService, transcription services.TranscribeService) *RecordingHandler {
	return &RecordingHandler{files: files, audio: audio, transcription: transcription}
}

func (handler *RecordingHandler) Create(c *gin.Context) {
	file, err := c.FormFile("recording")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recording file is required"})
		return
	}

	recording, err := handler.files.ReceiveRecording(c.Request.Context(), file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer os.Remove(recording.Path)

	audio, err := handler.audio.ExtractAudio(c.Request.Context(), recording.Path)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	whisper_json, err := handler.transcription.TranscribeWAV(audio.Path, audio.ID, "whisper/whisper.cpp/ggml-tiny.en.bin")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("WHISPER TRANSCRIPTION: %s", whisper_json)

	transcription_struct, err := handler.transcription.ConvertTranscribedJsonToStruct([]byte(whisper_json))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("CONVERTED TRANSCRIPTION: %v", transcription_struct.Transcription)

	diarization_segments, err := diarization.RunDiarization(audio.Path)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("DIARIZATION SEGMENTS: %v", diarization_segments)

	merged_segments, err := handler.transcription.MergeTranscriptionWithDiarization(transcription_struct, diarization_segments)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                   recording.ID,
		"filename":             recording.Filename,
		"size":                 recording.Size,
		"whisper_transcript":   whisper_json,
		"diarization_segments": diarization_segments,
		"merged_segments":      merged_segments,
	})
}
