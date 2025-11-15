package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stratus-meridian/apx-private/control/pkg/ratelimit"
	"github.com/stratus-meridian/apx-private/control/tenant"
	"github.com/stratus-meridian/apx/router/internal/middleware"
	"github.com/stratus-meridian/apx/router/pkg/responses"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestQuotaEnforcement_FreeTier tests quota enforcement for free tier
func TestQuotaEnforcement_FreeTier(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	// Setup Redis client for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use test DB
	})
	defer redisClient.Close()

	// Clear test data
	redisClient.FlushDB(ctx)

	// Create quota enforcer
	enforcer := ratelimit.NewQuotaEnforcer(redisClient)

	// Create test tenant (Free tier with 10,000 monthly quota)
	testTenant := &tenant.Tenant{
		Organization: tenant.Organization{
			ID:     "test-org",
			Name:   "Test Organization",
			Tier:   tenant.TierFree,
			Status: tenant.StatusActive,
			Quotas: tenant.GetDefaultQuotas(tenant.TierFree),
		},
		Product: tenant.Product{
			ID:     "test-product",
			OrgID:  "test-org",
			Name:   "Test Product",
		},
		Environment: tenant.Environment{
			ID:         "prod",
			OrgID:      "test-org",
			ProductID:  "test-product",
			ResourceID: "test-org_test-product_prod",
			Type:       tenant.EnvProduction,
			Status:     tenant.StatusActive,
		},
		ResourceID: "test-org_test-product_prod",
	}

	// Create quota middleware
	quotaMiddleware := middleware.NewQuotaMiddleware(enforcer, logger)

	// Test handler that returns 200 OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Wrap with quota middleware
	handler := quotaMiddleware.Handler()(testHandler)

	// Test 1: First request should succeed
	req := httptest.NewRequest("POST", "/test", nil)
	ctx = context.WithValue(ctx, middleware.TenantContextKey, testTenant)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Test 2: Simulate 10,000 requests (quota limit for free tier)
	for i := 0; i < 9999; i++ {
		_, err := enforcer.IncrementQuota(ctx, testTenant.ResourceID, testTenant.Organization.Tier, 1)
		require.NoError(t, err)
	}

	// Test 3: Request at quota limit should still succeed (we've used 10,000 total)
	req = httptest.NewRequest("POST", "/test", nil)
	ctx = context.WithValue(context.Background(), middleware.TenantContextKey, testTenant)
	req = req.WithContext(ctx)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Test 4: Request over quota limit should return HTTP 402
	req = httptest.NewRequest("POST", "/test", nil)
	ctx = context.WithValue(context.Background(), middleware.TenantContextKey, testTenant)
	req = req.WithContext(ctx)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusPaymentRequired, rec.Code)

	// Validate HTTP 402 response structure
	var paymentRequired responses.PaymentRequiredResponse
	err := json.NewDecoder(rec.Body).Decode(&paymentRequired)
	require.NoError(t, err)

	assert.Equal(t, "quota_exceeded", paymentRequired.Error)
	assert.Equal(t, "free", paymentRequired.Tier)
	assert.Equal(t, int64(10001), paymentRequired.CurrentUsage) // Over limit by 1
	assert.Equal(t, int64(10000), paymentRequired.Limit)
	assert.Equal(t, int64(1), paymentRequired.Overage)
	assert.NotEmpty(t, paymentRequired.UpgradeURL)
	assert.Equal(t, "pro", paymentRequired.SuggestedTier)
	assert.NotNil(t, paymentRequired.Pricing)
	assert.Equal(t, int64(4900), paymentRequired.Pricing.MonthlyPrice) // $49/month for Pro
}

// TestQuotaEnforcement_HTTP402Response validates the HTTP 402 response includes all required fields
func TestQuotaEnforcement_HTTP402Response(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer redisClient.Close()

	redisClient.FlushDB(ctx)

	enforcer := ratelimit.NewQuotaEnforcer(redisClient)

	testTenant := &tenant.Tenant{
		Organization: tenant.Organization{
			ID:     "test-org",
			Tier:   tenant.TierFree,
			Status: tenant.StatusActive,
			Quotas: tenant.GetDefaultQuotas(tenant.TierFree),
		},
		ResourceID: "test-org_test-product_prod",
	}

	// Exhaust quota
	for i := 0; i < 10001; i++ {
		enforcer.IncrementQuota(ctx, testTenant.ResourceID, testTenant.Organization.Tier, 1)
	}

	quotaMiddleware := middleware.NewQuotaMiddleware(enforcer, logger)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := quotaMiddleware.Handler()(testHandler)

	req := httptest.NewRequest("POST", "/test", nil)
	ctx = context.WithValue(context.Background(), middleware.TenantContextKey, testTenant)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusPaymentRequired, rec.Code)

	var response responses.PaymentRequiredResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Validate all required fields are present
	assert.Equal(t, "quota_exceeded", response.Error)
	assert.NotEmpty(t, response.Message)
	assert.Equal(t, "free", response.Tier)
	assert.Greater(t, response.CurrentUsage, int64(10000))
	assert.Equal(t, int64(10000), response.Limit)
	assert.Greater(t, response.Overage, int64(0))
	assert.NotZero(t, response.ResetAt)
	assert.NotEmpty(t, response.UpgradeURL)
	assert.Equal(t, "pro", response.SuggestedTier)
	assert.NotNil(t, response.Pricing)
	assert.Equal(t, "pro", response.Pricing.Tier)
	assert.Equal(t, int64(4900), response.Pricing.MonthlyPrice)
	assert.Equal(t, int64(1000000), response.Pricing.MonthlyQuota) // Pro: 1M/month
}

