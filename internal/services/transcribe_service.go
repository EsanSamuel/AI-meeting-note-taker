package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"example.com/internal/diarization"
)

type WhisperOutput struct {
	SystemInfo    string         `json:"systeminfo"`
	Model         WhisperModel   `json:"model"`
	Params        WhisperParams  `json:"params"`
	Result        WhisperResult  `json:"result"`
	Transcription []WhisperEntry `json:"transcription"`
}

type WhisperModel struct {
	Type         string    `json:"type"`
	Multilingual bool      `json:"multilingual"`
	Vocab        int       `json:"vocab"`
	Audio        AudioText `json:"audio"`
	Text         AudioText `json:"text"`
	Mels         int       `json:"mels"`
	Ftype        int       `json:"ftype"`
}

type AudioText struct {
	Ctx   int `json:"ctx"`
	State int `json:"state"`
	Head  int `json:"head"`
	Layer int `json:"layer"`
}

type WhisperParams struct {
	Model     string `json:"model"`
	Language  string `json:"language"`
	Translate bool   `json:"translate"`
}

type WhisperResult struct {
	Language string `json:"language"`
}

type WhisperEntry struct {
	Timestamps Timestamps `json:"timestamps"`
	Offsets    Offsets    `json:"offsets"`
	Text       string     `json:"text"`
}

type Timestamps struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Offsets struct {
	From int64 `json:"from"` // milliseconds
	To   int64 `json:"to"`
}

type DiarizationSegment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker"`
}

type MergedSegment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker"`
	Text    string  `json:"text"`
}

type TranscriptChunk struct {
	Segments []MergedSegment
	Start    float64
	End      float64
}

type TranscribeService interface {
	ConvertTranscribedJsonToStruct(jsonData []byte) (*WhisperOutput, error)
	TranscribeWAV(audioPath, audioID, modelPath string) (string, error)
	MergeTranscriptionWithDiarization(transcription *WhisperOutput, diarizationSegments []diarization.Segment, audioID string) ([]MergedSegment, error)
	ChunkTranscript(transcripts []MergedSegment, maxDuration float64) []TranscriptChunk
}

type transcribeService struct{}

func NewTranscribeService() TranscribeService {
	return &transcribeService{}
}

func (s *transcribeService) ConvertTranscribedJsonToStruct(jsonData []byte) (*WhisperOutput, error) {
	var output WhisperOutput
	err := json.Unmarshal(jsonData, &output)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transcription data: %v", err)
	}
	return &output, nil
}

func (s *transcribeService) TranscribeWAV(audioPath, audioID, modelPath string) (string, error) {
	binPath := filepath.Join("whisper", "whisper.cpp", "build", "bin", "Release", "whisper-cli.exe")

	cmd := exec.Command(binPath, "-m", modelPath, "-f", audioPath, "-oj")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper failed: %v: %s", err, stderr.String())
	}

	outputFilePath := audioPath + ".json"
	data, err := os.ReadFile(outputFilePath)
	if err != nil {
		return "", fmt.Errorf("reading whisper output at %s (stderr: %s): %w", outputFilePath, stderr.String(), err)
	}
	return string(data), nil
}

func overlap(start1, end1, start2, end2 float64) float64 {
	o := math.Min(end1, end2) - math.Max(start1, start2)
	if o < 0 {
		return 0
	}
	return o
}

func (s *transcribeService) MergeTranscriptionWithDiarization(transcription *WhisperOutput, diarizationSegments []diarization.Segment, audioID string) ([]MergedSegment, error) {
	merged := make([]MergedSegment, 0, len(transcription.Transcription))

	for _, w := range transcription.Transcription {
		bestSpeaker := "unknown"
		bestOverlap := 0.0

		for _, d := range diarizationSegments {
			// Whisper offsets are milliseconds; diarization segments are seconds.
			o := overlap(float64(w.Offsets.From)/1000, float64(w.Offsets.To)/1000, d.Start, d.End)
			if o > bestOverlap {
				bestOverlap = o
				bestSpeaker = d.Speaker
			}
		}
		merged = append(merged, MergedSegment{
			Start:   float64(w.Offsets.From) / 1000,
			End:     float64(w.Offsets.To) / 1000,
			Speaker: bestSpeaker,
			Text:    w.Text,
		})
	}

	transcriptFolder := "transcripts"
	if _, err := os.Stat(transcriptFolder); os.IsNotExist(err) {
		if err := os.Mkdir(transcriptFolder, 0755); err != nil {
			return nil, fmt.Errorf("failed to create transcript folder: %v", err)
		}
	}

	transcriptPath := filepath.Join(transcriptFolder, fmt.Sprintf("%s_transcript.json", audioID))
	err := os.WriteFile(transcriptPath, []byte(fmt.Sprintf("%+v", merged)), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcript file: %v", err)
	}

	return merged, nil
}

func (s *transcribeService) ChunkTranscript(segments []MergedSegment, maxDuration float64) []TranscriptChunk {
	var chunks []TranscriptChunk

	if len(segments) == 0 {
		return chunks
	}

	current := TranscriptChunk{
		Start: segments[0].Start,
	}

	for _, segment := range segments {
		duration := segment.End - current.Start

		if len(current.Segments) > 0 && duration > maxDuration {
			current.End = current.Segments[len(current.Segments)-1].End
			chunks = append(chunks, current)

			current = TranscriptChunk{
				Start: segment.Start,
			}
		}

		current.Segments = append(current.Segments, segment)
	}

	if len(current.Segments) > 0 {
		current.End = current.Segments[len(current.Segments)-1].End
		chunks = append(chunks, current)
	}
	return chunks
}
