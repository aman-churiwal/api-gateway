package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/redis/go-redis/v9"
)

type FixedWindowLimiter struct {
	redis  *storage.RedisClient
	limit  int
	window time.Duration
}

func NewFixedWindow(redis *storage.RedisClient, limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		redis:  redis,
		limit:  limit,
		window: window, // Window of time duration
	}
}

func (f *FixedWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	currentWindow := time.Now().Unix() / int64(f.window.Seconds())
	redisKey := fmt.Sprintf("ratelimit:fixed:%s:%d", key, currentWindow)

	count, err := f.redis.Incr(ctx, redisKey)
	if err != nil {
		return false, err
	}

	if count == 1 {
		f.redis.Expire(ctx, redisKey, f.window)
	}

	return count <= int64(f.limit), nil
}

func (f *FixedWindowLimiter) Remaining(ctx context.Context, key string) (int, error) {
	currentWindow := time.Now().Unix() / int64(f.window.Seconds())
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, currentWindow)

	val, err := f.redis.Get(ctx, redisKey)
	if err == redis.Nil {
		return f.limit, nil
	}

	if err != nil {
		return 0, err
	}

	count, _ := strconv.Atoi(val)
	remaining := f.limit - count

	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

func (f *FixedWindowLimiter) Limit() int {
	return f.limit
}

func (f *FixedWindowLimiter) Window() time.Duration {
	return f.window
}

// Returns the time at which the limit resets
func (f *FixedWindowLimiter) Reset(ctx context.Context, key string) (time.Time, error) {
	currentWindow := time.Now().Unix() / int64(f.window.Seconds())
	nextWindow := (currentWindow + 1) * int64(f.window.Seconds())
	return time.Unix(nextWindow, 0), nil
}
