package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"example.com/internal/diarization"
	"example.com/internal/inference"
	"example.com/internal/repository"
	"example.com/internal/services"
	dbservices "example.com/internal/services/db"
	logClient "github.com/EsanSamuel/sensory/LogClient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RecordingHandler struct {
	files         services.FileService
	audio         services.AudioService
	transcription services.TranscribeService
	meetings      *dbservices.MeetingService
	transcripts   *dbservices.TranscriptService
	Logger        *logClient.Client
}

type WhisperInferenceResult struct {
	CPU            float64 `json:"cpu"`
	RAM            uint64  `json:"ram"`
	InferenceTime  float64 `json:"inference_time"`
	RealTimeFactor float64 `json:"real_time_factor"`
	AudioDuration  float64 `json:"audio_duration"`
}

func NewRecordingHandler(files services.FileService, audio services.AudioService, transcription services.TranscribeService, meetings *dbservices.MeetingService, transcripts *dbservices.TranscriptService, Logger *logClient.Client) *RecordingHandler {
	return &RecordingHandler{files: files, audio: audio, transcription: transcription, meetings: meetings, transcripts: transcripts, Logger: Logger}
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

	meetingID, err := uuid.Parse(audio.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("invalid audio meeting ID: %v", err)})
		return
	}
	startedAt := time.Now()
	meeting, err := handler.meetings.CreateMeeting(c.Request.Context(), repository.Meeting{
		ID:              meetingID,
		Title:           recording.Filename,
		StartedAt:       startedAt,
		EndedAt:         startedAt.Add(audio.Duration),
		DurationSeconds: audio.Duration.Seconds(),
		AudioPath:       audio.Path,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("creating meeting: %v", err)})
		return
	}

	duration := time.Now()
	whisper_json, err := handler.transcription.TranscribeWAV(audio.Path, audio.ID, "whisper/whisper.cpp/ggml-tiny.en.bin")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("WHISPER TRANSCRIPTION: %s", whisper_json)
	fmt.Printf("TRANSCRIPTION TIME: %f seconds", time.Since(duration).Seconds())

	transcript_elapsed := time.Since(duration).Seconds()
	whisperInference, err := inference.CollectWhisperMetrics(whisper_json.Pid, time.Duration(transcript_elapsed*float64(time.Second)), float64(audio.Duration))
	whisperInferenceResult := whisperInference

	if err != nil {
		fmt.Printf("WHISPER INFERENCE METRICS ERROR: %v\n", err)
	} else {

		fmt.Printf("WHISPER INFERENCE METRICS:\n%+v\n", whisperInference)
		handler.Logger.INFO(fmt.Sprintf(
			"WHISPER INFERENCE METRICS | audio_duration=%.3f seconds | cpu=%.3f%% | ram=%d bytes | inference_time=%.3f seconds | real_time_factor=%.3f",
			whisperInference.AudioDuration,
			whisperInference.CPU,
			whisperInference.RAM,
			whisperInference.InferenceTime,
			whisperInference.RealTimeFactor,
		))
	}

	transcription_struct, err := handler.transcription.ConvertTranscribedJsonToStruct([]byte(whisper_json.Result))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("CONVERTED TRANSCRIPTION: %v", transcription_struct.Transcription)

	diarization_duration := time.Now()
	diarization_segments, err := diarization.RunDiarization(audio.Path)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("DIARIZATION SEGMENTS: %v", diarization_segments)
	fmt.Printf("DIARIZATION TIME: %f seconds", time.Since(diarization_duration).Seconds())

	merged_segments, err := handler.transcription.MergeTranscriptionWithDiarization(transcription_struct, diarization_segments, audio.ID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	for _, segment := range merged_segments {
		if _, err := handler.transcripts.CreateTranscriptSegment(c.Request.Context(), repository.TranscriptSegment{
			MeetingID: meeting.ID,
			StartTime: segment.Start,
			EndTime:   segment.End,
			SpeakerID: segment.Speaker,
			Speaker:   segment.Speaker,
			Text:      segment.Text,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("saving transcript segment: %v", err)})
			return
		}
	}

	chunks := handler.transcription.ChunkTranscript(merged_segments, 15.0, meetingID)
	for i, chunk := range chunks {
		println("Chunk", i+1)
		println("Start:", chunk.Start)
		println("End:", chunk.End)
		println("Formatted Chunk:")
		println(services.FormatChunk(chunk))
	}

	summarizationResult, err := handler.transcription.SummarizeTranscripts(chunks, audio.ID, meetingID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("summarizing transcript: %v", err)})
		return
	}

	for i, analysis := range summarizationResult {
		println("Summary for Chunk", i+1)
		println("Summary:", analysis.Summary)
		println("Action Items:", analysis.ActionItems)
		println("Decisions:", analysis.Decisions)

		err = handler.meetings.AddMeetingSummary(c.Request.Context(), meetingID, analysis.Summary)

		for _, decision := range analysis.Decisions {
			if _, err := handler.meetings.CreateMeetingDecision(c.Request.Context(), repository.MeetingDecision{
				MeetingID:        meeting.ID,
				Decision:         decision.Text,
				TimestampSeconds: decision.Timestamp,
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("saving meeting decision: %v", err)})
				return
			}
		}

		for _, actionItem := range analysis.ActionItems {
			assignee := ""
			if actionItem.Assignee != nil {
				assignee = *actionItem.Assignee
			}
			if _, err := handler.meetings.CreateMeetingActionItem(c.Request.Context(), repository.MeetingActionItem{
				MeetingID:        meeting.ID,
				Task:             actionItem.Task,
				Assignee:         assignee,
				TimestampSeconds: actionItem.Timestamp,
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("saving meeting action item: %v", err)})
				return
			}
		}
	}

	if err := handler.meetings.AddSummary(c.Request.Context(), meeting.ID, strings.Join(summaryTexts(summarizationResult), " ")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("saving meeting summary: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                   recording.ID,
		"meeting_id":           meeting.ID,
		"filename":             recording.Filename,
		"size":                 recording.Size,
		"whisper_transcript":   whisper_json,
		"diarization_segments": diarization_segments,
		"merged_segments":      merged_segments,
		"whisper_inference":    whisperInferenceResult,
	})
}

func summaryTexts(analyses []services.MeetingAnalysis) []string {
	texts := make([]string, 0, len(analyses))
	for _, analysis := range analyses {
		if analysis.Summary != "" {
			texts = append(texts, analysis.Summary)
		}
	}
	return texts
}
