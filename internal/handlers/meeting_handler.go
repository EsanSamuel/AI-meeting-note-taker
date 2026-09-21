package handlers

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"example.com/internal/repository"
	services "example.com/internal/services/db"
)

type MeetingHandler struct {
	svc *services.MeetingService
}

func NewMeetingHandler(svc *services.MeetingService) *MeetingHandler {
	return &MeetingHandler{svc: svc}
}

// ---- Meetings ----

type createMeetingRequest struct {
	Title           string    `json:"title" binding:"required"`
	StartedAt       time.Time `json:"started_at" binding:"required"`
	EndedAt         time.Time `json:"ended_at" binding:"required"`
	DurationSeconds float64   `json:"duration_seconds"`
	AudioPath       string    `json:"audio_path"`
	VideoPath       string    `json:"video_path"`
}

func (h *MeetingHandler) CreateMeeting(c *gin.Context) {
	var req createMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	meeting, err := h.svc.CreateMeeting(c.Request.Context(), repository.Meeting{
		Title:           req.Title,
		StartedAt:       req.StartedAt,
		EndedAt:         req.EndedAt,
		DurationSeconds: req.DurationSeconds,
		AudioPath:       req.AudioPath,
		VideoPath:       req.VideoPath,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, meeting)
}

func (h *MeetingHandler) GetMeeting(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	meeting, err := h.svc.GetMeeting(c.Request.Context(), id)
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, meeting)
}

func (h *MeetingHandler) ServeMeetingAudio(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	meeting, err := h.svc.GetMeeting(c.Request.Context(), id)
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	if meeting.AudioPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "meeting audio not found"})
		return
	}
	if _, err := os.Stat(meeting.AudioPath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "meeting audio file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.File(meeting.AudioPath)
}

func (h *MeetingHandler) ListMeetings(c *gin.Context) {
	meetings, err := h.svc.ListMeetings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, meetings)
}

type updateMeetingRequest struct {
	Title           string    `json:"title" binding:"required"`
	StartedAt       time.Time `json:"started_at" binding:"required"`
	EndedAt         time.Time `json:"ended_at" binding:"required"`
	DurationSeconds float64   `json:"duration_seconds"`
	AudioPath       string    `json:"audio_path"`
	VideoPath       string    `json:"video_path"`
}

func (h *MeetingHandler) UpdateMeeting(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	var req updateMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	meeting, err := h.svc.UpdateMeeting(c.Request.Context(), repository.Meeting{
		ID:              id,
		Title:           req.Title,
		StartedAt:       req.StartedAt,
		EndedAt:         req.EndedAt,
		DurationSeconds: req.DurationSeconds,
		AudioPath:       req.AudioPath,
		VideoPath:       req.VideoPath,
	})
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, meeting)
}

func (h *MeetingHandler) DeleteMeeting(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteMeeting(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- Decisions ----

type createMeetingDecisionRequest struct {
	Decision         string  `json:"decision" binding:"required"`
	TimestampSeconds float64 `json:"timestamp_seconds"`
}

func (h *MeetingHandler) CreateMeetingDecision(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	var req createMeetingDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	decision, err := h.svc.CreateMeetingDecision(c.Request.Context(), repository.MeetingDecision{
		MeetingID:        meetingID,
		Decision:         req.Decision,
		TimestampSeconds: req.TimestampSeconds,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, decision)
}

func (h *MeetingHandler) GetMeetingDecision(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	decision, err := h.svc.GetMeetingDecision(c.Request.Context(), id)
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, decision)
}

func (h *MeetingHandler) ListMeetingDecisions(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	decisions, err := h.svc.ListMeetingDecisions(c.Request.Context(), meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, decisions)
}

func (h *MeetingHandler) DeleteMeetingDecision(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteMeetingDecision(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *MeetingHandler) DeleteMeetingDecisions(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteMeetingDecisions(c.Request.Context(), meetingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- Action items ----

type createMeetingActionItemRequest struct {
	Task             string  `json:"task" binding:"required"`
	Assignee         string  `json:"assignee"`
	TimestampSeconds float64 `json:"timestamp_seconds"`
}

func (h *MeetingHandler) CreateMeetingActionItem(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	var req createMeetingActionItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.svc.CreateMeetingActionItem(c.Request.Context(), repository.MeetingActionItem{
		MeetingID:        meetingID,
		Task:             req.Task,
		Assignee:         req.Assignee,
		TimestampSeconds: req.TimestampSeconds,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *MeetingHandler) GetMeetingActionItem(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	item, err := h.svc.GetMeetingActionItem(c.Request.Context(), id)
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *MeetingHandler) ListMeetingActionItems(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	items, err := h.svc.ListMeetingActionItems(c.Request.Context(), meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *MeetingHandler) ListIncompleteActionItems(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	items, err := h.svc.ListIncompleteActionItems(c.Request.Context(), meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

type updateMeetingActionItemRequest struct {
	Task             string  `json:"task" binding:"required"`
	Assignee         string  `json:"assignee"`
	TimestampSeconds float64 `json:"timestamp_seconds"`
	Completed        bool    `json:"completed"`
}

func (h *MeetingHandler) UpdateMeetingActionItem(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	var req updateMeetingActionItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.svc.UpdateMeetingActionItem(c.Request.Context(), repository.MeetingActionItem{
		ID:               id,
		Task:             req.Task,
		Assignee:         req.Assignee,
		TimestampSeconds: req.TimestampSeconds,
		Completed:        req.Completed,
	})
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *MeetingHandler) MarkActionItemCompleted(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	item, err := h.svc.MarkActionItemCompleted(c.Request.Context(), id)
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *MeetingHandler) MarkActionItemIncomplete(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	item, err := h.svc.MarkActionItemIncomplete(c.Request.Context(), id)
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *MeetingHandler) DeleteMeetingActionItem(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteMeetingActionItem(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *MeetingHandler) DeleteMeetingActionItems(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteMeetingActionItems(c.Request.Context(), meetingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- helpers ----

func parseUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + param})
		return uuid.Nil, false
	}
	return id, true
}

func respondNotFoundOrError(c *gin.Context, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
