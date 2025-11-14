package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stratus-meridian/apx-private/control/pkg/ratelimit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// RateLimitMiddleware enforces per-tenant rate limits using the token bucket limiter.
// This middleware must be applied AFTER TenantContextMiddleware to access tenant information.
//
// Behavior:
//  1. Extracts tenant information from request context
//  2. Gets rate limit configuration for tenant tier
//  3. Consumes a token from the rate limiter
//  4. Adds X-RateLimit-* headers to response
//  5. Returns 429 Too Many Requests if rate limit exceeded
//  6. Logs rate limit events for monitoring
//
// Headers added to all responses:
//   X-RateLimit-Limit: Maximum requests per minute
//   X-RateLimit-Remaining: Tokens remaining in bucket
//   X-RateLimit-Reset: Unix timestamp when bucket refills
//
// Headers added to 429 responses:
//   Retry-After: Seconds to wait before retrying
type RateLimitMiddleware struct {
	limiter ratelimit.Limiter
	logger  *zap.Logger
	tracer  trace.Tracer
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(limiter ratelimit.Limiter, logger *zap.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
		tracer:  otel.Tracer("router.middleware.ratelimit"),
	}
}

// Handler returns the middleware handler function
func (m *RateLimitMiddleware) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Start span for rate limit check
			ctx, span := m.tracer.Start(ctx, "rate_limit_check")
			defer span.End()

			startTime := time.Now()

			// Get tenant from context (set by TenantContextMiddleware)
			tenant, ok := GetTenant(ctx)
			if !ok {
				// This should never happen if middleware order is correct
				span.RecordError(fmt.Errorf("tenant not found in context"))
				m.logger.Error("rate limit check failed: tenant not in context",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method))
				m.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to check rate limit")
				return
			}

			// Get tenant tier and resource ID
			tier := string(tenant.Organization.Tier)
			resourceID := tenant.ResourceID

			span.SetAttributes(
				attribute.String("tenant.resource_id", resourceID),
				attribute.String("tenant.tier", tier),
			)

			// Get rate limit configuration for tenant tier
			config := ratelimit.GetTenantConfigForTier(resourceID, tier)

			// Check and consume token from rate limiter
			result, err := m.limiter.Consume(ctx, resourceID, 1)
			if err != nil {
				// Log error but allow request (fail open behavior)
				span.RecordError(err)
				m.logger.Error("rate limit check error",
					zap.Error(err),
					zap.String("resource_id", resourceID),
					zap.String("tier", tier))

				// Add rate limit headers with current config
				m.addRateLimitHeaders(w, config.RequestsPerMinute, config.BurstLimit, time.Now().Add(time.Minute))

				// Allow request to proceed
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			duration := time.Since(startTime)

			// Add rate limit headers to response
			m.addRateLimitHeaders(w, result.Limit, result.Remaining, result.ResetAt)

			span.SetAttributes(
				attribute.Bool("rate_limit.allowed", result.Allowed),
				attribute.Int64("rate_limit.remaining", result.Remaining),
				attribute.Int64("rate_limit.limit", result.Limit),
				attribute.Int64("rate_limit.quota_remaining", result.QuotaRemaining),
			)

			// Check if rate limit exceeded
			if !result.Allowed {
				// Add Retry-After header (seconds until reset)
				retryAfter := result.RetryAfter
				if retryAfter <= 0 {
					retryAfter = int64(time.Until(result.ResetAt).Seconds())
					if retryAfter < 1 {
						retryAfter = 1
					}
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))

				// Log rate limit denial
				m.logger.Warn("rate limit exceeded",
					zap.String("resource_id", resourceID),
					zap.String("tier", tier),
					zap.Int64("limit", result.Limit),
					zap.Time("reset_at", result.ResetAt),
					zap.Int64("retry_after", retryAfter),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.Duration("check_duration", duration))

				// Send 429 response
				m.sendRateLimitError(w, tier, result)
				return
			}

			// Log successful rate limit check (at debug level to reduce noise)
			m.logger.Debug("rate limit check passed",
				zap.String("resource_id", resourceID),
				zap.String("tier", tier),
				zap.Int64("remaining", result.Remaining),
				zap.Int64("limit", result.Limit),
				zap.Duration("check_duration", duration))

			// Request allowed, proceed to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// addRateLimitHeaders adds standard rate limit headers to the response
func (m *RateLimitMiddleware) addRateLimitHeaders(w http.ResponseWriter, limit, remaining int64, resetAt time.Time) {
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))
}

// sendRateLimitError sends a 429 Too Many Requests response with details
func (m *RateLimitMiddleware) sendRateLimitError(w http.ResponseWriter, tier string, result *ratelimit.Result) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"error":   "rate_limit_exceeded",
		"message": fmt.Sprintf("Rate limit of %d requests per minute exceeded", result.Limit),
		"tier":    tier,
		"limit":   result.Limit,
		"remaining": result.Remaining,
		"reset_at": result.ResetAt.Format(time.RFC3339),
		"retry_after": result.RetryAfter,
	}

	// Add quota information if available
	if result.QuotaRemaining >= 0 {
		response["quota_remaining"] = result.QuotaRemaining
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.Error("failed to encode rate limit error response", zap.Error(err))
	}
}

// sendError sends a JSON error response
func (m *RateLimitMiddleware) sendError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": map[string]string{
			"code":    errorCode,
			"message": message,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.Error("failed to encode error response", zap.Error(err))
	}
}

// RateLimit is a convenience function that creates and returns a rate limit middleware
// This is useful for quick setup without creating the middleware struct explicitly
func RateLimit(limiter ratelimit.Limiter, logger *zap.Logger) Middleware {
	middleware := NewRateLimitMiddleware(limiter, logger)
	return middleware.Handler()
}
