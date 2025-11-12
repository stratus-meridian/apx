package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements tenant-isolated rate limiting using Redis
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
	}
}

// RateLimitKey generates a tenant-isolated Redis key
// Pattern: apx:rl:{tenant_id}:{resource}
// This ensures complete keyspace isolation between tenants
func RateLimitKey(tenantID, resource string) string {
	// CRITICAL: Tenant ID MUST be part of key to prevent cross-tenant pollution
	return fmt.Sprintf("apx:rl:%s:%s", tenantID, resource)
}

// CheckRateLimit checks if a tenant has exceeded their rate limit
// Returns nil if under limit, error if exceeded
func (r *RedisRateLimiter) CheckRateLimit(ctx context.Context, tenantID string, resource string, limit int, window time.Duration) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required for rate limiting")
	}

	key := RateLimitKey(tenantID, resource)

	// Use Redis INCR for atomic counter increment
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to increment rate limit counter: %w", err)
	}

	// Set expiration on first increment
	if count == 1 {
		if err := r.client.Expire(ctx, key, window).Err(); err != nil {
			return fmt.Errorf("failed to set expiration: %w", err)
		}
	}

	// Check if limit exceeded
	if count > int64(limit) {
		ttl, _ := r.client.TTL(ctx, key).Result()
		return &RateLimitExceededError{
			TenantID:  tenantID,
			Resource:  resource,
			Limit:     limit,
			Current:   int(count),
			RetryAfter: ttl,
		}
	}

	return nil
}

// GetCurrentUsage returns current rate limit usage for a tenant
func (r *RedisRateLimiter) GetCurrentUsage(ctx context.Context, tenantID string, resource string) (int, error) {
	if tenantID == "" {
		return 0, fmt.Errorf("tenant_id is required")
	}

	key := RateLimitKey(tenantID, resource)
	count, err := r.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get rate limit usage: %w", err)
	}

	return count, nil
}

// ResetRateLimit resets the rate limit counter for a tenant (admin operation)
func (r *RedisRateLimiter) ResetRateLimit(ctx context.Context, tenantID string, resource string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	key := RateLimitKey(tenantID, resource)
	return r.client.Del(ctx, key).Err()
}

// ListTenantKeys returns all rate limit keys for a specific tenant (debugging only)
// WARNING: This should only be used for debugging/admin purposes
func (r *RedisRateLimiter) ListTenantKeys(ctx context.Context, tenantID string) ([]string, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	pattern := fmt.Sprintf("apx:rl:%s:*", tenantID)

	var keys []string
	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan keys: %w", err)
	}

	return keys, nil
}

// RateLimitExceededError represents a rate limit violation
type RateLimitExceededError struct {
	TenantID   string
	Resource   string
	Limit      int
	Current    int
	RetryAfter time.Duration
}

func (e *RateLimitExceededError) Error() string {
	return fmt.Sprintf("rate limit exceeded for tenant %s on resource %s: %d/%d (retry after %v)",
		e.TenantID, e.Resource, e.Current, e.Limit, e.RetryAfter)
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitExceededError)
	return ok
}
