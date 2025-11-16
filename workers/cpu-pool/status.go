package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Status constants for Redis storage
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusComplete   = "complete"
	StatusFailed     = "failed"
)

// StatusStore interface for managing request status in Redis
type StatusStore interface {
	UpdateStatus(ctx context.Context, requestID, status string) error
	UpdateProgress(ctx context.Context, requestID string, progress int) error
	SetResult(ctx context.Context, requestID string, result json.RawMessage) error
	SetError(ctx context.Context, requestID string, errorMsg string) error
}

// RedisStatusStore implements StatusStore using Redis
type RedisStatusStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisStatusStore(client *redis.Client, ttl time.Duration) StatusStore {
	return &RedisStatusStore{
		client: client,
		ttl:    ttl,
	}
}

func (r *RedisStatusStore) UpdateStatus(ctx context.Context, requestID, status string) error {
	key := fmt.Sprintf("status:%s", requestID)
	return r.client.HSet(ctx, key, "status", status).Err()
}

func (r *RedisStatusStore) UpdateProgress(ctx context.Context, requestID string, progress int) error {
	key := fmt.Sprintf("status:%s", requestID)
	return r.client.HSet(ctx, key, "progress", progress).Err()
}

func (r *RedisStatusStore) SetResult(ctx context.Context, requestID string, result json.RawMessage) error {
	key := fmt.Sprintf("status:%s", requestID)
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, "status", StatusComplete)
	pipe.HSet(ctx, key, "progress", 100)
	pipe.HSet(ctx, key, "result", string(result))
	pipe.HSet(ctx, key, "completed_at", time.Now().Unix())
	pipe.Expire(ctx, key, r.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStatusStore) SetError(ctx context.Context, requestID string, errorMsg string) error {
	key := fmt.Sprintf("status:%s", requestID)
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, "status", StatusFailed)
	pipe.HSet(ctx, key, "error", errorMsg)
	pipe.HSet(ctx, key, "completed_at", time.Now().Unix())
	pipe.Expire(ctx, key, r.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

