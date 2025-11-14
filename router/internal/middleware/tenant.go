package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/apx/control/tenant"
	"go.uber.org/zap"
)

const (
	TenantIDKey      contextKey = "tenant_id"
	TenantTierKey    contextKey = "tenant_tier"
	TenantContextKey contextKey = "apx.tenant"
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

			// Build lightweight tenant context for downstream middleware
			tenantCtx := buildTenantContext(tenantID, tenantTier)

			// Add to response headers for debugging
			w.Header().Set("X-Tenant-ID", tenantID)
			w.Header().Set("X-Tenant-Tier", tenantTier)

			logger.Debug("tenant context extracted",
				zap.String("tenant_id", tenantID),
				zap.String("tenant_tier", tenantTier),
			)

			ctx = context.WithValue(ctx, TenantContextKey, tenantCtx)

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

// GetTenant extracts the full tenant context if available.
// Falls back to constructing a minimal tenant object using headers.
func GetTenant(ctx context.Context) (*tenant.Tenant, bool) {
	if t, ok := ctx.Value(TenantContextKey).(*tenant.Tenant); ok && t != nil {
		return t, true
	}

	tenantID := GetTenantID(ctx)
	if tenantID == "" {
		return nil, false
	}

	tenantTier := GetTenantTier(ctx)
	return buildTenantContext(tenantID, tenantTier), true
}

func buildTenantContext(tenantID, tenantTier string) *tenant.Tenant {
	if tenantID == "" {
		tenantID = "unknown"
	}

	tier := tenant.Tier(strings.ToLower(tenantTier))
	if tier == "" {
		tier = tenant.TierFree
	}

	resourceID := tenantID

	return &tenant.Tenant{
		Organization: tenant.Organization{
			ID:     tenantID,
			Name:   tenantID,
			Tier:   tier,
			Status: tenant.StatusActive,
			Quotas: tenant.GetDefaultQuotas(tier),
		},
		Product: tenant.Product{
			ID:    "default",
			OrgID: tenantID,
			Name:  "Default",
		},
		Environment: tenant.Environment{
			ID:         "default",
			OrgID:      tenantID,
			ProductID:  "default",
			ResourceID: resourceID,
			Name:       "Default",
			Type:       tenant.EnvProduction,
			Status:     tenant.StatusActive,
		},
		ResourceID: resourceID,
	}
}
