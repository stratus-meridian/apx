package middleware

import (
	"context"
	"net/http"

	"github.com/stratus-meridian/apx-router-open-core/internal/auth"
	"go.uber.org/zap"
)

// Context keys for storing tenant information
type contextKey string

const tenantContextKey contextKey = "tenant"

// TenantContextMiddleware resolves and injects tenant information into the request context.
// This middleware extracts the API key from the Authorization header and resolves
// the tenant using the provided TenantResolver.
type TenantContextMiddleware struct {
	resolver auth.TenantResolver
	logger   *zap.Logger
}

// NewTenantContextMiddleware creates a new tenant context middleware.
func NewTenantContextMiddleware(resolver auth.TenantResolver, logger *zap.Logger) *TenantContextMiddleware {
	return &TenantContextMiddleware{
		resolver: resolver,
		logger:   logger,
	}
}

// Handler returns the middleware handler function.
func (m *TenantContextMiddleware) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract API key from Authorization header
			// Format: "Bearer <api-key>"
			apiKey := extractAPIKey(r)

			var tenant *auth.SimpleTenant
			var err error

			if apiKey != "" {
				// Resolve tenant from API key
				tenant, err = m.resolver.ResolveTenant(ctx, apiKey)
				if err != nil {
					m.logger.Warn("failed to resolve tenant, using default",
						zap.Error(err),
						zap.String("path", r.URL.Path))
					tenant = m.resolver.GetDefaultTenant(ctx)
				}
			} else {
				// No API key provided, use default tenant
				tenant = m.resolver.GetDefaultTenant(ctx)
			}

			// Add tenant to request context
			ctx = context.WithValue(ctx, tenantContextKey, tenant)
			r = r.WithContext(ctx)

			m.logger.Debug("tenant resolved",
				zap.String("tenant_id", tenant.ID),
				zap.String("tier", tenant.Tier),
				zap.String("path", r.URL.Path))

			next.ServeHTTP(w, r)
		})
	}
}

// extractAPIKey extracts the API key from the Authorization header.
// Expected format: "Bearer <api-key>"
func extractAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	// Check for "Bearer " prefix
	const prefix = "Bearer "
	if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
		return auth[len(prefix):]
	}

	// Also support X-API-Key header (common alternative)
	return r.Header.Get("X-API-Key")
}

// GetTenant retrieves the tenant from the request context.
// Returns (tenant, true) if found, (nil, false) if not found.
func GetTenant(ctx context.Context) (*auth.SimpleTenant, bool) {
	tenant, ok := ctx.Value(tenantContextKey).(*auth.SimpleTenant)
	return tenant, ok
}

// TenantContext is a convenience function that creates and returns tenant context middleware.
func TenantContext(resolver auth.TenantResolver, logger *zap.Logger) Middleware {
	middleware := NewTenantContextMiddleware(resolver, logger)
	return middleware.Handler()
}
