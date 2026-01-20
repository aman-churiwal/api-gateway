package ratelimit

import (
	"context"
	"time"
)

type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)

	Remaining(ctx context.Context, key string) (int, error)

	Limit() int

	Window() time.Duration

	Reset(ctx context.Context, key string) (time.Time, error)
}
