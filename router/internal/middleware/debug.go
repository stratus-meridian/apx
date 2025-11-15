package middleware

import (
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

// debugEnabled is controlled via ROUTER_DEBUG_MIDDLEWARE=1
var debugEnabled = os.Getenv("ROUTER_DEBUG_MIDDLEWARE") == "1"

// IsDebugEnabled exposes whether middleware step logging is enabled.
func IsDebugEnabled() bool {
	return debugEnabled
}

// WithStepLogging wraps a Middleware with start/end logs and duration.
// When ROUTER_DEBUG_MIDDLEWARE is not enabled, the original middleware
// is returned without additional behavior.
func WithStepLogging(name string, logger *zap.Logger, mw Middleware) Middleware {
	if !debugEnabled || mw == nil {
		return mw
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			return mw(next)
		}

		wrapped := mw(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()

			requestID := GetRequestID(ctx)
			tenantID := GetTenantID(ctx)
			tenantTier := GetTenantTier(ctx)

			logger.Info("middleware.step.start",
				zap.String("step", name),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("request_id", requestID),
				zap.String("tenant_id", tenantID),
				zap.String("tenant_tier", tenantTier),
			)

			wrapped.ServeHTTP(w, r)

			logger.Info("middleware.step.end",
				zap.String("step", name),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("request_id", requestID),
				zap.String("tenant_id", tenantID),
				zap.String("tenant_tier", tenantTier),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

