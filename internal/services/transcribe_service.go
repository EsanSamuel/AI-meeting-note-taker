package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"example.com/internal/config"
	"example.com/internal/diarization"
	"example.com/internal/llama"
	"example.com/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

type TranscriptionResult struct {
	Result string
	Pid    int
}

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
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
	SpeakerID string  `json:"speaker_id"`
	Speaker   string  `json:"speaker"`
	Text      string  `json:"text"`
}

type UpdateTranscriptSegmentSpeaker struct {
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
	SpeakerID string  `json:"speaker_id"`
	Speaker   string  `json:"speaker"`
	Text      string  `json:"text"`
}

type TranscriptChunk struct {
	Segments []MergedSegment
	Start    float64
	End      float64
}

type SummarizationResult struct {
	Summary string
	Index   int
	Err     error
}

type MeetingAnalysis struct {
	Summary     string       `json:"summary"`
	Decisions   []Decision   `json:"decisions"`
	ActionItems []ActionItem `json:"action_items"`
}

type Decision struct {
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type ActionItem struct {
	Task      string  `json:"task"`
	Assignee  *string `json:"assignee"`
	Timestamp float64 `json:"timestamp"`
}

type TranscribeService interface {
	ConvertTranscribedJsonToStruct(jsonData []byte) (*WhisperOutput, error)
	TranscribeWAV(audioPath, audioID, modelPath string) (TranscriptionResult, error)
	MergeTranscriptionWithDiarization(transcription *WhisperOutput, diarizationSegments []diarization.Segment, audioID string) ([]MergedSegment, error)
	ChunkTranscript(transcripts []MergedSegment, maxDuration float64, meetingID uuid.UUID) []TranscriptChunk
	SummarizeTranscripts(transcriptChunks []TranscriptChunk, audioID string, meetingID uuid.UUID) ([]MeetingAnalysis, error)
	UpdateSpeakers(ctx context.Context, meetingID uuid.UUID, speakers map[string]string) error
	SummaryAllSummaryChunks(summaries []string) (string, error)
	ChatWithTranscript(ctx context.Context, meetingID uuid.UUID, query string) (string, error)
	GenerateTranscriptFromLocalFile(ctx context.Context, meetingID uuid.UUID) ([]MergedSegment, error)
}

type transcribeService struct {
	llama          llama.LlamaService
	vector         repository.VectorRepository
	transcriptRepo repository.TranscriptRepository
}

func NewTranscribeService(llama llama.LlamaService, vector repository.VectorRepository, transcriptRepo repository.TranscriptRepository) TranscribeService {
	return &transcribeService{
		llama:          llama,
		vector:         vector,
		transcriptRepo: transcriptRepo,
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

func (s *transcribeService) TranscribeWAV(audioPath, audioID, modelPath string) (TranscriptionResult, error) {
	binPath := filepath.Join("whisper", "whisper.cpp", "build", "bin", "Release", "whisper-cli.exe")

	cmd := exec.Command(binPath, "-m", modelPath, "-f", audioPath, "-oj")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return TranscriptionResult{}, fmt.Errorf("whisper failed: %v: %s", err, stderr.String())
	}

	pid := cmd.Process.Pid

	outputFilePath := audioPath + ".json"
	data, err := os.ReadFile(outputFilePath)
	if err != nil {
		return TranscriptionResult{}, fmt.Errorf("reading whisper output at %s (stderr: %s): %w", outputFilePath, stderr.String(), err)
	}
	return TranscriptionResult{Result: string(data), Pid: pid}, nil
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

	serilaizedMerged, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize merged segments: %v", err)
	}

	transcriptPath := filepath.Join(transcriptFolder, fmt.Sprintf("%s_transcript.json", audioID))
	err = os.WriteFile(transcriptPath, serilaizedMerged, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcript file: %v", err)
	}

	return merged, nil
}

func (s *transcribeService) ChunkTranscript(segments []MergedSegment, maxDuration float64, meetingID uuid.UUID) []TranscriptChunk {
	var ctx = context.Background()
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

	for _, chunk := range chunks {
		chunkString := FormatChunk(chunk)
		embedding, err := s.llama.GenerateEmbedding(chunkString)
		if err != nil {
			fmt.Printf("Error generating embedding %s", err)
			return nil
		}

		fmt.Println(embedding, len(embedding))

		_, err = s.vector.CreateVector(ctx, repository.CreateTranscriptVectorEmbeddingParams{MeetingID: pgtype.UUID{Bytes: meetingID, Valid: true}, Chunk: chunkString, Embedding: pgvector.NewVector(embedding)})
		if err != nil {
			fmt.Printf("Error creating vector to db %s", err)
			return nil
		}

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

func (s *transcribeService) SummarizeTranscripts(transcriptChunks []TranscriptChunk, audioID string, meetingID uuid.UUID) ([]MeetingAnalysis, error) {
	if len(transcriptChunks) == 0 {
		return []MeetingAnalysis{}, fmt.Errorf("no transcript chunks provided for summarization")
	}

	summarizedTranscripts := make([]string, len(transcriptChunks))
	results := make(chan SummarizationResult, len(transcriptChunks))

	for i, chunk := range transcriptChunks {
		fmt.Printf(
			"Processing chunk %d: %.2f to %.2f with %d segments\n",
			i+1,
			chunk.Start,
			chunk.End,
			len(chunk.Segments),
		)

		formattedChunks := FormatChunk(chunk)
		prompt := fmt.Sprintf(`You are a meeting analysis assistant.

Analyze the following meeting transcript and extract the important information.

Your response MUST be valid JSON and MUST follow this exact structure:

{
  "summary": "A concise summary of the discussion.",
  "decisions": [
    {
      "text": "A decision that was explicitly made during the meeting.",
      "timestamp": 42.5
    }
  ],
  "action_items": [
    {
      "task": "The task that needs to be completed.",
      "assignee": "The speaker responsible for completing the task.",
      "timestamp": 67.2
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
12. For every decision, use the timestamp of the transcript segment where the decision was made.
13. For every action item, use the timestamp of the transcript segment where the task was assigned or agreed upon.
14. Timestamps must be returned in seconds as a number.
15. Do not create timestamps that are not present in the transcript.

Transcript: %v`, formattedChunks)

		go func(index int, prompt string) {
			//summarizedTranscript, err := s.llama.SummarizeText(prompt)
			summarizedTranscript, err := config.Ai(prompt)
			if err != nil {
				fmt.Printf("Error summarizing transcript chunk: %v\n", err)
				results <- SummarizationResult{Index: index, Err: err}
				return
			}
			fmt.Println(summarizedTranscript)
			results <- SummarizationResult{Index: index, Summary: summarizedTranscript}
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

	var analyses []MeetingAnalysis

	for _, summary := range summarizedTranscripts {
		var summaryStruct MeetingAnalysis
		err := json.Unmarshal([]byte(summary), &summaryStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal meeting analysis: %w", err)
		}
		analyses = append(analyses, summaryStruct)
	}

	if err := os.MkdirAll("summaries", 0755); err != nil {
		return nil, fmt.Errorf("failed to create summaries directory: %w", err)
	}

	filename := fmt.Sprintf("%s_meeting_analysis.json", audioID)
	summaryPath := filepath.Join("summaries", filename)
	summaryData, err := json.MarshalIndent(analyses, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal meeting analysis: %w", err)
	}

	if err := os.WriteFile(summaryPath, summaryData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write meeting analysis file: %w", err)
	}

	return analyses, nil
}

func (s *transcribeService) SummaryAllSummaryChunks(summaries []string) (string, error) {
	combinedSummaries := strings.Join(summaries, "\n\n---\n\n")
	prompt := fmt.Sprintf(`
You are an expert meeting-summary synthesizer.

You will receive multiple summaries generated from different chunks of the same meeting. Your task is to combine them into ONE detailed, coherent, and accurate final meeting summary.

IMPORTANT:
- The summaries are chronological chunks of the same meeting.
- Do NOT treat each chunk as a separate meeting.
- Combine related information across chunks.
- Remove repetitive information.
- Preserve important details, decisions, action items, questions, conclusions, and context.
- Do not invent information that is not present in the provided summaries.
- If a piece of information is uncertain or incomplete, do not guess.
- Preserve speaker names when they are provided.
- If speaker names are unavailable, use the speaker identifiers provided.
- When the same topic appears in multiple chunks, merge the information into one coherent section.
- Do not simply concatenate the summaries.
- Produce a natural, readable summary that represents the meeting as a whole.

Your final summary should include the following sections:

1. OVERVIEW
Provide a concise but informative description of what the entire meeting was about, including its main purpose and outcome.

2. DETAILED DISCUSSION
Describe the major topics discussed during the meeting in logical order rather than simply listing the chunk summaries.
For each important topic:
- Explain what was discussed.
- Include relevant details.
- Identify who raised or discussed the topic when that information is available.
- Include important arguments, explanations, concerns, or suggestions.

3. KEY DECISIONS
List the decisions or agreements reached during the meeting.
For each decision:
- State the decision clearly.
- Identify who was involved when known.
- Include relevant context.

4. ACTION ITEMS
Identify concrete tasks that need to be completed after the meeting.
For each action item, include:
- Task
- Assigned person, if explicitly stated
- Deadline, if explicitly stated
- Status, if it can be determined

Do not invent an assignee or deadline.

5. IMPORTANT POINTS
List important facts, requirements, concerns, observations, or conclusions that should not be forgotten.

6. OPEN QUESTIONS
List questions or issues that remain unresolved at the end of the meeting.
If there are no unresolved questions, state that clearly.

7. FINAL OUTCOME
Provide a concise conclusion describing what was ultimately agreed upon, decided, or accomplished during the meeting.

QUALITY REQUIREMENTS:
- Be detailed but avoid unnecessary repetition.
- Maintain the chronological and logical relationship between topics.
- Distinguish between proposals and actual decisions.
- Distinguish between discussed action items and completed actions.
- Do not turn casual comments into formal decisions.
- Do not create action items unless the summaries indicate that a task exists.
- Do not attribute statements to a speaker unless the speaker is identified in the input.
- Do not add information from your general knowledge.
- Do not mention that you are combining summaries.
- Write as if you directly analyzed the entire meeting transcript.


INPUT:
The following are summaries of consecutive chunks from the same meeting:

%s

OUTPUT:
Return only the final meeting summary using the sections described above.
`, combinedSummaries)

	summary, err := config.Ai(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to summary all summary chunks into one: %w", err)
	}
	return summary, nil
}

func (s *transcribeService) UpdateSpeakers(ctx context.Context, meetingID uuid.UUID, speakers map[string]string) error {
	// Update db file
	for speakerID, speakerName := range speakers {
		err := s.transcriptRepo.UpdateTranscriptSpeaker(ctx, repository.UpdateSpeakerParams{MeetingID: pgtype.UUID{Bytes: meetingID, Valid: true}, SpeakerID: speakerID, Speaker: pgtype.Text{String: speakerName, Valid: speakerName != ""}})
		if err != nil {
			return fmt.Errorf("failed to update meeting speakers name: %w", err)
		}
	}

	// Local file update
	meetingFile := fmt.Sprintf("%s_transcript.json", meetingID)

	meetingFilePath := filepath.Join("transcripts", meetingFile)

	data, err := os.ReadFile(meetingFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var transcriptFile []UpdateTranscriptSegmentSpeaker

	if err := json.Unmarshal(data, &transcriptFile); err != nil {
		return fmt.Errorf("failed to unmarshal transcript file: %w", err)
	}

	for i := range transcriptFile {
		speakerID := transcriptFile[i].Speaker
		if name, ok := speakers[transcriptFile[i].Speaker]; ok {
			transcriptFile[i].SpeakerID = speakerID
			transcriptFile[i].Speaker = name
		}
	}

	transcriptFileIndent, err := json.MarshalIndent(transcriptFile, "", "  ")

	err = os.WriteFile(meetingFilePath, transcriptFileIndent, 0644)
	if err != nil {
		return fmt.Errorf("failed to create transcript file: %v", err)
	}

	return nil
}

func (s *transcribeService) ChatWithTranscript(ctx context.Context, meetingID uuid.UUID, query string) (string, error) {
	queryEmbedding, err := s.llama.GenerateEmbedding(query)
	if err != nil {
		return "", fmt.Errorf("failed to embed query: %v", err)
	}

	// Search for the most relevant chunks for this meeting, ranked by
	// vector distance to the query embedding.
	results, err := s.vector.SearchTranscriptChunk(ctx, repository.SearchTranscriptChunkParams{
		MeetingID: pgtype.UUID{Bytes: meetingID, Valid: true},
		Embedding: pgvector.NewVector(queryEmbedding),
		Limit:     5, // tune based on chunk size / context window
	})
	if err != nil {
		return "", fmt.Errorf("failed to search transcript chunks: %v", err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no relevant transcript context found")
	}

	// Build context string from retrieved chunks
	var contextBuilder strings.Builder
	for _, r := range results {
		contextBuilder.WriteString(r.Chunk)
		contextBuilder.WriteString("\n---\n")
	}

	prompt := fmt.Sprintf(
		"You are answering questions about a meeting transcript. Use only the context below to answer.\n\nContext:\n%s\n\nQuestion: %s\nAnswer:",
		contextBuilder.String(),
		query,
	)

	answer, err := config.Ai(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %v", err)
	}

	return answer, nil
}

func (s *transcribeService) GenerateTranscriptFromLocalFile(ctx context.Context, meetingID uuid.UUID) ([]MergedSegment, error) {
	meetingFile := fmt.Sprintf("%s_transcript.json", meetingID)
	meetingFilePath := filepath.Join("transcripts", meetingFile)

	data, err := os.ReadFile(meetingFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript file: %v", err)
	}
	var transcriptFile []MergedSegment

	if err := json.Unmarshal(data, &transcriptFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transcript file: %w", err)
	}

	return transcriptFile, nil
}
