package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Recording represents a successfully received screen recording file.
type Recording struct {
	ID       string
	Path     string
	Filename string
	Size     int64
}

type FileService interface {
	ReceiveRecording(ctx context.Context, header *multipart.FileHeader) (*Recording, error)
}

type fileService struct {
	storageDir string
	maxSize    int64 // bytes
}

func NewFileService(storageDir string, maxSize int64) FileService {
	return &fileService{
		storageDir: storageDir,
		maxSize:    maxSize,
	}
}

func (s *fileService) ReceiveRecording(ctx context.Context, header *multipart.FileHeader) (*Recording, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if header.Size > s.maxSize {
		return nil, fmt.Errorf("recording size %d exceeds max allowed %d", header.Size, s.maxSize)
	}

	src, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	if err := os.MkdirAll(s.storageDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}

	id := uuid.NewString()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".webm"
	}
	destPath := filepath.Join(s.storageDir, id+ext)

	dest, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	// Defense in depth: header.Size comes from the client and can lie.
	// Cap the actual read regardless of what the header claims.
	written, err := io.Copy(dest, io.LimitReader(src, s.maxSize+1))
	if err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to write recording to disk: %w", err)
	}
	if written > s.maxSize {
		os.Remove(destPath)
		return nil, fmt.Errorf("recording exceeds max size of %d bytes", s.maxSize)
	}

	return &Recording{
		ID:       id,
		Path:     destPath,
		Filename: header.Filename,
		Size:     written,
	}, nil
}
