package services

import (
	"context"

	"github.com/google/uuid"

	"example.com/internal/repository"
)

type TranscriptService struct {
	repo repository.TranscriptRepository
}

func NewTranscriptService(repo repository.TranscriptRepository) *TranscriptService {
	return &TranscriptService{repo: repo}
}

func (s *TranscriptService) CreateTranscriptSegment(ctx context.Context, seg repository.TranscriptSegment) (repository.TranscriptSegment, error) {
	return s.repo.CreateTranscriptSegment(ctx, seg)
}

func (s *TranscriptService) GetTranscriptSegment(ctx context.Context, id uuid.UUID) (repository.TranscriptSegment, error) {
	return s.repo.GetTranscriptSegment(ctx, id)
}

func (s *TranscriptService) ListTranscriptSegments(ctx context.Context, meetingID uuid.UUID) ([]repository.TranscriptSegment, error) {
	return s.repo.ListTranscriptSegments(ctx, meetingID)
}

func (s *TranscriptService) UpdateTranscriptSegment(ctx context.Context, seg repository.TranscriptSegment) (repository.TranscriptSegment, error) {
	return s.repo.UpdateTranscriptSegment(ctx, seg)
}

func (s *TranscriptService) DeleteTranscriptSegment(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTranscriptSegment(ctx, id)
}

func (s *TranscriptService) DeleteTranscriptSegmentsByMeeting(ctx context.Context, meetingID uuid.UUID) error {
	return s.repo.DeleteTranscriptSegmentsByMeeting(ctx, meetingID)
}
