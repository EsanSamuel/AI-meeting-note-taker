package llama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
)

type LlamaService interface {
	StartLlamaAIServer() error
	SummarizeText(prompt string) (string, error)
	StartEmbeddingServer() error
	GenerateEmbedding(text string) ([]float32, error)
}

type llamaService struct {
}

type EmbeddingResponse struct {
	index     int
	Embedding [][]float32
}

type ChatRequest struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func NewLlamaService() LlamaService {
	return &llamaService{}
}

func (s *llamaService) StartLlamaAIServer() error {
	llamaPath := filepath.Join("llama", "llama.cpp", "build", "bin", "llama-server.exe")
	modelPath := filepath.Join("llama", "llama.cpp", "models", "qwen", "qwen2.5-1.5b-instruct-q4_k_m.gguf")
	cmd := exec.Command(
		llamaPath,
		"-m", modelPath,
		"--port", "8083",
	)
	return cmd.Start()
}

func (s *llamaService) SummarizeText(prompt string) (string, error) {
	body := ChatRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(
		"http://localhost:8083/v1/chat/completions",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf(
			"llama server returned %s: %s",
			resp.Status,
			string(body),
		)
	}

	var result ChatResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("summary decode failed: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("llama server returned no choices")
	}

	return result.Choices[0].Message.Content, nil
}

func (s *llamaService) StartEmbeddingServer() error {
	llamaPath := filepath.Join("llama", "llama.cpp", "build", "bin", "llama-server.exe")
	modelPath := filepath.Join("llama", "llama.cpp", "models", "embedding", "bge-small-en-v1.5-q4_k_m.gguf")
	cmd := exec.Command(
		llamaPath,
		"-m", modelPath,
		"--embedding",
		"--port", "8082",
	)

	return cmd.Start()
}

// llama\llama.cpp\build\bin>llama-server.exe  -m ..\models\embedding\bge-small-en-v1.5-q4_k_m.gguf     --embedding   --port 8
func (s *llamaService) GenerateEmbedding(text string) ([]float32, error) {
	body := map[string]string{
		"content": text,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("http://localhost:8082/embedding", "application/json",
		bytes.NewReader(data))

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf(
			"embedding server returned %s: %s",
			resp.Status,
			string(body),
		)
	}

	var result []EmbeddingResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("GenerateEmbeddingServer decode failed: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	if len(result[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding is empty")
	}

	embedding := result[0].Embedding[0]

	if len(embedding) != 384 {
		return nil, fmt.Errorf(
			"expected 384 dimensions, got %d",
			len(embedding),
		)
	}

	return embedding, nil
}
