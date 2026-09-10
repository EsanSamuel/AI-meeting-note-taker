package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"example.com/internal/diarization"
	"example.com/internal/llama"
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
	SummarizeTranscripts(transcriptChunks []TranscriptChunk) ([]string, error)
}

type transcribeService struct {
	llama llama.LlamaService
}

func NewTranscribeService(llama llama.LlamaService) TranscribeService {
	return &transcribeService{
		llama: llama,
	}
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

	// Save the merged segments to a JSON file for later reference
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

	// create the first chunk
	current := TranscriptChunk{
		Start: segments[0].Start,
	}

	// iterate through the segments and create chunks based on the maxDuration
	for _, segment := range segments {
		duration := segment.End - current.Start

		// if the current chunk has segments and the duration exceeds maxDuration, finalize the current chunk and start a new one
		if len(current.Segments) > 0 && duration > maxDuration {
			current.End = current.Segments[len(current.Segments)-1].End
			chunks = append(chunks, current)

			// start a new chunk with the current segment
			current = TranscriptChunk{
				Start: segment.Start,
			}
		}
		// add the segment to the current chunk
		current.Segments = append(current.Segments, segment)
	}

	// finalize the last chunk if it has segments
	if len(current.Segments) > 0 {
		current.End = current.Segments[len(current.Segments)-1].End
		chunks = append(chunks, current)
	}
	return chunks
}

func FormatChunk(chunk TranscriptChunk) string {
	var b strings.Builder

	for _, segment := range chunk.Segments {
		fmt.Fprintf(
			&b,
			"[%.2f - %.2f] %s: %s\n",
			segment.Start,
			segment.End,
			segment.Speaker,
			segment.Text,
		)
	}

	return b.String()
}

type SummarizationResult struct {
	Summary string
	Index   string
	Err     error
}

func (s *transcribeService) SummarizeTranscripts(transcriptChunks []TranscriptChunk) ([]string, error) {
	if len(transcriptChunks) == 0 {
		return nil, fmt.Errorf("no transcript chunks provided for summarization")
	}

	summarizedTranscripts := make([]string, 0, len(transcriptChunks))
	results := make(chan SummarizationResult, len(transcriptChunks))

	for i, chunk := range transcriptChunks {
		fmt.Printf("Chunk from %.2f to %.2f with %d segments\n", i, chunk.Start, chunk.End, len(chunk.Segments))

		formattedChunks := FormatChunk(chunk)
		prompt := fmt.Sprintf(`You are a meeting analysis assistant.

Analyze the following meeting transcript and extract the important information.

Your response MUST be valid JSON and MUST follow this exact structure:

{
  "summary": "A concise summary of the discussion.",
  "decisions": [
    "A decision that was explicitly made during the meeting."
  ],
  "action_items": [
    {
      "task": "The task that needs to be completed.",
      "assignee": "The speaker responsible for completing the task."
    }
  ]
}

Rules:

1. Return ONLY valid JSON.
2. Do not use Markdown.
3. Do not include Markdown code fences around the response.
4. The "summary" must briefly describe the important points discussed.
5. Only include decisions that were actually made. Do not invent decisions.
6. Only include action items that were actually assigned or clearly agreed upon.
7. If the assignee is not clear, use null.
8. If there are no decisions, return an empty array.
9. If there are no action items, return an empty array.
10. Preserve the speaker identifiers exactly as they appear in the transcript.
11. Do not invent information that is not present in the transcript.

Transcript: %v`, formattedChunks)

		go func(index int, prompt string) {
			summarizedTranscript, err := s.llama.SummarizeText(prompt)
			if err != nil {
				fmt.Printf("Error summarizing transcript chunk: %v\n", err)
				results <- SummarizationResult{Index: fmt.Sprintf("Chunk %d", index), Err: err}
				return
			}
			fmt.Println(summarizedTranscript)
			results <- SummarizationResult{Index: fmt.Sprintf("Chunk %d", index), Summary: summarizedTranscript}
		}(i, prompt)
	}

	// Collect exactly one result for every chunk.
	for range transcriptChunks {
		result := <-results

		if result.Err != nil {
			return nil, fmt.Errorf(
				"failed to summarize chunk %d: %w",
				result.Index,
				result.Err,
			)
		}

		summarizedTranscripts[result.Index] = result.Summary
	}

	return summarizedTranscripts, nil
}
