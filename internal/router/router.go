package router

import (
	"net/http"

	"example.com/internal/handlers"
	"github.com/gin-gonic/gin"
)

func New(recordingHandler *handlers.RecordingHandler) *gin.Engine {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	api.POST("/recordings", recordingHandler.Create)

	return router
}
