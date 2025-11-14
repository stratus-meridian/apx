package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apx/control/pkg/ratelimit"
	"github.com/apx/control/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockLimiter is a mock implementation of ratelimit.Limiter for testing
type mockLimiter struct {
	consumeFunc func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error)
	allowFunc   func(ctx context.Context, tenantID string) (*ratelimit.Result, error)
	resetFunc   func(ctx context.Context, tenantID string) error
	quotaFunc   func(ctx context.Context, tenantID string) (int64, int64, error)
	closeFunc   func() error
}

func (m *mockLimiter) Consume(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
	if m.consumeFunc != nil {
		return m.consumeFunc(ctx, tenantID, tokens)
	}
	return &ratelimit.Result{Allowed: true, Remaining: 99, Limit: 100, ResetAt: time.Now().Add(time.Minute)}, nil
}

func (m *mockLimiter) Allow(ctx context.Context, tenantID string) (*ratelimit.Result, error) {
	if m.allowFunc != nil {
		return m.allowFunc(ctx, tenantID)
	}
	return &ratelimit.Result{Allowed: true, Remaining: 100, Limit: 100, ResetAt: time.Now().Add(time.Minute)}, nil
}

func (m *mockLimiter) Reset(ctx context.Context, tenantID string) error {
	if m.resetFunc != nil {
		return m.resetFunc(ctx, tenantID)
	}
	return nil
}

func (m *mockLimiter) GetQuota(ctx context.Context, tenantID string) (int64, int64, error) {
	if m.quotaFunc != nil {
		return m.quotaFunc(ctx, tenantID)
	}
	return 0, 10000, nil
}

func (m *mockLimiter) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// createTestTenantForRateLimit creates a test tenant with the given tier for rate limit tests
func createTestTenantForRateLimit(tier tenant.Tier) *tenant.Tenant {
	return &tenant.Tenant{
		Organization: tenant.Organization{
			ID:   "test_org",
			Name: "Test Organization",
			Tier: tier,
		},
		Product: tenant.Product{
			ID:    "test_product",
			OrgID: "test_org",
			Name:  "Test Product",
		},
		Environment: tenant.Environment{
			ID:         "prod",
			OrgID:      "test_org",
			ProductID:  "test_product",
			ResourceID: "test_org_test_product_prod",
			Name:       "Production",
			Type:       tenant.EnvProduction,
		},
		ResourceID: "test_org_test_product_prod",
	}
}

// TestRateLimitMiddleware_AllowRequest tests that allowed requests pass through
func TestRateLimitMiddleware_AllowRequest(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			return &ratelimit.Result{
				Allowed:   true,
				Remaining: 99,
				Limit:     100,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Create request with tenant context
	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierPro)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	// Record response
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())

	// Verify rate limit headers
	assert.Equal(t, "100", rr.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "99", rr.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))
}

// TestRateLimitMiddleware_DenyRequest tests that denied requests return 429
func TestRateLimitMiddleware_DenyRequest(t *testing.T) {
	logger := zap.NewNop()
	resetAt := time.Now().Add(30 * time.Second)
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			return &ratelimit.Result{
				Allowed:    false,
				Remaining:  0,
				Limit:      100,
				ResetAt:    resetAt,
				RetryAfter: 30,
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when rate limited")
	}))

	// Create request with tenant context
	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierFree)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	// Record response
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Verify rate limit headers
	assert.Equal(t, "100", rr.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rr.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))
	assert.Equal(t, "30", rr.Header().Get("Retry-After"))

	// Verify response body
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "rate_limit_exceeded", response["error"])
	assert.Equal(t, "free", response["tier"])
	assert.Equal(t, float64(100), response["limit"])
	assert.Equal(t, float64(0), response["remaining"])
}

