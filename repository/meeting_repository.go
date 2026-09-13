package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"example.com/internal/db/sqlc"
)

func (r *MeetingRepository) CreateMeeting(ctx context.Context, arg sqlc.CreateMeetingParams) (sqlc.Meeting, error) {
	return r.Querier.CreateMeeting(ctx, arg)
}

func (r *MeetingRepository) GetMeeting(ctx context.Context, id pgtype.UUID) (sqlc.Meeting, error) {
	return r.Querier.GetMeeting(ctx, id)
}

func (r *MeetingRepository) ListMeetings(ctx context.Context) ([]sqlc.Meeting, error) {
	return r.Querier.ListMeetings(ctx)
}

func (r *MeetingRepository) ListMeetingsWithDecisionsAndActionItems(ctx context.Context) ([]sqlc.ListMeetingsWithDecisionsAndActionItemsRow, error) {
	return r.Querier.ListMeetingsWithDecisionsAndActionItems(ctx)
}

func (r *MeetingRepository) UpdateMeeting(ctx context.Context, arg sqlc.UpdateMeetingParams) (sqlc.Meeting, error) {
	return r.Querier.UpdateMeeting(ctx, arg)
}

func (r *MeetingRepository) DeleteMeeting(ctx context.Context, id pgtype.UUID) error {
	return r.Querier.DeleteMeeting(ctx, id)
}

func (r *MeetingRepository) CreateMeetingDecision(ctx context.Context, arg sqlc.CreateMeetingDecisionParams) (sqlc.MeetingDecision, error) {
	return r.Querier.CreateMeetingDecision(ctx, arg)
}

func (r *MeetingRepository) GetMeetingDecision(ctx context.Context, id pgtype.UUID) (sqlc.MeetingDecision, error) {
	return r.Querier.GetMeetingDecision(ctx, id)
}

func (r *MeetingRepository) ListMeetingDecisions(ctx context.Context, meetingID pgtype.UUID) ([]sqlc.MeetingDecision, error) {
	return r.Querier.ListMeetingDecisions(ctx, meetingID)
}

func (r *MeetingRepository) DeleteMeetingDecision(ctx context.Context, id pgtype.UUID) error {
	return r.Querier.DeleteMeetingDecision(ctx, id)
}

func (r *MeetingRepository) DeleteMeetingDecisions(ctx context.Context, meetingID pgtype.UUID) error {
	return r.Querier.DeleteMeetingDecisions(ctx, meetingID)
}

func (r *MeetingRepository) CreateMeetingActionItem(ctx context.Context, arg sqlc.CreateMeetingActionItemParams) (sqlc.MeetingActionItem, error) {
	return r.Querier.CreateMeetingActionItem(ctx, arg)
}

func (r *MeetingRepository) GetMeetingActionItem(ctx context.Context, id pgtype.UUID) (sqlc.MeetingActionItem, error) {
	return r.Querier.GetMeetingActionItem(ctx, id)
}

func (r *MeetingRepository) ListMeetingActionItems(ctx context.Context, meetingID pgtype.UUID) ([]sqlc.MeetingActionItem, error) {
	return r.Querier.ListMeetingActionItems(ctx, meetingID)
}

func (r *MeetingRepository) ListIncompleteActionItems(ctx context.Context, meetingID pgtype.UUID) ([]sqlc.MeetingActionItem, error) {
	return r.Querier.ListIncompleteActionItems(ctx, meetingID)
}

func (r *MeetingRepository) UpdateMeetingActionItem(ctx context.Context, arg sqlc.UpdateMeetingActionItemParams) (sqlc.MeetingActionItem, error) {
	return r.Querier.UpdateMeetingActionItem(ctx, arg)
}

func (r *MeetingRepository) MarkActionItemCompleted(ctx context.Context, id pgtype.UUID) (sqlc.MeetingActionItem, error) {
	return r.Querier.MarkActionItemCompleted(ctx, id)
}

func (r *MeetingRepository) MarkActionItemIncomplete(ctx context.Context, id pgtype.UUID) (sqlc.MeetingActionItem, error) {
	return r.Querier.MarkActionItemIncomplete(ctx, id)
}

func (r *MeetingRepository) DeleteMeetingActionItem(ctx context.Context, id pgtype.UUID) error {
	return r.Querier.DeleteMeetingActionItem(ctx, id)
}

func (r *MeetingRepository) DeleteMeetingActionItems(ctx context.Context, meetingID pgtype.UUID) error {
	return r.Querier.DeleteMeetingActionItems(ctx, meetingID)
}