package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type AudioResult struct {
	ID         string
	Data       []byte
	SampleRate int
	Channels   int
	Path       string
}

type AudioService interface {
	ExtractAudio(ctx context.Context, input string) (*AudioResult, error)
}

type audioService struct {
	baseTempDir string
	timeout     time.Duration
}

func NewAudioService(baseTempDir string, timeout time.Duration) AudioService {
	if baseTempDir == "" {
		baseTempDir = os.TempDir()
	}
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &audioService{baseTempDir: baseTempDir, timeout: timeout}
}

func (s *audioService) ExtractAudio(ctx context.Context, input string) (*AudioResult, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if _, err := os.Stat(input); err != nil {
		return nil, fmt.Errorf("input file not accessible: %w", err)
	}

	id := uuid.NewString()

	audioDir := "audio"
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create audio directory: %w", err)
	}

	audioPath := filepath.Join(audioDir, id+".wav")
	var stderr bytes.Buffer
	cmd := ffmpeg_go.Input(input).
		Output(audioPath, ffmpeg_go.KwArgs{
			"vn": "",    // disable video
			"ac": 1,     // mono
			"ar": 16000, // 16 kHz, tuned for downstream STT
		}).
		WithErrorOutput(&stderr)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg extraction failed: %w: %s", err, stderr.String())
	}

	data, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read extracted audio: %w", err)
	}

	if err := os.WriteFile(audioPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to save extracted audio: %w", err)
	}

	return &AudioResult{
		ID:         id,
		Data:       data,
		SampleRate: 16000,
		Channels:   1,
		Path:       audioPath,
	}, nil
}
