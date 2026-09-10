package llama

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
)

type LlamaService interface {
	SummarizeText(prompt string) (string, error)
}

type llamaService struct {
}

func NewLlamaService() LlamaService {
	return &llamaService{}
}

func (s *llamaService) SummarizeText(prompt string) (string, error) {
	llamaPath := filepath.Join("llama", "llama.cpp", "build", "bin", "llama-cli.exe")
	modelPath := filepath.Join("llama", "llama.cpp", "models", "qwen", "qwen2.5-1.5b-instruct-q4_k_m.gguf")
	cmd := exec.Command(
		llamaPath,
		"-m", modelPath,
		"-p", prompt,
		"-n", "30",
		"-t", "6",
		"--no-warmup",
		"-st",
	)

	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running llama: %v", err)
	}

	return stdout.String(), nil
}
