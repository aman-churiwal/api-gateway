package middleware

import (
	"time"

	"github.com/aman-churiwal/api-gateway/internal/models"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Buffered channel for async logging
var logChannel chan models.RequestLog

// Initializes the request logger
func InitRequestLogger(db *storage.Postgres, bufferSize int) {
	logChannel = make(chan models.RequestLog, bufferSize)

	// Start background worker to batch insert logs
	go func() {
		batch := make([]models.RequestLog, 0, 100)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case log := <-logChannel:
				batch = append(batch, log)

				// Insert when batch is full
				if len(batch) >= 100 {
					insertBatch(db, batch)
					batch = make([]models.RequestLog, 0, 100)
				}
			case <-ticker.C:
				// Periodically insert remaining logs
				if len(batch) > 0 {
					insertBatch(db, batch)
					batch = make([]models.RequestLog, 0, 100)
				}
			}
		}
	}()
}

// Inserts a batch of logs into the database
func insertBatch(db *storage.Postgres, logs []models.RequestLog) {
	if len(logs) == 0 {
		return
	}

	if err := db.DB.Create(&logs).Error; err != nil {
		// Log error but dont block
		println("Failed to insert request logs:", err.Error())
	}
}

// Logs all HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Extract API key ID if present
		var apiKeyID *uuid.UUID
		if apiKeyInterface, exists := c.Get("api_key_id"); exists {
			if id, ok := apiKeyInterface.(uuid.UUID); ok {
				apiKeyID = &id
			}
		}

		// Extract backend server if present
		backendServer := c.GetHeader("X-Backend-Server")

		// Create log entry
		logEntry := models.RequestLog{
			Timestamp:      start,
			APIKeyID:       apiKeyID,
			Method:         c.Request.Method,
			Path:           c.Request.URL.Path,
			StatusCode:     c.Writer.Status(),
			ResponseTimeMs: int(duration.Milliseconds()),
			IPAddress:      c.ClientIP(),
			UserAgent:      c.Request.UserAgent(),
			BackendServer:  backendServer,
		}

		// Send to channel for async processing
		select {
		case logChannel <- logEntry:
			// Successfully queued
		default:
			// Channel full, skip logging to avoid blocking
			println("Request log channel full, skipping log entry")
		}
	}
}
