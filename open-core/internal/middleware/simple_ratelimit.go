package middleware

import (
	"sync"
	"time"
)

// SimpleRateLimiter implements a basic in-memory token bucket rate limiter.
// This is a demonstration implementation suitable for development and small deployments.
//
// The commercial version uses Redis for distributed rate limiting across
// multiple router instances with persistent state.
//
// Algorithm: Token Bucket
// - Each tenant has a bucket with a fixed capacity (RPM limit)
// - Tokens are added at a constant rate (refill)
// - Each request consumes one token
// - If no tokens available, request is rate limited
type SimpleRateLimiter struct {
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	capacity   float64
	refillRate float64 // tokens per second
}

// NewSimpleRateLimiter creates a new in-memory rate limiter.
func NewSimpleRateLimiter() *SimpleRateLimiter {
	limiter := &SimpleRateLimiter{
		buckets: make(map[string]*tokenBucket),
	}

	// Start cleanup goroutine to prevent memory leaks
	go limiter.cleanup()

	return limiter
}

// Allow checks if a request should be allowed based on rate limit.
// tenantID: unique identifier for the tenant
// rpm: requests per minute limit for this tenant
// Returns true if request should be allowed, false if rate limited.
func (rl *SimpleRateLimiter) Allow(tenantID string, rpm int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[tenantID]
	if !exists {
		// Create new bucket for this tenant
		capacity := float64(rpm)
		bucket = &tokenBucket{
			tokens:     capacity,
			lastRefill: time.Now(),
			capacity:   capacity,
			refillRate: capacity / 60.0, // convert RPM to tokens per second
		}
		rl.buckets[tenantID] = bucket
	}

	// Refill tokens based on time elapsed since last refill
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = min(bucket.capacity, bucket.tokens+elapsed*bucket.refillRate)
	bucket.lastRefill = now

	// Check if token available
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	// Rate limit exceeded
	return false
}

// cleanup removes stale buckets to prevent memory leaks.
// Runs periodically in the background.
func (rl *SimpleRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// Remove buckets not used in last 10 minutes
		cutoff := time.Now().Add(-10 * time.Minute)
		for id, bucket := range rl.buckets {
			if bucket.lastRefill.Before(cutoff) {
				delete(rl.buckets, id)
			}
		}
		rl.mu.Unlock()
	}
}

// Close stops the cleanup goroutine and releases resources.
func (rl *SimpleRateLimiter) Close() error {
	// In this simple implementation, we don't track the cleanup goroutine
	// In production, you'd want to use a context to gracefully stop it
	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
