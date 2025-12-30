package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// BitGoRequestLog represents a log entry for BitGo API requests
type BitGoRequestLog struct {
	ID            string            `json:"id"`
	Timestamp     string            `json:"timestamp"`
	Method        string            `json:"method"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers"`
	Body          interface{}       `json:"body,omitempty"`
	Response      interface{}       `json:"response,omitempty"`
	StatusCode    int               `json:"responseStatus,omitempty"`
	Duration      int64             `json:"duration,omitempty"`
	Error         string            `json:"error,omitempty"`
	CorrelationID string            `json:"correlationId,omitempty"`
}

// BitGoRequestLogger captures and broadcasts BitGo API requests
type BitGoRequestLogger struct {
	clients map[*websocket.Conn]bool
	logs    []BitGoRequestLog
	maxLogs int
}

// NewBitGoRequestLogger creates a new request logger
func NewBitGoRequestLogger() *BitGoRequestLogger {
	return &BitGoRequestLogger{
		clients: make(map[*websocket.Conn]bool),
		logs:    make([]BitGoRequestLog, 0),
		maxLogs: 100, // Keep last 100 requests
	}
}

// LogRequest adds a new request log and broadcasts to connected clients
func (l *BitGoRequestLogger) LogRequest(logEntry BitGoRequestLog) {
	log.Printf("📨 LogRequest called for: %s %s", logEntry.Method, logEntry.URL)

	// Add timestamp if not provided
	if logEntry.Timestamp == "" {
		logEntry.Timestamp = time.Now().Format("15:04:05")
	}

	// Add to logs (keep only last maxLogs)
	l.logs = append(l.logs, logEntry)
	if len(l.logs) > l.maxLogs {
		l.logs = l.logs[1:]
	}

	log.Printf("🔔 Broadcasting to %d WebSocket clients", len(l.clients))

	// Broadcast to all connected clients
	l.broadcast(logEntry)
}

// broadcast sends log entry to all connected WebSocket clients
func (l *BitGoRequestLogger) broadcast(logEntry BitGoRequestLog) {
	message, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}

	// Remove disconnected clients and send to active ones
	for client := range l.clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error broadcasting to client: %v", err)
			client.Close()
			delete(l.clients, client)
		}
	}
}

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from localhost during development
		origin := r.Header.Get("Origin")
		// Allow localhost, 127.0.0.1, and any local development origins
		return origin == "http://localhost:3000" ||
			origin == "https://localhost:3000" ||
			strings.Contains(origin, "localhost") ||
			strings.Contains(origin, "127.0.0.1") ||
			origin == "" // Allow connections without origin (direct curl, etc.)
	},
}

// HandleWebSocket handles WebSocket connections for BitGo request logs
func (s *Server) HandleBitGoRequestLogs(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}
	defer conn.Close()

	// Add client to logger
	if s.bitgoRequestLogger == nil {
		s.bitgoRequestLogger = NewBitGoRequestLogger()
	}

	s.bitgoRequestLogger.clients[conn] = true
	log.Printf("New WebSocket client connected for BitGo request logs")

	// Send existing logs to new client
	for _, logEntry := range s.bitgoRequestLogger.logs {
		message, err := json.Marshal(logEntry)
		if err != nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Failed to send existing log to client: %v", err)
			delete(s.bitgoRequestLogger.clients, conn)
			return
		}
	}

	// Set up ping handler to keep connection alive
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	// Start a goroutine to send periodic pings
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to ping WebSocket client: %v", err)
				delete(s.bitgoRequestLogger.clients, conn)
				return
			}
		}
	}()

	// Keep connection alive by reading control messages
	for {
		messageType, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket client disconnected unexpectedly: %v", err)
			} else {
				log.Printf("WebSocket client disconnected: %v", err)
			}
			delete(s.bitgoRequestLogger.clients, conn)
			break
		}

		// Handle ping/pong messages, ignore text messages (we don't expect any data from client)
		if messageType == websocket.PingMessage {
			if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				log.Printf("Failed to send pong: %v", err)
				delete(s.bitgoRequestLogger.clients, conn)
				break
			}
		}
	}
}

// ObscureToken masks authentication token for security
func obscureToken(token string) string {
	if len(token) <= 10 {
		return "***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}

// ConvertToCURL generates a CURL command from the request log
func (log BitGoRequestLog) ToCURL() string {
	curl := fmt.Sprintf("curl -X %s \"%s\"", log.Method, log.URL)

	for key, value := range log.Headers {
		if key == "Authorization" && strings.HasPrefix(value, "Bearer ") {
			token := strings.TrimPrefix(value, "Bearer ")
			value = "Bearer " + obscureToken(token)
		}
		curl += fmt.Sprintf(" \\\\\n  -H \"%s: %s\"", key, value)
	}

	if log.Body != nil {
		bodyBytes, _ := json.Marshal(log.Body)
		curl += fmt.Sprintf(" \\\\\n  -d '%s'", string(bodyBytes))
	}

	return curl
}
