package ratelimit

import (
	"time"

	"github.com/aman-churiwal/api-gateway/internal/storage"
)

func NewLimiter(redis *storage.RedisClient, algorithm string, limit int, window time.Duration) Limiter {
	switch algorithm {
	case "token_bucket":
		refillRate := limit / int(window.Seconds())
		if refillRate == 0 {
			refillRate = 1
		}
		return NewTokenBucket(redis, limit, refillRate)
	case "fixed_window":
		return NewFixedWindow(redis, limit, window)
	default:
		return NewFixedWindow(redis, limit, window)
	}
}