// TestQuotaEnforcement_ProTierOverage tests that Pro tier allows overage
func TestQuotaEnforcement_ProTierOverage(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer redisClient.Close()

	redisClient.FlushDB(ctx)

	enforcer := ratelimit.NewQuotaEnforcer(redisClient)

	// Pro tier tenant with 1M monthly quota
	proTenant := &tenant.Tenant{
		Organization: tenant.Organization{
			ID:     "pro-org",
			Tier:   tenant.TierPro,
			Status: tenant.StatusActive,
			Quotas: tenant.GetDefaultQuotas(tenant.TierPro),
		},
		ResourceID: "pro-org_product_prod",
	}

	quotaMiddleware := middleware.NewQuotaMiddleware(enforcer, logger)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := quotaMiddleware.Handler()(testHandler)

	// Simulate using 1M requests (quota limit)
	for i := 0; i < 1000000; i += 1000 {
		enforcer.IncrementQuota(ctx, proTenant.ResourceID, proTenant.Organization.Tier, 1000)
	}

	// Test: Pro tier should ALLOW overage (config.OverageAllowed = true)
	req := httptest.NewRequest("POST", "/test", nil)
	ctx = context.WithValue(context.Background(), middleware.TenantContextKey, proTenant)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Pro tier should still be allowed even over quota
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestQuotaEnforcement_MonthlyReset tests that quotas reset correctly
func TestQuotaEnforcement_MonthlyReset(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer redisClient.Close()

	redisClient.FlushDB(ctx)

	enforcer := ratelimit.NewQuotaEnforcer(redisClient)

	testTenant := &tenant.Tenant{
		Organization: tenant.Organization{
			ID:     "test-org",
			Tier:   tenant.TierFree,
			Status: tenant.StatusActive,
		},
		ResourceID: "test-org_product_prod",
	}

	// Exhaust quota
	for i := 0; i < 10001; i++ {
		enforcer.IncrementQuota(ctx, testTenant.ResourceID, testTenant.Organization.Tier, 1)
	}

	// Verify quota is exhausted
	status, err := enforcer.CheckQuota(ctx, testTenant.ResourceID, testTenant.Organization.Tier)
	require.NoError(t, err)
	assert.True(t, status.IsExhausted)

	// Reset quota (simulating monthly reset)
	err = enforcer.ResetQuota(ctx, testTenant.ResourceID)
	require.NoError(t, err)

	// Verify quota is reset
	status, err = enforcer.CheckQuota(ctx, testTenant.ResourceID, testTenant.Organization.Tier)
	require.NoError(t, err)
	assert.False(t, status.IsExhausted)
	assert.Equal(t, int64(0), status.Used)
	assert.Equal(t, int64(10000), status.Remaining)
}

// TestQuotaEnforcement_ResetAllQuotas tests bulk quota reset
func TestQuotaEnforcement_ResetAllQuotas(t *testing.T) {
	ctx := context.Background()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	defer redisClient.Close()

	redisClient.FlushDB(ctx)

	enforcer := ratelimit.NewQuotaEnforcer(redisClient)

	// Create multiple tenants with different usage
	tenants := []string{
		"tenant1_product_prod",
		"tenant2_product_prod",
		"tenant3_product_prod",
	}

	for _, tenantID := range tenants {
		for i := 0; i < 5000; i++ {
			enforcer.IncrementQuota(ctx, tenantID, tenant.TierFree, 1)
		}
	}

	// Reset all quotas
	resetCount, err := enforcer.ResetAllQuotas(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resetCount, 3) // At least 3 tenants reset

	// Verify all tenants have reset quotas
	for _, tenantID := range tenants {
		status, err := enforcer.CheckQuota(ctx, tenantID, tenant.TierFree)
		require.NoError(t, err)
		assert.Equal(t, int64(0), status.Used, "Tenant %s should have 0 usage after reset", tenantID)
	}
}
