// +build integration

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stratus-meridian/apx-private/control/pkg/ratelimit"
	"github.com/stratus-meridian/apx-private/control/tenant"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// getRedisClient creates a Redis client for testing
func getRedisClient() *redis.Client {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	return redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           1, // Use DB 1 for tests
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})
}

// setupRedisLimiter creates a Redis limiter for testing
func setupRedisLimiter(t *testing.T) (ratelimit.Limiter, func()) {
	client := getRedisClient()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}

	// Create limiter with test config
	config := ratelimit.DefaultConfig()
	config.RedisAddr = client.Options().Addr
	config.RedisPassword = client.Options().Password
	config.RedisDB = client.Options().DB
	config.FailureMode = ratelimit.FailOpen

	limiter, err := ratelimit.NewRedisLimiter(config)
	require.NoError(t, err)

	// Cleanup function
	cleanup := func() {
		// Clean up test keys
		ctx := context.Background()
		iter := client.Scan(ctx, 0, "rl:tenant:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
		limiter.Close()
		client.Close()
	}

	return limiter, cleanup
}

// TestRateLimitMiddleware_Integration_BasicFlow tests basic rate limiting with Redis
func TestRateLimitMiddleware_Integration_BasicFlow(t *testing.T) {
	limiter, cleanup := setupRedisLimiter(t)
	defer cleanup()

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	// Configure tenant with low limit for testing
	tenant := createTestTenantForRateLimit(tenant.TierFree)
	tenantConfig := ratelimit.GetTenantConfigForTier(tenant.ResourceID, string(tenant.Organization.Tier))

	// Override with lower limit for faster testing
	tenantConfig.RequestsPerMinute = 5
	tenantConfig.BurstLimit = 5
	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(tenantConfig)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "request %d should succeed", i+1)
		assert.Equal(t, "5", rr.Header().Get("X-RateLimit-Limit"))
	}

	// 6th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "6th request should be rate limited")
	assert.Equal(t, "0", rr.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rr.Header().Get("Retry-After"))
}

// TestRateLimitMiddleware_Integration_TokenRefill tests token refill over time
func TestRateLimitMiddleware_Integration_TokenRefill(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping time-dependent test in short mode")
	}

	limiter, cleanup := setupRedisLimiter(t)
	defer cleanup()

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	tenant := createTestTenantForRateLimit(tenant.TierFree)

	// Configure with 2 tokens per second (120 per minute)
	tenantConfig := ratelimit.GetTenantConfigForTier(tenant.ResourceID, string(tenant.Organization.Tier))
	tenantConfig.RequestsPerMinute = 120 // 2 per second
	tenantConfig.BurstLimit = 3
	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(tenantConfig)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Consume all tokens
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Next request should fail
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Wait for refill (1 second = 2 tokens)
	time.Sleep(1100 * time.Millisecond)

	// Should be able to make 2 more requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request after refill %d should succeed", i+1)
	}
}

// TestRateLimitMiddleware_Integration_DifferentTenants tests isolation between tenants
func TestRateLimitMiddleware_Integration_DifferentTenants(t *testing.T) {
	limiter, cleanup := setupRedisLimiter(t)
	defer cleanup()

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	// Create two tenants with different limits
	tenant1 := createTestTenant(tenant.TierFree)
	tenant1.ResourceID = "org1_product1_prod"
	config1 := ratelimit.GetTenantConfigForTier(tenant1.ResourceID, "free")
	config1.RequestsPerMinute = 2
	config1.BurstLimit = 2
	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(config1)

	tenant2 := createTestTenant(tenant.TierPro)
	tenant2.ResourceID = "org2_product2_prod"
	config2 := ratelimit.GetTenantConfigForTier(tenant2.ResourceID, "pro")
	config2.RequestsPerMinute = 5
	config2.BurstLimit = 5
	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(config2)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust tenant1's limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), TenantContextKey, tenant1)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Tenant1 should be rate limited
	req1 := httptest.NewRequest("GET", "/test", nil)
	ctx1 := context.WithValue(req1.Context(), TenantContextKey, tenant1)
	req1 = req1.WithContext(ctx1)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	assert.Equal(t, http.StatusTooManyRequests, rr1.Code, "tenant1 should be rate limited")

	// Tenant2 should still be able to make requests
	req2 := httptest.NewRequest("GET", "/test", nil)
	ctx2 := context.WithValue(req2.Context(), TenantContextKey, tenant2)
	req2 = req2.WithContext(ctx2)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code, "tenant2 should not be affected")
}

