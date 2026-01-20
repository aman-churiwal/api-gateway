package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedis(addr, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

func (r *RedisClient) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return r.client.ZAdd(ctx, key, members...).Err()
}

func (r *RedisClient) ZCount(ctx context.Context, key, min, max string) (int64, error) {
	return r.client.ZCount(ctx, key, min, max).Result()
}

func (r *RedisClient) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(ctx, key, start, stop).Result()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
