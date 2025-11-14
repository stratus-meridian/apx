package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/stratus-meridian/apx-private/control/tenant"
	pkgauth "github.com/stratus-meridian/apx/router/pkg/auth"
	"go.uber.org/zap"
)

const (
	TenantIDKey      contextKey = "tenant_id"
	TenantTierKey    contextKey = "tenant_tier"
	TenantContextKey contextKey = "apx.tenant"
)

// TenantResolver is the interface for resolving tenants from API keys
type TenantResolver interface {
	ResolveTenant(ctx context.Context, apiKey string) (*tenant.Tenant, error)
	GetDefaultTenant(ctx context.Context) *tenant.Tenant
}

// TenantContext extracts tenant information from API keys (secure resolution)
// SECURITY: This middleware does NOT trust client-supplied X-Tenant-* headers.
// Tenant context is resolved exclusively from validated API keys.
func TenantContext(resolver TenantResolver, logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			var tenantCtx *tenant.Tenant

			// Extract API key from Authorization header
			apiKey, err := pkgauth.ExtractAPIKey(r)
			if err == nil && apiKey != "" {
				// Resolve tenant from API key
				tenantCtx, err = resolver.ResolveTenant(ctx, apiKey)
				if err != nil {
					// Invalid or not found API key -> return 401 Unauthorized
					logger.Warn("failed to resolve tenant from API key",
						zap.Error(err),
						zap.String("path", r.URL.Path))

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"unauthorized","message":"Invalid or expired API key"}`))
					return
				}

				logger.Debug("tenant resolved from API key",
					zap.String("tenant_id", tenantCtx.ResourceID),
					zap.String("tier", string(tenantCtx.Organization.Tier)),
					zap.String("org_id", tenantCtx.Organization.ID))
			} else {
				// No API key provided -> use default tenant with restrictive quotas
				tenantCtx = resolver.GetDefaultTenant(ctx)
				logger.Debug("using default tenant (no API key provided)")
			}

			// Add to request context
			ctx = context.WithValue(ctx, TenantIDKey, tenantCtx.ResourceID)
			ctx = context.WithValue(ctx, TenantTierKey, string(tenantCtx.Organization.Tier))
			ctx = context.WithValue(ctx, TenantContextKey, tenantCtx)

			// Add to response headers for debugging
			w.Header().Set("X-Tenant-ID", tenantCtx.ResourceID)
			w.Header().Set("X-Tenant-Tier", string(tenantCtx.Organization.Tier))

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
