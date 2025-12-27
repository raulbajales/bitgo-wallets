package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Database  string    `json:"database"`
}

func (s *Server) healthCheck(c *gin.Context) {
	// Check database connection
	dbStatus := "ok"
	if err := s.db.Ping(); err != nil {
		dbStatus = "error"
	}

	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Database:  dbStatus,
	}

	statusCode := http.StatusOK
	if dbStatus == "error" {
		response.Status = "error"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}
