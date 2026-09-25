package router

import (
	"net/http"

	"example.com/internal/db/sqlc"
	"example.com/internal/handlers"
	"example.com/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(
	queries *sqlc.Queries,
	authHandler *handlers.AuthHandler,
	recordingHandler *handlers.RecordingHandler,
	meetingHandler *handlers.MeetingHandler,
	transcriptHandler *handlers.TranscriptHandler,
) *gin.Engine {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5174", "http://wails.localhost:34115"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")

	// public auth routes
	api.POST("/auth/setup", authHandler.Setup)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/invitations/accept", authHandler.AcceptInvitation)

	// authenticated routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(queries))

	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/auth/me", authHandler.Me)

	// admin/owner auth routes
	admin := protected.Group("")
	admin.Use(middleware.RequireRoles("owner", "admin"))

	admin.POST("/auth/invitations", authHandler.CreateInvitation)
	admin.GET("/auth/organization/members", authHandler.ListOrganizationMembers)
	admin.GET("/auth/users/:id", authHandler.GetUser)
	admin.PUT("/auth/users/:id", authHandler.UpdateUser)
	admin.PATCH("/auth/users/:id/role", authHandler.UpdateUserRole)

	// recording routes
	protected.POST("/recordings", recordingHandler.Create)
	protected.POST("/recordings/:id/generate-ai-results", recordingHandler.GenerateAIResults)

	// meeting routes
	protected.POST("/meetings", meetingHandler.CreateMeeting)
	protected.GET("/meetings", meetingHandler.ListMeetings)
	protected.GET("/meetings/:id", meetingHandler.GetMeeting)
	protected.GET("/meetings/:id/audio", meetingHandler.ServeMeetingAudio)
	protected.PUT("/meetings/:id", meetingHandler.UpdateMeeting)
	protected.DELETE("/meetings/:id", meetingHandler.DeleteMeeting)

	protected.POST("/meetings/:id/decisions", meetingHandler.CreateMeetingDecision)
	protected.GET("/meetings/:id/decisions", meetingHandler.ListMeetingDecisions)
	protected.DELETE("/meetings/:id/decisions", meetingHandler.DeleteMeetingDecisions)
	protected.GET("/decisions/:id", meetingHandler.GetMeetingDecision)
	protected.DELETE("/decisions/:id", meetingHandler.DeleteMeetingDecision)

	protected.POST("/meetings/:id/action-items", meetingHandler.CreateMeetingActionItem)
	protected.GET("/meetings/:id/action-items", meetingHandler.ListMeetingActionItems)
	protected.GET("/meetings/:id/action-items/incomplete", meetingHandler.ListIncompleteActionItems)
	protected.DELETE("/meetings/:id/action-items", meetingHandler.DeleteMeetingActionItems)
	protected.GET("/action-items/:id", meetingHandler.GetMeetingActionItem)
	protected.PUT("/action-items/:id", meetingHandler.UpdateMeetingActionItem)
	protected.PATCH("/action-items/:id/complete", meetingHandler.MarkActionItemCompleted)
	protected.PATCH("/action-items/:id/incomplete", meetingHandler.MarkActionItemIncomplete)
	protected.DELETE("/action-items/:id", meetingHandler.DeleteMeetingActionItem)

	// transcript routes
	protected.POST("/meetings/:id/transcript-segments", transcriptHandler.CreateTranscriptSegment)
	protected.GET("/meetings/:id/transcript-segments", transcriptHandler.ListTranscriptSegments)
	protected.DELETE("/meetings/:id/transcript-segments", transcriptHandler.DeleteTranscriptSegmentsByMeeting)
	protected.PUT("/meetings/:id/transcript-segments/update-speakers", transcriptHandler.UpdateSpeakers)
	protected.GET("/transcript-segments/:id", transcriptHandler.GetTranscriptSegment)
	protected.PUT("/transcript-segments/:id", transcriptHandler.UpdateTranscriptSegment)
	protected.DELETE("/transcript-segments/:id", transcriptHandler.DeleteTranscriptSegment)

	return router
}
