package middleware

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

const (
	TenantIDKey   contextKey = "tenant_id"
	TenantTierKey contextKey = "tenant_tier"
)

// TenantContext extracts tenant information from headers
// (previously set by Envoy WASM filter or API key lookup)
func TenantContext(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract tenant ID from header (set by edge WASM filter)
			tenantID := r.Header.Get("X-Tenant-ID")
			tenantTier := r.Header.Get("X-Tenant-Tier")

			// If not present, extract from JWT payload or API key
			if tenantID == "" {
				// TODO: Implement JWT/API key extraction
				tenantID = "unknown"
				tenantTier = "free"
			}

			// Add to request context
			ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, TenantTierKey, tenantTier)

			// Add to response headers for debugging
			w.Header().Set("X-Tenant-ID", tenantID)
			w.Header().Set("X-Tenant-Tier", tenantTier)

			logger.Debug("tenant context extracted",
				zap.String("tenant_id", tenantID),
				zap.String("tenant_tier", tenantTier),
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTenantID retrieves tenant ID from request context
func GetTenantID(ctx context.Context) string {
	if tenantID, ok := ctx.Value(TenantIDKey).(string); ok {
		return tenantID
	}
	return ""
}

// GetTenantTier retrieves tenant tier from request context
func GetTenantTier(ctx context.Context) string {
	if tenantTier, ok := ctx.Value(TenantTierKey).(string); ok {
		return tenantTier
	}
	return "free"
}
