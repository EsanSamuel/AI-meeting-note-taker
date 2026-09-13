package services

import (
	"context"

	"github.com/google/uuid"

	"example.com/internal/repository"
)

type MeetingService struct {
	repo *repository.MeetingRepository
}

func NewMeetingService(repo *repository.MeetingRepository) *MeetingService {
	return &MeetingService{repo: repo}
}

// ---- Meetings ----

func (s *MeetingService) CreateMeeting(ctx context.Context, m repository.Meeting) (repository.Meeting, error) {
	return s.repo.CreateMeeting(ctx, m)
}

func (s *MeetingService) GetMeeting(ctx context.Context, id uuid.UUID) (repository.Meeting, error) {
	return s.repo.GetMeeting(ctx, id)
}

func (s *MeetingService) ListMeetings(ctx context.Context) ([]repository.Meeting, error) {
	return s.repo.ListMeetings(ctx)
}

func (s *MeetingService) UpdateMeeting(ctx context.Context, m repository.Meeting) (repository.Meeting, error) {
	return s.repo.UpdateMeeting(ctx, m)
}

func (s *MeetingService) DeleteMeeting(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteMeeting(ctx, id)
}

// ---- Decisions ----

func (s *MeetingService) CreateMeetingDecision(ctx context.Context, d repository.MeetingDecision) (repository.MeetingDecision, error) {
	return s.repo.CreateMeetingDecision(ctx, d)
}

func (s *MeetingService) GetMeetingDecision(ctx context.Context, id uuid.UUID) (repository.MeetingDecision, error) {
	return s.repo.GetMeetingDecision(ctx, id)
}

func (s *MeetingService) ListMeetingDecisions(ctx context.Context, meetingID uuid.UUID) ([]repository.MeetingDecision, error) {
	return s.repo.ListMeetingDecisions(ctx, meetingID)
}

func (s *MeetingService) DeleteMeetingDecision(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteMeetingDecision(ctx, id)
}

func (s *MeetingService) DeleteMeetingDecisions(ctx context.Context, meetingID uuid.UUID) error {
	return s.repo.DeleteMeetingDecisions(ctx, meetingID)
}

// ---- Action items ----

func (s *MeetingService) CreateMeetingActionItem(ctx context.Context, a repository.MeetingActionItem) (repository.MeetingActionItem, error) {
	return s.repo.CreateMeetingActionItem(ctx, a)
}

func (s *MeetingService) GetMeetingActionItem(ctx context.Context, id uuid.UUID) (repository.MeetingActionItem, error) {
	return s.repo.GetMeetingActionItem(ctx, id)
}

func (s *MeetingService) ListMeetingActionItems(ctx context.Context, meetingID uuid.UUID) ([]repository.MeetingActionItem, error) {
	return s.repo.ListMeetingActionItems(ctx, meetingID)
}

func (s *MeetingService) ListIncompleteActionItems(ctx context.Context, meetingID uuid.UUID) ([]repository.MeetingActionItem, error) {
	return s.repo.ListIncompleteActionItems(ctx, meetingID)
}

func (s *MeetingService) UpdateMeetingActionItem(ctx context.Context, a repository.MeetingActionItem) (repository.MeetingActionItem, error) {
	return s.repo.UpdateMeetingActionItem(ctx, a)
}

func (s *MeetingService) MarkActionItemCompleted(ctx context.Context, id uuid.UUID) (repository.MeetingActionItem, error) {
	return s.repo.MarkActionItemCompleted(ctx, id)
}

func (s *MeetingService) MarkActionItemIncomplete(ctx context.Context, id uuid.UUID) (repository.MeetingActionItem, error) {
	return s.repo.MarkActionItemIncomplete(ctx, id)
}

func (s *MeetingService) DeleteMeetingActionItem(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteMeetingActionItem(ctx, id)
}

func (s *MeetingService) DeleteMeetingActionItems(ctx context.Context, meetingID uuid.UUID) error {
	return s.repo.DeleteMeetingActionItems(ctx, meetingID)
}
