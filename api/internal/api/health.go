package api

import (
	"context"
	"net/http"
	"time"

	"bitgo-wallets-api/internal/bitgo"

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

// DetailedHealthResponse includes background service status
type DetailedHealthResponse struct {
	Status         string                 `json:"status"`
	Timestamp      time.Time              `json:"timestamp"`
	Version        string                 `json:"version"`
	Database       string                 `json:"database"`
	BackgroundJobs map[string]interface{} `json:"backgroundJobs"`
	Notifications  map[string]interface{} `json:"notifications"`
}

func (s *Server) detailedHealthCheck(c *gin.Context) {
	// Check database connection
	dbStatus := "ok"
	if err := s.db.Ping(); err != nil {
		dbStatus = "error"
	}

	// Get background job status
	pollingWorkerHealth := s.pollingWorker.HealthCheck()

	response := DetailedHealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Database:  dbStatus,
		BackgroundJobs: map[string]interface{}{
			"pollingWorker": pollingWorkerHealth,
		},
		Notifications: map[string]interface{}{
			"service": "running",
		},
	}

	statusCode := http.StatusOK
	if dbStatus == "error" || pollingWorkerHealth["status"] != "running" {
		response.Status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// testBitGo makes a simple BitGo API call to test request logging
func (s *Server) testBitGo(c *gin.Context) {
	ctx := context.Background()

	// Make a simple BitGo API call - this should trigger request logging
	wallets, err := s.bitgoClient.ListWallets(ctx, bitgo.WalletListOptions{
		Coin:  "tbtc",
		Limit: 1,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "BitGo test call completed (with expected error)",
			"error":   err.Error(),
			"note":    "Check the Requests tab to see if the API call was logged",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "BitGo test call succeeded",
		"wallets": len(wallets.Wallets),
		"note":    "Check the Requests tab to see the logged API call",
	})
}
