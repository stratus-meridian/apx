package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisLimiter implements token bucket rate limiting using Redis
type RedisLimiter struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisLimiter creates a new Redis-based rate limiter
func NewRedisLimiter(client *redis.Client, logger *zap.Logger) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		logger: logger,
	}
}

// RateLimit represents rate limit configuration for a tier
type RateLimit struct {
	Capacity   int     // Max tokens (burst capacity)
	RefillRate float64 // Tokens per second (sustained rate)
}

// Allow checks if request is allowed based on rate limit
// Uses token bucket algorithm with Redis for atomic operations
func (rl *RedisLimiter) Allow(ctx context.Context, tenantID, tier string) (bool, error) {
	if tenantID == "" {
		rl.logger.Warn("rate limit check without tenant ID")
		return true, nil // Fail open
	}

	// Get rate limit config for tier
	limit := rl.getLimitForTier(tier)

	// Redis key for tenant's token bucket
	key := fmt.Sprintf("apx:rl:%s:tokens", tenantID)

	// Lua script for atomic token bucket operations
	// This ensures thread-safe token bucket updates across distributed router instances
	script := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])

		local tokens_key = key
		local timestamp_key = key .. ":ts"

		local last_tokens = tonumber(redis.call("get", tokens_key))
		local last_time = tonumber(redis.call("get", timestamp_key))

		if last_tokens == nil then
			last_tokens = capacity
		end

		if last_time == nil then
			last_time = now
		end

		local delta = math.max(0, now - last_time)
		local new_tokens = math.min(capacity, last_tokens + (delta * rate))

		if new_tokens < requested then
			return 0  -- Rate limited
		end

		new_tokens = new_tokens - requested

		redis.call("setex", tokens_key, 3600, new_tokens)
		redis.call("setex", timestamp_key, 3600, now)

		return 1  -- Allowed
	`

	result, err := rl.client.Eval(ctx, script, []string{key},
		limit.Capacity, limit.RefillRate, time.Now().Unix(), 1).Result()

	if err != nil {
		rl.logger.Error("rate limit check failed", zap.Error(err))
		// Fail open (allow request) on Redis errors to prevent outages
		return true, err
	}

	allowed := result.(int64) == 1

	if !allowed {
		rl.logger.Warn("rate limit exceeded",
			zap.String("tenant_id", tenantID),
			zap.String("tier", tier),
			zap.Int("capacity", limit.Capacity),
			zap.Float64("rate", limit.RefillRate))
	}

	return allowed, nil
}

// getLimitForTier returns rate limit configuration for a given tier
func (rl *RedisLimiter) getLimitForTier(tier string) RateLimit {
	limits := map[string]RateLimit{
		"free":       {Capacity: 10, RefillRate: 1.0},     // 10 req burst, 1/s sustained
		"pro":        {Capacity: 100, RefillRate: 10.0},   // 100 req burst, 10/s sustained
		"enterprise": {Capacity: 1000, RefillRate: 100.0}, // 1000 req burst, 100/s sustained
	}

	if limit, ok := limits[tier]; ok {
		return limit
	}
	return limits["free"] // Default to free tier
}

// GetCurrentUsage returns current token count for a tenant
func (rl *RedisLimiter) GetCurrentUsage(ctx context.Context, tenantID string) (float64, error) {
	if tenantID == "" {
		return 0, fmt.Errorf("tenant_id is required")
	}

	key := fmt.Sprintf("apx:rl:%s:tokens", tenantID)
	tokens, err := rl.client.Get(ctx, key).Float64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get rate limit usage: %w", err)
	}

	return tokens, nil
}

// ResetRateLimit resets the rate limit for a tenant (admin operation)
func (rl *RedisLimiter) ResetRateLimit(ctx context.Context, tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	key := fmt.Sprintf("apx:rl:%s:tokens", tenantID)
	tsKey := fmt.Sprintf("apx:rl:%s:tokens:ts", tenantID)

	pipe := rl.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, tsKey)
	_, err := pipe.Exec(ctx)

	return err
}
