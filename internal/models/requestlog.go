package models

import (
	"time"

	"github.com/google/uuid"
)

// Represents a logged API request
type RequestLog struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	Timestamp      time.Time  `gorm:"index" json:"timestamp"`
	APIKeyID       *uuid.UUID `gorm:"index" json:"api_key_id,omitempty"`
	Method         string     `json:"method"`
	Path           string     `gorm:"index" json:"path"`
	StatusCode     int        `gorm:"index" json:"status_code"`
	ResponseTimeMs int        `json:"response_time_ms"`
	IPAddress      string     `json:"ip_address"`
	UserAgent      string     `json:"user_agent"`
	BackendServer  string     `json:"backend_server,omitempty"`
}

func (RequestLog) TableName() string {
	return "request_logs"
}
