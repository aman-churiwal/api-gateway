package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/redis/go-redis/v9"
)

type TokenBucket struct {
	redis       *storage.RedisClient
	capacity    int // Total Capacity of the bucket
	refillRate  int // Tokens per second
	refillEvery time.Duration
}

type bucketState struct {
	Tokens     float64   `json:"tokens"`
	LastRefill time.Time `json:"last_refill"`
}

func NewTokenBucket(redis *storage.RedisClient, capacity int, refillRate int) *TokenBucket {
	return &TokenBucket{
		redis:       redis,
		capacity:    capacity,
		refillRate:  refillRate,
		refillEvery: time.Second, // taking 1 second as the default refillEvery
	}
}

func (t *TokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("ratelimit:bucket:%s", key)

	data, err := t.redis.Get(ctx, redisKey)
	var state bucketState

	if err == redis.Nil {
		// This is the first request
		// Initialize the bucket
		state = bucketState{
			Tokens:     float64(t.capacity),
			LastRefill: time.Now(),
		}
	} else if err != nil {
		return false, err
	} else {
		json.Unmarshal([]byte(data), &state)
	}

	// Refilling token based on time elapsed
	now := time.Now()
	elapsed := now.Sub(state.LastRefill)
	tokensToAdd := elapsed.Seconds() * float64(t.refillRate)
	state.Tokens = math.Min(state.Tokens+tokensToAdd, float64(t.capacity))
	state.LastRefill = now

	// Consuming One Token for a request
	if state.Tokens >= 1 {
		state.Tokens -= 1

		// Saving the state in Redis
		stateJson, _ := json.Marshal(state)
		t.redis.Set(ctx, redisKey, stateJson, time.Hour)

		return true, nil
	}

	stateJson, _ := json.Marshal(state)
	t.redis.Set(ctx, redisKey, stateJson, time.Hour)

	return false, nil
}

func (t *TokenBucket) Remaining(ctx context.Context, key string) (int, error) {
	redisKey := fmt.Sprintf("ratelimit:bucket:%s", key)

	data, err := t.redis.Get(ctx, redisKey)
	if err == redis.Nil {
		return t.capacity, nil
	}
	if err != nil {
		return 0, err
	}

	var state bucketState
	json.Unmarshal([]byte(data), &state)

	// Calculate current tokens with refill
	now := time.Now()
	elapsed := now.Sub(state.LastRefill)
	tokensToAdd := elapsed.Seconds() * float64(t.refillRate)
	currentTokens := math.Min(state.Tokens+tokensToAdd, float64(t.capacity))

	return int(currentTokens), nil
}

func (t *TokenBucket) Limit() int {
	return t.capacity
}

func (t *TokenBucket) Window() time.Duration {
	// For token bucket, window represents the time to fully refill
	return time.Duration(t.capacity/t.refillRate) * time.Second
}

func (t *TokenBucket) Reset(ctx context.Context, key string) (time.Time, error) {
	redisKey := fmt.Sprintf("ratelimit:bucket:%s", key)

	data, err := t.redis.Get(ctx, redisKey)
	if err == redis.Nil {
		return time.Now(), nil
	}
	if err != nil {
		return time.Time{}, err
	}

	var state bucketState
	json.Unmarshal([]byte(data), &state)

	// Calculate time until bucket is full again
	tokensNeeded := float64(t.capacity) - state.Tokens
	secondsToFull := tokensNeeded / float64(t.refillRate)

	return time.Now().Add(time.Duration(secondsToFull) * time.Second), nil
}
