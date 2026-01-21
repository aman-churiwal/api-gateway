package models

type RateLimitTier struct {
	Name              string `gorm:"primaryKey" json:"name"`
	RequestsPerMinute int    `gorm:"not null" json:"requests_per_minute"`
	RequestsPerHour   int    `gorm:"not null" json:"requests_per_hour"`
	Algorithm         string `gorm:"not null" json:"algorithm"` // "fixed_window" "token_bucket" "sliding_window"
}

func (RateLimitTier) TableName() string {
	return "rate_limit_tiers"
}
