package api

import (
	"log"
	"time"

	"github.com/google/uuid"
)

// BitGoLogger implements bitgo.Logger and captures requests for the debug console
type BitGoLogger struct {
	requestLogger *BitGoRequestLogger
	currentReq    map[string]*BitGoRequestLog // Track ongoing requests by correlation ID
}

// NewBitGoLogger creates a logger that captures BitGo API requests
func NewBitGoLogger(requestLogger *BitGoRequestLogger) *BitGoLogger {
	log.Printf("🔧 Creating BitGo custom logger for request capture")
	return &BitGoLogger{
		requestLogger: requestLogger,
		currentReq:    make(map[string]*BitGoRequestLog),
	}
}

// Info logs info messages and captures BitGo API requests
func (l *BitGoLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)

	// Add more explicit logging to debug the issue
	log.Printf("🔍 DEBUG: BitGoLogger.Info called with msg='%s'", msg)
	log.Printf("🔍 DEBUG: requestLogger is nil: %v", l.requestLogger == nil)

	if msg == "Making BitGo API request" {
		log.Printf("🚀 Capturing BitGo API request start")
		l.handleRequestStart(fields...)
	} else if msg == "BitGo API response" {
		log.Printf("📨 Capturing BitGo API response")
		l.handleRequestResponse(fields...)
	} else {
		log.Printf("🔍 DEBUG: Message '%s' didn't match expected patterns", msg)
	}
}

// Warn logs warning messages
func (l *BitGoLogger) Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] %s %v", msg, fields)
}

// Error logs error messages and captures API errors
func (l *BitGoLogger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, fields)
	l.handleRequestError(msg, fields...)
}

// Debug logs debug messages
func (l *BitGoLogger) Debug(msg string, fields ...interface{}) {
	log.Printf("[DEBUG] %s %v", msg, fields)
}

// handleRequestStart captures the start of a BitGo API request
func (l *BitGoLogger) handleRequestStart(fields ...interface{}) {
	log.Printf("🚀 handleRequestStart called with %d fields: %+v", len(fields), fields)

	if l.requestLogger == nil {
		log.Printf("⚠️ requestLogger is nil!")
		return
	}

	var (
		method        string
		url           string
		correlationID string
		body          interface{}
	)

	// Parse fields (they come in key-value pairs)
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}

		switch key {
		case "method":
			if v, ok := fields[i+1].(string); ok {
				method = v
			}
		case "url":
			if v, ok := fields[i+1].(string); ok {
				url = v
			}
		case "correlation_id":
			if v, ok := fields[i+1].(string); ok {
				correlationID = v
			}
		case "body":
			body = fields[i+1]
		}
	}

	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	// Create BitGo request log entry
	logEntry := &BitGoRequestLog{
		ID:            correlationID,
		Timestamp:     time.Now().Format("15:04:05"),
		Method:        method,
		URL:           url,
		Headers:       l.createHeaders(),
		Body:          body,
		CorrelationID: correlationID,
	}

	log.Printf("📋 Created BitGo request log: %s %s (ID: %s)", method, url, correlationID)

	// Store for correlation with response
	l.currentReq[correlationID] = logEntry
}

// handleRequestResponse captures BitGo API response
func (l *BitGoLogger) handleRequestResponse(fields ...interface{}) {
	if l.requestLogger == nil {
		return
	}

	var (
		statusCode    int
		correlationID string
	)

	// Parse fields
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}

		switch key {
		case "status_code":
			if v, ok := fields[i+1].(int); ok {
				statusCode = v
			}
		case "correlation_id":
			if v, ok := fields[i+1].(string); ok {
				correlationID = v
			}
		}
	}

	// Find the original request
	if req, exists := l.currentReq[correlationID]; exists {
		req.StatusCode = statusCode
		req.Duration = time.Since(parseTime(req.Timestamp)).Milliseconds()

		log.Printf("✅ Completing BitGo request log: %s (Status: %d)", correlationID, statusCode)

		// Log the complete request
		l.requestLogger.LogRequest(*req)

		// Clean up
		delete(l.currentReq, correlationID)
	} else {
		log.Printf("⚠️ No matching request found for correlation ID: %s", correlationID)
	}
}

// handleRequestError captures BitGo API errors
func (l *BitGoLogger) handleRequestError(msg string, fields ...interface{}) {
	if l.requestLogger == nil {
		return
	}

	var correlationID string

	// Try to extract correlation ID from fields
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok && key == "correlation_id" {
			if v, ok := fields[i+1].(string); ok {
				correlationID = v
			}
		}
	}

	// Find the original request and add error
	if req, exists := l.currentReq[correlationID]; exists {
		req.Error = msg
		req.Duration = time.Since(parseTime(req.Timestamp)).Milliseconds()

		// Log the complete request with error
		l.requestLogger.LogRequest(*req)

		// Clean up
		delete(l.currentReq, correlationID)
	} else if correlationID == "" {
		// If no correlation ID, create a standalone error log
		logEntry := BitGoRequestLog{
			ID:        uuid.New().String(),
			Timestamp: time.Now().Format("15:04:05"),
			Error:     msg,
		}
		l.requestLogger.LogRequest(logEntry)
	}
}

// createHeaders creates the headers map for BitGo requests
func (l *BitGoLogger) createHeaders() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"User-Agent":    "bitgo-wallets-api/1.0",
		"Authorization": "Bearer [TOKEN]", // Will be obscured in display
	}
}

// parseTime parses timestamp back to time for duration calculation
func parseTime(timestamp string) time.Time {
	now := time.Now()
	t, err := time.Parse("15:04:05", timestamp)
	if err != nil {
		return now
	}

	// Adjust to today's date
	return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location())
}