// TestRateLimitMiddleware_MissingTenant tests behavior when tenant is not in context
func TestRateLimitMiddleware_MissingTenant(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when tenant is missing")
	}))

	// Create request without tenant context
	req := httptest.NewRequest("GET", "/test", nil)

	// Record response
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	// Verify response body
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	errorMap := response["error"].(map[string]interface{})
	assert.Equal(t, "internal_error", errorMap["code"])
}

// TestRateLimitMiddleware_LimiterError tests fail-open behavior on limiter error
func TestRateLimitMiddleware_LimiterError(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			return nil, fmt.Errorf("redis connection failed")
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handlerCalled := false
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with tenant context
	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierPro)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	// Record response
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify request was allowed (fail-open behavior)
	assert.True(t, handlerCalled, "handler should be called on limiter error (fail-open)")
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify rate limit headers are still added (with estimated values)
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))
}

// TestRateLimitMiddleware_FreeTier tests rate limiting for free tier
func TestRateLimitMiddleware_FreeTier(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			// Free tier: 100 RPM
			return &ratelimit.Result{
				Allowed:   true,
				Remaining: 50,
				Limit:     100,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierFree)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "100", rr.Header().Get("X-RateLimit-Limit"))
}

// TestRateLimitMiddleware_ProTier tests rate limiting for pro tier
func TestRateLimitMiddleware_ProTier(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			// Pro tier: 1000 RPM
			return &ratelimit.Result{
				Allowed:   true,
				Remaining: 500,
				Limit:     1000,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierPro)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "1000", rr.Header().Get("X-RateLimit-Limit"))
}

// TestRateLimitMiddleware_EnterpriseTier tests rate limiting for enterprise tier
func TestRateLimitMiddleware_EnterpriseTier(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			// Enterprise tier: 10000 RPM
			return &ratelimit.Result{
				Allowed:   true,
				Remaining: 9000,
				Limit:     10000,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierEnterprise)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "10000", rr.Header().Get("X-RateLimit-Limit"))
}

// TestRateLimitMiddleware_QuotaRemaining tests quota information in response
func TestRateLimitMiddleware_QuotaRemaining(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			return &ratelimit.Result{
				Allowed:        false,
				Remaining:      0,
				Limit:          100,
				ResetAt:        time.Now().Add(time.Minute),
				RetryAfter:     60,
				QuotaRemaining: 5000,
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierFree)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(5000), response["quota_remaining"])
}

// TestRateLimitMiddleware_RetryAfterCalculation tests Retry-After header calculation
func TestRateLimitMiddleware_RetryAfterCalculation(t *testing.T) {
	logger := zap.NewNop()
	resetAt := time.Now().Add(45 * time.Second)
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			return &ratelimit.Result{
				Allowed:    false,
				Remaining:  0,
				Limit:      100,
				ResetAt:    resetAt,
				RetryAfter: 0, // Not set, should be calculated
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierFree)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Retry-After should be calculated from ResetAt
	retryAfter := rr.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter)
	// Should be approximately 45 seconds (allow some variation due to test execution time)
	// We just verify it's a reasonable number
	assert.NotEqual(t, "0", retryAfter)
}

// TestRateLimitMiddleware_MultipleRequests tests multiple sequential requests
func TestRateLimitMiddleware_MultipleRequests(t *testing.T) {
	logger := zap.NewNop()
	remaining := int64(3)

	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			result := &ratelimit.Result{
				Allowed:   remaining > 0,
				Remaining: remaining,
				Limit:     3,
				ResetAt:   time.Now().Add(time.Minute),
			}
			if remaining > 0 {
				remaining--
			}
			return result, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tenant := createTestTenantForRateLimit(tenant.TierFree)

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "request %d should succeed", i+1)
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "4th request should be rate limited")
}

