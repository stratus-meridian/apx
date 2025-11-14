package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/stratus-meridian/apx-router-open-core/internal/auth"
	"go.uber.org/zap"
)

// RateLimitMiddleware enforces per-tenant rate limits.
// This middleware must be applied AFTER TenantContext middleware.
type RateLimitMiddleware struct {
	limiter *SimpleRateLimiter
	logger  *zap.Logger
}

// NewRateLimitMiddleware creates a new rate limiting middleware.
func NewRateLimitMiddleware(limiter *SimpleRateLimiter, logger *zap.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
	}
}

// Handler returns the middleware handler function.
func (m *RateLimitMiddleware) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get tenant from context (set by TenantContext middleware)
			tenant, ok := GetTenant(ctx)
			if !ok {
				m.logger.Error("rate limit check failed: tenant not in context")
				m.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to check rate limit")
				return
			}

			tenantID := tenant.ID
			rpm := tenant.RPM

			// Check rate limit
			allowed := m.limiter.Allow(tenantID, rpm)
			if !allowed {
				m.logger.Warn("rate limit exceeded",
					zap.String("tenant_id", tenantID),
					zap.Int("rpm", rpm),
					zap.String("path", r.URL.Path))

				m.sendRateLimitError(w, tenant)
				return
			}

			// Request allowed, proceed
			next.ServeHTTP(w, r)
		})
	}
}

// sendRateLimitError sends an HTTP 429 Too Many Requests response.
func (m *RateLimitMiddleware) sendRateLimitError(w http.ResponseWriter, tenant *auth.SimpleTenant) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Limit", string(rune(tenant.RPM)))
	w.Header().Set("Retry-After", "60") // Retry after 60 seconds
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "rate_limit_exceeded",
			"message": "Rate limit exceeded. Please retry after 60 seconds.",
			"details": map[string]interface{}{
				"tenant_id": tenant.ID,
				"tier":      tenant.Tier,
				"limit_rpm": tenant.RPM,
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

// sendError sends a JSON error response.
func (m *RateLimitMiddleware) sendError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": map[string]string{
			"code":    errorCode,
			"message": message,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// RateLimit is a convenience function that creates and returns rate limit middleware.
func RateLimit(limiter *SimpleRateLimiter, logger *zap.Logger) Middleware {
	middleware := NewRateLimitMiddleware(limiter, logger)
	return middleware.Handler()
}