// TestRateLimitMiddleware_Integration_FreeTierLimit tests free tier specific limits
func TestRateLimitMiddleware_Integration_FreeTierLimit(t *testing.T) {
	limiter, cleanup := setupRedisLimiter(t)
	defer cleanup()

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	tenant := createTestTenantForRateLimit(tenant.TierFree)

	// Use default free tier config: 100 RPM
	config := ratelimit.GetTenantConfigForTier(tenant.ResourceID, "free")
	assert.Equal(t, int64(100), config.RequestsPerMinute)
	assert.Equal(t, int64(150), config.BurstLimit)
	assert.Equal(t, int64(10000), config.MonthlyQuota)

	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(config)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "100", rr.Header().Get("X-RateLimit-Limit"))
}

// TestRateLimitMiddleware_Integration_ProTierLimit tests pro tier specific limits
func TestRateLimitMiddleware_Integration_ProTierLimit(t *testing.T) {
	limiter, cleanup := setupRedisLimiter(t)
	defer cleanup()

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	tenant := createTestTenantForRateLimit(tenant.TierPro)

	// Use default pro tier config: 1000 RPM
	config := ratelimit.GetTenantConfigForTier(tenant.ResourceID, "pro")
	assert.Equal(t, int64(1000), config.RequestsPerMinute)
	assert.Equal(t, int64(1500), config.BurstLimit)
	assert.Equal(t, int64(1000000), config.MonthlyQuota)

	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(config)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "1000", rr.Header().Get("X-RateLimit-Limit"))
}

// TestRateLimitMiddleware_Integration_EnterpriseTierUnlimited tests enterprise tier limits
func TestRateLimitMiddleware_Integration_EnterpriseTierUnlimited(t *testing.T) {
	limiter, cleanup := setupRedisLimiter(t)
	defer cleanup()

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	tenant := createTestTenantForRateLimit(tenant.TierEnterprise)

	// Use default enterprise tier config: 10000 RPM, unlimited quota
	config := ratelimit.GetTenantConfigForTier(tenant.ResourceID, "enterprise")
	assert.Equal(t, int64(10000), config.RequestsPerMinute)
	assert.Equal(t, int64(15000), config.BurstLimit)
	assert.Equal(t, int64(-1), config.MonthlyQuota) // Unlimited

	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(config)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "10000", rr.Header().Get("X-RateLimit-Limit"))
}

// TestRateLimitMiddleware_Integration_ConcurrentRequests tests concurrent request handling
func TestRateLimitMiddleware_Integration_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	limiter, cleanup := setupRedisLimiter(t)
	defer cleanup()

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	tenant := createTestTenantForRateLimit(tenant.TierFree)

	// Set low limit for testing
	config := ratelimit.GetTenantConfigForTier(tenant.ResourceID, "free")
	config.RequestsPerMinute = 10
	config.BurstLimit = 10
	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(config)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make 20 concurrent requests
	successCount := 0
	deniedCount := 0
	results := make(chan int, 20)

	for i := 0; i < 20; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			results <- rr.Code
		}()
	}

	// Collect results
	for i := 0; i < 20; i++ {
		code := <-results
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusTooManyRequests {
			deniedCount++
		}
	}

	// Should have exactly 10 successes and 10 denials
	assert.Equal(t, 10, successCount, "should have 10 successful requests")
	assert.Equal(t, 10, deniedCount, "should have 10 denied requests")
}

// TestRateLimitMiddleware_Integration_RedisFailover tests fail-open behavior
func TestRateLimitMiddleware_Integration_RedisFailover(t *testing.T) {
	// Create limiter with invalid Redis address to simulate failure
	config := ratelimit.DefaultConfig()
	config.RedisAddr = "invalid:9999"
	config.FailureMode = ratelimit.FailOpen
	config.DialTimeout = 100 * time.Millisecond

	limiter, err := ratelimit.NewRedisLimiter(config)
	require.NoError(t, err) // Should not error on creation, just warn

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tenant := createTestTenantForRateLimit(tenant.TierFree)
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// With fail-open, request should succeed even when Redis is down
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())

	limiter.Close()
}

// TestRateLimitMiddleware_Integration_PerformanceWithRedis tests performance with Redis
func TestRateLimitMiddleware_Integration_PerformanceWithRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	limiter, cleanup := setupRedisLimiter(t)
	defer cleanup()

	logger := zap.NewNop()
	middleware := NewRateLimitMiddleware(limiter, logger)

	tenant := createTestTenantForRateLimit(tenant.TierPro)
	config := ratelimit.GetTenantConfigForTier(tenant.ResourceID, "pro")
	limiter.(*ratelimit.RedisLimiter).SetTenantConfig(config)

	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Warm up
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// Measure performance
	iterations := 100
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
		req = req.WithContext(ctx)

		start := time.Now()
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	t.Logf("Average middleware overhead with Redis: %v", avgDuration)

	// With Redis on localhost, should be under 10ms (relaxed from 5ms for network overhead)
	assert.Less(t, avgDuration, 10*time.Millisecond, "middleware overhead should be less than 10ms")
}
