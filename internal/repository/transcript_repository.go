package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/internal/db/sqlc"
)

type TranscriptSegment struct {
	ID        uuid.UUID `json:"id"`
	MeetingID uuid.UUID `json:"meeting_id"`
	StartTime float64   `json:"start_time"`
	EndTime   float64   `json:"end_time"`
	SpeakerID string    `json:"speaker_id"`
	Speaker   string    `json:"speaker"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateSpeakerParams struct {
	Speaker   pgtype.Text `json:"speaker"`
	MeetingID pgtype.UUID `json:"meeting_id"`
	SpeakerID string      `json:"speaker_id"`
}

type TranscriptRepository interface {
	CreateTranscriptSegment(ctx context.Context, s TranscriptSegment) (TranscriptSegment, error)
	GetTranscriptSegment(ctx context.Context, id uuid.UUID) (TranscriptSegment, error)
	ListTranscriptSegments(ctx context.Context, meetingID uuid.UUID) ([]TranscriptSegment, error)
	UpdateTranscriptSegment(ctx context.Context, s TranscriptSegment) (TranscriptSegment, error)
	UpdateTranscriptSpeaker(ctx context.Context, s UpdateSpeakerParams) error
	DeleteTranscriptSegment(ctx context.Context, id uuid.UUID) error
	DeleteTranscriptSegmentsByMeeting(ctx context.Context, meetingID uuid.UUID) error
}

type transcriptRepository struct {
	q *sqlc.Queries
}

func NewTranscriptRepository(pool *pgxpool.Pool) TranscriptRepository {
	return &transcriptRepository{q: sqlc.New(pool)}
}

func (r *transcriptRepository) CreateTranscriptSegment(ctx context.Context, s TranscriptSegment) (TranscriptSegment, error) {
	row, err := r.q.CreateTranscriptSegment(ctx, sqlc.CreateTranscriptSegmentParams{
		MeetingID: pgtype.UUID{Bytes: s.MeetingID, Valid: true},
		StartTime: s.StartTime,
		EndTime:   s.EndTime,
		SpeakerID: s.SpeakerID,
		Speaker:   pgtype.Text{String: s.Speaker, Valid: s.Speaker != ""},
		Text:      s.Text,
	})
	if err != nil {
		return TranscriptSegment{}, err
	}
	return segmentFromRow(row), nil
}

func (r *transcriptRepository) GetTranscriptSegment(ctx context.Context, id uuid.UUID) (TranscriptSegment, error) {
	row, err := r.q.GetTranscriptSegment(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return TranscriptSegment{}, err
	}
	return segmentFromRow(row), nil
}

func (r *transcriptRepository) ListTranscriptSegments(ctx context.Context, meetingID uuid.UUID) ([]TranscriptSegment, error) {
	rows, err := r.q.ListTranscriptSegments(ctx, pgtype.UUID{Bytes: meetingID, Valid: true})
	if err != nil {
		return nil, err
	}
	segments := make([]TranscriptSegment, len(rows))
	for i, row := range rows {
		segments[i] = segmentFromRow(row)
	}
	return segments, nil
}

func (r *transcriptRepository) UpdateTranscriptSegment(ctx context.Context, s TranscriptSegment) (TranscriptSegment, error) {
	row, err := r.q.UpdateTranscriptSegment(ctx, sqlc.UpdateTranscriptSegmentParams{
		ID:        pgtype.UUID{Bytes: s.ID, Valid: true},
		Text:      s.Text,
		StartTime: s.StartTime,
		EndTime:   s.EndTime,
		Speaker:   pgtype.Text{String: s.Speaker, Valid: s.Speaker != ""},
	})
	if err != nil {
		return TranscriptSegment{}, err
	}
	return segmentFromRow(row), nil
}

func (r *transcriptRepository) UpdateTranscriptSpeaker(ctx context.Context, s UpdateSpeakerParams) error {
	_, err := r.q.UpdateSpeakers(ctx, sqlc.UpdateSpeakersParams{
		SpeakerID: s.SpeakerID,
		MeetingID: s.MeetingID,
		Speaker:   s.Speaker,
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *transcriptRepository) DeleteTranscriptSegment(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTranscriptSegment(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func (r *transcriptRepository) DeleteTranscriptSegmentsByMeeting(ctx context.Context, meetingID uuid.UUID) error {
	return r.q.DeleteTranscriptSegmentsByMeeting(ctx, pgtype.UUID{Bytes: meetingID, Valid: true})
}

func segmentFromRow(row sqlc.TranscriptSegment) TranscriptSegment {
	return TranscriptSegment{
		ID:        uuid.UUID(row.ID.Bytes),
		MeetingID: uuid.UUID(row.MeetingID.Bytes),
		StartTime: row.StartTime,
		EndTime:   row.EndTime,
		SpeakerID: row.SpeakerID,
		Speaker:   row.Speaker.String,
		Text:      row.Text,
		CreatedAt: row.CreatedAt.Time,
	}
}