package auth

import (
	"context"
)

// SimpleTenantResolver is a demonstration tenant resolver that doesn't require
// external dependencies (no Firestore, no database).
//
// WARNING: This implementation is for demonstration purposes only and should
// NOT be used in production. It treats all API keys as valid and returns a
// default tenant. The commercial version uses Firestore with proper security.
type SimpleTenantResolver struct {
	defaultTenant *SimpleTenant
}

// NewSimpleTenantResolver creates a new simple tenant resolver with a default tenant.
func NewSimpleTenantResolver() *SimpleTenantResolver {
	return &SimpleTenantResolver{
		defaultTenant: &SimpleTenant{
			ID:   "demo-tenant-001",
			Name: "Demo Tenant",
			Tier: "free",
			RPM:  10, // 10 requests per minute for demo
		},
	}
}

// ResolveTenant returns the default tenant for any API key.
// In the commercial version, this would:
// - Validate the API key against Firestore
// - Check tenant status (active, suspended, deleted)
// - Load organization, product, and environment context
// - Apply security policies
func (r *SimpleTenantResolver) ResolveTenant(ctx context.Context, apiKey string) (*SimpleTenant, error) {
	// Demo mode: always return default tenant
	// Real implementation would validate apiKey against a secure store
	return r.defaultTenant, nil
}

// GetDefaultTenant returns a restrictive default tenant for unauthenticated requests.
func (r *SimpleTenantResolver) GetDefaultTenant(ctx context.Context) *SimpleTenant {
	return r.defaultTenant
}

// Close releases any resources (none in this simple implementation).
func (r *SimpleTenantResolver) Close() error {
	return nil
}
