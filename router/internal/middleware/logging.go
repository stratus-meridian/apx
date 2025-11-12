package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Logging logs HTTP requests with all propagated headers
func Logging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Extract headers from context (added by previous middleware)
			requestID := GetRequestID(r.Context())
			tenantID := GetTenantID(r.Context())
			tenantTier := GetTenantTier(r.Context())
			policyVersion := GetPolicyVersion(r.Context())
			region := GetRegion(r.Context())

			// Call next handler
			next.ServeHTTP(w, r)

			// Log request with all propagated headers
			logger.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", time.Since(start)),
				zap.String("request_id", requestID),
				zap.String("tenant_id", tenantID),
				zap.String("tenant_tier", tenantTier),
				zap.String("policy_version", policyVersion),
				zap.String("region", region),
			)
		})
	}
}
