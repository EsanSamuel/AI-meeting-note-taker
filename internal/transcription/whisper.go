package transcription

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func TranscribeWAV(audioPath, audioID, modelPath string) (string, error) {
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

/*segments := []services.MergedSegment{
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
		println("Formatted Chunk:")
		println(services.FormatChunk(chunk))
	}

	summarizationResult, err := transcribeService.SummarizeTranscripts(chunks)
	if err != nil {
		println("Error summarizing transcript:", err.Error())
	}

	for i, analysis := range summarizationResult {
		println("Summary for Chunk", i+1)
		println("Summary:", analysis.Summary)
		println("Action Items:", analysis.ActionItems)
		println("Decisions:", analysis.Decisions)
	}*/
