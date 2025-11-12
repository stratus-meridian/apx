package middleware

import (
	"context"
	"net/http"
	"os"

	"github.com/apx/router/internal/policy"
	"go.uber.org/zap"
)

const (
	RegionKey contextKey = "region"
)

// PolicyVersionTag adds the active policy version and region to request context
func PolicyVersionTag(store *policy.Store, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For V-002, use a default policy version
			// In production, this would be retrieved from the policy store based on tenant
			policyVersion := "default@1.0.0"

			// Get region from environment (set in config)
			region := os.Getenv("GCP_REGION")
			if region == "" {
				region = "us-central1"
			}

			// Add to context
			ctx := context.WithValue(r.Context(), PolicyVersionKey, policyVersion)
			ctx = context.WithValue(ctx, RegionKey, region)

			// Add to response headers for verification
			w.Header().Set("X-Policy-Version", policyVersion)
			w.Header().Set("X-Region", region)

			logger.Debug("policy and region context added",
				zap.String("policy_version", policyVersion),
				zap.String("region", region),
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRegion retrieves region from context
func GetRegion(ctx context.Context) string {
	if region, ok := ctx.Value(RegionKey).(string); ok {
		return region
	}
	return ""
}
