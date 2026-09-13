package services

import (
	"bytes"
	"context"
	"encoding/binary"
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
	Duration   time.Duration
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
	duration, err := wavDuration(data)
	if err != nil {
		return nil, fmt.Errorf("failed to determine audio length: %w", err)
	}

	return &AudioResult{
		ID:         id,
		Data:       data,
		SampleRate: 16000,
		Channels:   1,
		Duration:   duration,
		Path:       audioPath,
	}, nil
}

func wavDuration(data []byte) (time.Duration, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return 0, fmt.Errorf("invalid WAV data")
	}

	var sampleRate, byteRate uint32
	var dataSize uint32
	for offset := 12; offset+8 <= len(data); {
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkEnd := offset + 8 + chunkSize
		if chunkEnd > len(data) {
			return 0, fmt.Errorf("truncated WAV chunk")
		}
		switch string(data[offset : offset+4]) {
		case "fmt ":
			if chunkSize < 12 {
				return 0, fmt.Errorf("invalid WAV format chunk")
			}
			sampleRate = binary.LittleEndian.Uint32(data[offset+12 : offset+16])
			byteRate = binary.LittleEndian.Uint32(data[offset+16 : offset+20])
		case "data":
			dataSize = uint32(chunkSize)
		}
		offset = chunkEnd
		if chunkSize%2 != 0 {
			offset++
		}
	}
	if sampleRate == 0 || byteRate == 0 {
		return 0, fmt.Errorf("missing WAV format information")
	}
	return time.Duration((uint64(dataSize) * uint64(time.Second)) / uint64(byteRate)), nil
}
