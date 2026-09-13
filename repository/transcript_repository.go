package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"example.com/internal/db/sqlc"
)

func (r *TranscriptRepository) CreateTranscriptSegment(ctx context.Context, arg sqlc.CreateTranscriptSegmentParams) (sqlc.TranscriptSegment, error) {
	return r.Querier.CreateTranscriptSegment(ctx, arg)
}

func (r *TranscriptRepository) GetTranscriptSegment(ctx context.Context, id pgtype.UUID) (sqlc.TranscriptSegment, error) {
	return r.Querier.GetTranscriptSegment(ctx, id)
}

func (r *TranscriptRepository) ListTranscriptSegments(ctx context.Context, meetingID pgtype.UUID) ([]sqlc.TranscriptSegment, error) {
	return r.Querier.ListTranscriptSegments(ctx, meetingID)
}

func (r *TranscriptRepository) UpdateTranscriptSegment(ctx context.Context, arg sqlc.UpdateTranscriptSegmentParams) (sqlc.TranscriptSegment, error) {
	return r.Querier.UpdateTranscriptSegment(ctx, arg)
}

func (r *TranscriptRepository) DeleteTranscriptSegment(ctx context.Context, id pgtype.UUID) error {
	return r.Querier.DeleteTranscriptSegment(ctx, id)
}

func (r *TranscriptRepository) DeleteTranscriptSegmentsByMeeting(ctx context.Context, meetingID pgtype.UUID) error {
	return r.Querier.DeleteTranscriptSegmentsByMeeting(ctx, meetingID)
}