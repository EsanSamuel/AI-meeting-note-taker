package repository

import (
	"context"
	"time"

	"example.com/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Meeting struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at"`
	DurationSeconds float64   `json:"duration_seconds"`
	AudioPath       string    `json:"audio_path"`
	VideoPath       string    `json:"video_path"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type MeetingDecision struct {
	ID               uuid.UUID `json:"id"`
	MeetingID        uuid.UUID `json:"meeting_id"`
	Decision         string    `json:"decision"`
	TimestampSeconds float64   `json:"timestamp_seconds"`
	CreatedAt        time.Time `json:"created_at"`
}

type MeetingActionItem struct {
	ID               uuid.UUID `json:"id"`
	MeetingID        uuid.UUID `json:"meeting_id"`
	Task             string    `json:"task"`
	Assignee         string    `json:"assignee"`
	TimestampSeconds float64   `json:"timestamp_seconds"`
	Completed        bool      `json:"completed"`
	CreatedAt        time.Time `json:"created_at"`
}

type MeetingRepository struct {
	q *sqlc.Queries
}

func NewMeetingRepository(pool *pgxpool.Pool) *MeetingRepository {
	return &MeetingRepository{q: sqlc.New(pool)}
}

// ---- Meetings ----

func (r *MeetingRepository) CreateMeeting(ctx context.Context, m Meeting) (Meeting, error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	row, err := r.q.CreateMeeting(ctx, sqlc.CreateMeetingParams{
		ID:              pgtype.UUID{Bytes: m.ID, Valid: true},
		Title:           m.Title,
		StartedAt:       pgtype.Timestamptz{Time: m.StartedAt, Valid: true},
		EndedAt:         pgtype.Timestamptz{Time: m.EndedAt, Valid: true},
		DurationSeconds: pgtype.Float8{Float64: m.DurationSeconds, Valid: true},
		AudioPath:       pgtype.Text{String: m.AudioPath, Valid: m.AudioPath != ""},
		VideoPath:       pgtype.Text{String: m.VideoPath, Valid: m.VideoPath != ""},
	})
	if err != nil {
		return Meeting{}, err
	}
	return meetingFromRow(row), nil
}

func (r *MeetingRepository) GetMeeting(ctx context.Context, id uuid.UUID) (Meeting, error) {
	row, err := r.q.GetMeeting(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return Meeting{}, err
	}
	return meetingFromRow(row), nil
}

func (r *MeetingRepository) ListMeetings(ctx context.Context) ([]Meeting, error) {
	rows, err := r.q.ListMeetings(ctx)
	if err != nil {
		return nil, err
	}
	meetings := make([]Meeting, len(rows))
	for i, row := range rows {
		meetings[i] = meetingFromRow(row)
	}
	return meetings, nil
}

func (r *MeetingRepository) UpdateMeeting(ctx context.Context, m Meeting) (Meeting, error) {
	row, err := r.q.UpdateMeeting(ctx, sqlc.UpdateMeetingParams{
		ID:              pgtype.UUID{Bytes: m.ID, Valid: true},
		Title:           m.Title,
		StartedAt:       pgtype.Timestamptz{Time: m.StartedAt, Valid: true},
		EndedAt:         pgtype.Timestamptz{Time: m.EndedAt, Valid: true},
		DurationSeconds: pgtype.Float8{Float64: m.DurationSeconds, Valid: true},
		AudioPath:       pgtype.Text{String: m.AudioPath, Valid: m.AudioPath != ""},
		VideoPath:       pgtype.Text{String: m.VideoPath, Valid: m.VideoPath != ""},
	})
	if err != nil {
		return Meeting{}, err
	}
	return meetingFromRow(row), nil
}

func (r *MeetingRepository) DeleteMeeting(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteMeeting(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

// ---- Decisions ----

func (r *MeetingRepository) CreateMeetingDecision(ctx context.Context, d MeetingDecision) (MeetingDecision, error) {
	row, err := r.q.CreateMeetingDecision(ctx, sqlc.CreateMeetingDecisionParams{
		MeetingID:        pgtype.UUID{Bytes: d.MeetingID, Valid: true},
		Decision:         d.Decision,
		TimestampSeconds: d.TimestampSeconds,
	})
	if err != nil {
		return MeetingDecision{}, err
	}
	return decisionFromRow(row), nil
}

func (r *MeetingRepository) GetMeetingDecision(ctx context.Context, id uuid.UUID) (MeetingDecision, error) {
	row, err := r.q.GetMeetingDecision(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return MeetingDecision{}, err
	}
	return decisionFromRow(row), nil
}

func (r *MeetingRepository) ListMeetingDecisions(ctx context.Context, meetingID uuid.UUID) ([]MeetingDecision, error) {
	rows, err := r.q.ListMeetingDecisions(ctx, pgtype.UUID{Bytes: meetingID, Valid: true})
	if err != nil {
		return nil, err
	}
	decisions := make([]MeetingDecision, len(rows))
	for i, row := range rows {
		decisions[i] = decisionFromRow(row)
	}
	return decisions, nil
}

func (r *MeetingRepository) DeleteMeetingDecision(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteMeetingDecision(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func (r *MeetingRepository) DeleteMeetingDecisions(ctx context.Context, meetingID uuid.UUID) error {
	return r.q.DeleteMeetingDecisions(ctx, pgtype.UUID{Bytes: meetingID, Valid: true})
}

// ---- Action items ----

func (r *MeetingRepository) CreateMeetingActionItem(ctx context.Context, a MeetingActionItem) (MeetingActionItem, error) {
	row, err := r.q.CreateMeetingActionItem(ctx, sqlc.CreateMeetingActionItemParams{
		MeetingID:        pgtype.UUID{Bytes: a.MeetingID, Valid: true},
		Task:             a.Task,
		Assignee:         pgtype.Text{String: a.Assignee, Valid: a.Assignee != ""},
		TimestampSeconds: a.TimestampSeconds,
	})
	if err != nil {
		return MeetingActionItem{}, err
	}
	return actionItemFromRow(row), nil
}

func (r *MeetingRepository) GetMeetingActionItem(ctx context.Context, id uuid.UUID) (MeetingActionItem, error) {
	row, err := r.q.GetMeetingActionItem(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return MeetingActionItem{}, err
	}
	return actionItemFromRow(row), nil
}

func (r *MeetingRepository) ListMeetingActionItems(ctx context.Context, meetingID uuid.UUID) ([]MeetingActionItem, error) {
	rows, err := r.q.ListMeetingActionItems(ctx, pgtype.UUID{Bytes: meetingID, Valid: true})
	if err != nil {
		return nil, err
	}
	items := make([]MeetingActionItem, len(rows))
	for i, row := range rows {
		items[i] = actionItemFromRow(row)
	}
	return items, nil
}

func (r *MeetingRepository) ListIncompleteActionItems(ctx context.Context, meetingID uuid.UUID) ([]MeetingActionItem, error) {
	rows, err := r.q.ListIncompleteActionItems(ctx, pgtype.UUID{Bytes: meetingID, Valid: true})
	if err != nil {
		return nil, err
	}
	items := make([]MeetingActionItem, len(rows))
	for i, row := range rows {
		items[i] = actionItemFromRow(row)
	}
	return items, nil
}

func (r *MeetingRepository) UpdateMeetingActionItem(ctx context.Context, a MeetingActionItem) (MeetingActionItem, error) {
	row, err := r.q.UpdateMeetingActionItem(ctx, sqlc.UpdateMeetingActionItemParams{
		ID:               pgtype.UUID{Bytes: a.ID, Valid: true},
		Task:             a.Task,
		Assignee:         pgtype.Text{String: a.Assignee, Valid: a.Assignee != ""},
		TimestampSeconds: a.TimestampSeconds,
		Completed:        a.Completed,
	})
	if err != nil {
		return MeetingActionItem{}, err
	}
	return actionItemFromRow(row), nil
}

func (r *MeetingRepository) MarkActionItemCompleted(ctx context.Context, id uuid.UUID) (MeetingActionItem, error) {
	row, err := r.q.MarkActionItemCompleted(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return MeetingActionItem{}, err
	}
	return actionItemFromRow(row), nil
}

func (r *MeetingRepository) MarkActionItemIncomplete(ctx context.Context, id uuid.UUID) (MeetingActionItem, error) {
	row, err := r.q.MarkActionItemIncomplete(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return MeetingActionItem{}, err
	}
	return actionItemFromRow(row), nil
}

func (r *MeetingRepository) DeleteMeetingActionItem(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteMeetingActionItem(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func (r *MeetingRepository) DeleteMeetingActionItems(ctx context.Context, meetingID uuid.UUID) error {
	return r.q.DeleteMeetingActionItems(ctx, pgtype.UUID{Bytes: meetingID, Valid: true})
}

// ---- row -> domain mapping ----

func meetingFromRow(row sqlc.Meeting) Meeting {
	return Meeting{
		ID:              uuid.UUID(row.ID.Bytes),
		Title:           row.Title,
		StartedAt:       row.StartedAt.Time,
		EndedAt:         row.EndedAt.Time,
		DurationSeconds: row.DurationSeconds.Float64,
		AudioPath:       row.AudioPath.String,
		VideoPath:       row.VideoPath.String,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}

func decisionFromRow(row sqlc.MeetingDecision) MeetingDecision {
	return MeetingDecision{
		ID:               uuid.UUID(row.ID.Bytes),
		MeetingID:        uuid.UUID(row.MeetingID.Bytes),
		Decision:         row.Decision,
		TimestampSeconds: row.TimestampSeconds,
		CreatedAt:        row.CreatedAt.Time,
	}
}

func actionItemFromRow(row sqlc.MeetingActionItem) MeetingActionItem {
	return MeetingActionItem{
		ID:               uuid.UUID(row.ID.Bytes),
		MeetingID:        uuid.UUID(row.MeetingID.Bytes),
		Task:             row.Task,
		Assignee:         row.Assignee.String,
		TimestampSeconds: row.TimestampSeconds,
		Completed:        row.Completed,
		CreatedAt:        row.CreatedAt.Time,
	}
}
