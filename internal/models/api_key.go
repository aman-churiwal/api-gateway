package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIKey struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	KeyHash    string     `gorm:"uniqueIndex;not null" json:"-"`
	Name       string     `gorm:"not null" json:"name"`
	CreatedBy  string     `json:"created_by"`
	Tier       string     `gorm:"default:'basic'" json:"tier"`
	IsActive   bool       `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

func (a *APIKey) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (APIKey) TableName() string {
	return "api_keys"
}
