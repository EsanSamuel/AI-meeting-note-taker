package router

import (
	"net/http"

	"example.com/internal/handlers"
	"github.com/gin-gonic/gin"
)

func New(recordingHandler *handlers.RecordingHandler, meetingHandler *handlers.MeetingHandler, transcriptHandler *handlers.TranscriptHandler) *gin.Engine {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")

	// recording routes
	api.POST("/recordings", recordingHandler.Create)

	// meeting routes
	api.POST("/meetings", meetingHandler.CreateMeeting)
	api.GET("/meetings", meetingHandler.ListMeetings)
	api.GET("/meetings/:id", meetingHandler.GetMeeting)
	api.PUT("/meetings/:id", meetingHandler.UpdateMeeting)
	api.DELETE("/meetings/:id", meetingHandler.DeleteMeeting)

	api.POST("/meetings/:id/decisions", meetingHandler.CreateMeetingDecision)
	api.GET("/meetings/:id/decisions", meetingHandler.ListMeetingDecisions)
	api.DELETE("/meetings/:id/decisions", meetingHandler.DeleteMeetingDecisions)
	api.GET("/decisions/:id", meetingHandler.GetMeetingDecision)
	api.DELETE("/decisions/:id", meetingHandler.DeleteMeetingDecision)

	api.POST("/meetings/:id/action-items", meetingHandler.CreateMeetingActionItem)
	api.GET("/meetings/:id/action-items", meetingHandler.ListMeetingActionItems)
	api.GET("/meetings/:id/action-items/incomplete", meetingHandler.ListIncompleteActionItems)
	api.DELETE("/meetings/:id/action-items", meetingHandler.DeleteMeetingActionItems)
	api.GET("/action-items/:id", meetingHandler.GetMeetingActionItem)
	api.PUT("/action-items/:id", meetingHandler.UpdateMeetingActionItem)
	api.PATCH("/action-items/:id/complete", meetingHandler.MarkActionItemCompleted)
	api.PATCH("/action-items/:id/incomplete", meetingHandler.MarkActionItemIncomplete)
	api.DELETE("/action-items/:id", meetingHandler.DeleteMeetingActionItem)

	// transcript routes
	api.POST("/meetings/:id/transcript-segments", transcriptHandler.CreateTranscriptSegment)
	api.GET("/meetings/:id/transcript-segments", transcriptHandler.ListTranscriptSegments)
	api.DELETE("/meetings/:id/transcript-segments", transcriptHandler.DeleteTranscriptSegmentsByMeeting)
	api.GET("/transcript-segments/:id", transcriptHandler.GetTranscriptSegment)
	api.PUT("/transcript-segments/:id", transcriptHandler.UpdateTranscriptSegment)
	api.DELETE("/transcript-segments/:id", transcriptHandler.DeleteTranscriptSegment)

	return router
}