package middleware

import (
	"net/http"

	"github.com/apx/router/internal/ratelimit"
	"go.uber.org/zap"
)

// RateLimit returns middleware that enforces per-tenant rate limits
func RateLimit(limiter *ratelimit.RedisLimiter, logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get tenant info from context (set by TenantContext middleware)
			tenantID := GetTenantID(ctx)
			tenantTier := GetTenantTier(ctx)

			// Check rate limit
			allowed, err := limiter.Allow(ctx, tenantID, tenantTier)
			if err != nil {
				// Log error but allow request (fail open on Redis errors)
				logger.Error("rate limit check failed", zap.Error(err))
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				// Return 429 Too Many Requests
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", getRateLimitHeader(tenantTier))
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Too many requests. Please slow down."}`))

				logger.Warn("request rate limited",
					zap.String("tenant_id", tenantID),
					zap.String("tier", tenantTier),
					zap.String("path", r.URL.Path))
				return
			}

			// Add rate limit headers to successful responses
			w.Header().Set("X-RateLimit-Limit", getRateLimitHeader(tenantTier))

			next.ServeHTTP(w, r)
		})
	}
}

// getRateLimitHeader returns human-readable rate limit for a tier
func getRateLimitHeader(tier string) string {
	limits := map[string]string{
		"free":       "1/s",
		"pro":        "10/s",
		"enterprise": "100/s",
	}
	if limit, ok := limits[tier]; ok {
		return limit
	}
	return "1/s"
}
