package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/redis/go-redis/v9"
)

type SlidingWindowLimiter struct {
	redis  *storage.RedisClient
	limit  int
	window time.Duration
}

func NewSlidingWindowLimiter(redis *storage.RedisClient, limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		redis:  redis,
		limit:  limit,
		window: window,
	}
}

func (s *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("ratelimit:sliding:%s", key)
	now := time.Now()

	windowStart := now.Add(-s.window)

	// Using Redis sorted set with timestamps as scores
	pipe := s.redis.Pipeline()

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count requests in current window
	countCmd := pipe.ZCard(ctx, redisKey)

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	count := countCmd.Val()

	if count < int64(s.limit) {
		// Add current request
		s.redis.ZAdd(ctx, redisKey, redis.Z{
			Score:  float64(now.UnixNano()),
			Member: fmt.Sprintf("%d", now.UnixNano()),
		})
		s.redis.Expire(ctx, redisKey, s.window)
		return true, nil
	}

	return false, nil
}

func (s *SlidingWindowLimiter) Remaining(ctx context.Context, key string) (int, error) {
	redisKey := fmt.Sprintf("ratelimit:sliding:%s", key)
	now := time.Now()
	windowStart := now.Add(-s.window)

	// Count requests in current window
	count, err := s.redis.ZCount(ctx, redisKey, fmt.Sprintf("%d", windowStart.UnixNano()), fmt.Sprintf("%d", now.UnixNano()))
	if err != nil {
		return 0, err
	}

	remaining := s.limit - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (s *SlidingWindowLimiter) Limit() int {
	return s.limit
}

func (s *SlidingWindowLimiter) Window() time.Duration {
	return s.window
}

func (s *SlidingWindowLimiter) Reset(ctx context.Context, key string) (time.Time, error) {
	redisKey := fmt.Sprintf("ratelimit:sliding:%s", key)

	// Get the oldest entry in the sorted set
	oldest, err := s.redis.ZRange(ctx, redisKey, 0, 0)
	if err != nil || len(oldest) == 0 {
		// No entries, window resets now
		return time.Now(), nil
	}

	// Parse the oldest timestamp
	var oldestNano int64
	fmt.Sscanf(oldest[0], "%d", &oldestNano)

	// Reset time is when the oldest entry expires (oldest + window)
	resetTime := time.Unix(0, oldestNano).Add(s.window)
	return resetTime, nil
}