// TestRateLimitMiddleware_DifferentTenants tests that different tenants have separate limits
func TestRateLimitMiddleware_DifferentTenants(t *testing.T) {
	logger := zap.NewNop()
	counters := make(map[string]int64)

	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			counters[tenantID]++
			return &ratelimit.Result{
				Allowed:   true,
				Remaining: 99,
				Limit:     100,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create two different tenants
	tenant1 := createTestTenantForRateLimit(tenant.TierFree)
	tenant1.ResourceID = "org1_product1_prod"

	tenant2 := createTestTenantForRateLimit(tenant.TierPro)
	tenant2.ResourceID = "org2_product2_prod"

	// Make request for tenant 1
	req1 := httptest.NewRequest("GET", "/test", nil)
	ctx1 := context.WithValue(req1.Context(), TenantContextKey, tenant1)
	req1 = req1.WithContext(ctx1)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// Make request for tenant 2
	req2 := httptest.NewRequest("GET", "/test", nil)
	ctx2 := context.WithValue(req2.Context(), TenantContextKey, tenant2)
	req2 = req2.WithContext(ctx2)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	// Both should succeed
	assert.Equal(t, http.StatusOK, rr1.Code)
	assert.Equal(t, http.StatusOK, rr2.Code)

	// Verify each tenant was tracked separately
	assert.Equal(t, int64(1), counters["org1_product1_prod"])
	assert.Equal(t, int64(1), counters["org2_product2_prod"])
}

// TestRateLimit_ConvenienceFunction tests the convenience function
func TestRateLimit_ConvenienceFunction(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{}

	// Use convenience function
	middlewareFunc := RateLimit(limiter, logger)
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierPro)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestRateLimitMiddleware_HeadersOnError tests that headers are added even on errors
func TestRateLimitMiddleware_HeadersOnError(t *testing.T) {
	logger := zap.NewNop()
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			return &ratelimit.Result{
				Allowed:   false,
				Remaining: 0,
				Limit:     100,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierFree)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify all rate limit headers are present
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))
	assert.NotEmpty(t, rr.Header().Get("Retry-After"))
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

// TestRateLimitMiddleware_ResponseFormat tests the 429 response format
func TestRateLimitMiddleware_ResponseFormat(t *testing.T) {
	logger := zap.NewNop()
	resetAt := time.Now().Add(30 * time.Second)
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			return &ratelimit.Result{
				Allowed:        false,
				Remaining:      0,
				Limit:          100,
				ResetAt:        resetAt,
				RetryAfter:     30,
				QuotaRemaining: 5000,
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	tenant := createTestTenantForRateLimit(tenant.TierFree)
	ctx := context.WithValue(req.Context(), TenantContextKey, tenant)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify all required fields are present
	assert.Equal(t, "rate_limit_exceeded", response["error"])
	assert.Contains(t, response["message"], "Rate limit")
	assert.Equal(t, "free", response["tier"])
	assert.Equal(t, float64(100), response["limit"])
	assert.Equal(t, float64(0), response["remaining"])
	assert.NotEmpty(t, response["reset_at"])
	assert.Equal(t, float64(30), response["retry_after"])
	assert.Equal(t, float64(5000), response["quota_remaining"])
}

// TestRateLimitMiddleware_PerformanceUnder5ms tests middleware performance
func TestRateLimitMiddleware_PerformanceUnder5ms(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	logger := zap.NewNop()
	limiter := &mockLimiter{
		consumeFunc: func(ctx context.Context, tenantID string, tokens int64) (*ratelimit.Result, error) {
			return &ratelimit.Result{
				Allowed:   true,
				Remaining: 99,
				Limit:     100,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := NewRateLimitMiddleware(limiter, logger)
	handler := middleware.Handler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tenant := createTestTenantForRateLimit(tenant.TierPro)

	// Run 100 requests and measure average time
	var totalDuration time.Duration
	iterations := 100

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
	t.Logf("Average middleware overhead: %v", avgDuration)

	// With mock limiter, middleware overhead should be well under 5ms
	assert.Less(t, avgDuration, 5*time.Millisecond, "middleware overhead should be less than 5ms")
}
