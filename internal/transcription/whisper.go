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
