package middleware

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// HeaderTenantID is the header for tenant identification
	HeaderTenantID = "X-Tenant-ID"

	// ContextKeyCanary is the context key for canary status
	ContextKeyCanary contextKey = "apx.canary.status"

	// ContextKeyCanaryVersion is the context key for selected canary version
	ContextKeyCanaryVersion contextKey = "apx.canary.version"

	// PolicyVersionKey is the context key for the selected policy version
	// This is used by other middleware for backward compatibility
	PolicyVersionKey contextKey = "policy_version"
)

// CanaryDecider is a function type that decides canary routing
// Returns: (version string, isCanary bool, error)
type CanaryDecider func(ctx context.Context, policyName, tenantID string) (string, bool, error)

// Canary middleware for traffic splitting with consistent hashing
type Canary struct {
	// CanaryDecider is a function that decides canary routing
	// In production, this would use the splitter from .private/control/canary
	CanaryDecider CanaryDecider
}

// NewCanary creates new canary middleware
func NewCanary(decider CanaryDecider) *Canary {
	return &Canary{
		CanaryDecider: decider,
	}
}

// Handler returns HTTP middleware
func (c *Canary) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract tenant ID
		tenantID := r.Header.Get(HeaderTenantID)
		if tenantID == "" {
			// Default tenant for requests without tenant ID
			tenantID = "default"
		}

		// Get policy version from context (set by PolicyVersion middleware)
		policyVersion := GetVersionFromContext(r.Context())

		// If version is "latest", check canary config
		if policyVersion == "latest" && c.CanaryDecider != nil {
			// Policy name would come from request routing
			// For now, use a placeholder
			policyName := "default-policy"

			version, isCanary, err := c.CanaryDecider(r.Context(), policyName, tenantID)
			if err == nil {
				// Update policy version in context
				ctx := context.WithValue(r.Context(), ContextKeyPolicyVersion, version)
				ctx = context.WithValue(ctx, ContextKeyCanary, isCanary)
				ctx = context.WithValue(ctx, ContextKeyCanaryVersion, version)
				r = r.WithContext(ctx)

				// Add canary header to response
				if isCanary {
					w.Header().Set("X-Apx-Canary", "true")
					w.Header().Set("X-Apx-Canary-Version", version)
				} else {
					w.Header().Set("X-Apx-Canary", "false")
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// IsCanary checks if current request is using canary version
func IsCanary(ctx context.Context) bool {
	if isCanary, ok := ctx.Value(ContextKeyCanary).(bool); ok {
		return isCanary
	}
	return false
}

// GetCanaryVersion extracts the canary version from request context
func GetCanaryVersion(ctx context.Context) string {
	if version, ok := ctx.Value(ContextKeyCanaryVersion).(string); ok {
		return version
	}
	return ""
}

// GetPolicyVersion extracts the selected policy version from request context
// This is for backward compatibility with other middleware
func GetPolicyVersion(ctx context.Context) string {
	if version, ok := ctx.Value(PolicyVersionKey).(string); ok {
		return version
	}
	return ""
}

// SetPolicyVersion stores the selected policy version in context
func SetPolicyVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, PolicyVersionKey, version)
}
