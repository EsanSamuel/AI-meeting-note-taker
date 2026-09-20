package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/internal/repository"
	transcriptService "example.com/internal/services"
	services "example.com/internal/services/db"
)

type UpdateSpeakersRequest struct {
	Speakers map[string]string `json:"speakers"`
}

type TranscriptHandler struct {
	svc           *services.TranscriptService
	transcriptSvc transcriptService.TranscribeService
}

func NewTranscriptHandler(svc *services.TranscriptService, transcriptSvc transcriptService.TranscribeService) *TranscriptHandler {
	return &TranscriptHandler{svc: svc, transcriptSvc: transcriptSvc}
}

type createTranscriptSegmentRequest struct {
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
	SpeakerID string  `json:"speaker_id"`
	Speaker   string  `json:"speaker"`
	Text      string  `json:"text" binding:"required"`
}

func (h *TranscriptHandler) CreateTranscriptSegment(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	var req createTranscriptSegmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	segment, err := h.svc.CreateTranscriptSegment(c.Request.Context(), repository.TranscriptSegment{
		MeetingID: meetingID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		SpeakerID: req.SpeakerID,
		Speaker:   req.Speaker,
		Text:      req.Text,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, segment)
}

func (h *TranscriptHandler) GetTranscriptSegment(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	segment, err := h.svc.GetTranscriptSegment(c.Request.Context(), id)
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, segment)
}

func (h *TranscriptHandler) ListTranscriptSegments(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	segments, err := h.svc.ListTranscriptSegments(c.Request.Context(), meetingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, segments)
}

type updateTranscriptSegmentRequest struct {
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
	Speaker   string  `json:"speaker"`
	Text      string  `json:"text" binding:"required"`
}

func (h *TranscriptHandler) UpdateTranscriptSegment(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	var req updateTranscriptSegmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	segment, err := h.svc.UpdateTranscriptSegment(c.Request.Context(), repository.TranscriptSegment{
		ID:        id,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Speaker:   req.Speaker,
		Text:      req.Text,
	})
	if err != nil {
		respondNotFoundOrError(c, err)
		return
	}
	c.JSON(http.StatusOK, segment)
}

func (h *TranscriptHandler) DeleteTranscriptSegment(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteTranscriptSegment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TranscriptHandler) DeleteTranscriptSegmentsByMeeting(c *gin.Context) {
	meetingID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteTranscriptSegmentsByMeeting(c.Request.Context(), meetingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TranscriptHandler) UpdateSpeakers(c *gin.Context) {
	meetingID, ok := parseUUID(c, "meeting_id")
	if !ok {
		return
	}

	var req UpdateSpeakersRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if len(req.Speakers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "speakers cannot be empty",
		})
		return
	}

	err := h.transcriptSvc.UpdateSpeakers(
		c.Request.Context(),
		meetingID,
		req.Speakers,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "speakers updated successfully",
	})
}
