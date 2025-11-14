package auth

import (
	"context"
	"errors"
)

// Common errors for tenant resolution
var (
	ErrTenantNotFound     = errors.New("tenant not found for API key")
	ErrInvalidTenantState = errors.New("tenant is not in active state")
)

// TenantResolver defines the interface for resolving tenants from API keys.
// Implementations should be secure and never trust client-supplied headers.
//
// Open-core implementation: SimpleTenantResolver (header-based, demo only)
// Commercial implementation: FirestoreTenantResolver (secure, production-grade)
type TenantResolver interface {
	// ResolveTenant resolves a tenant from an API key.
	// Returns the complete tenant context.
	// Returns ErrTenantNotFound if the API key is not found.
	// Returns ErrInvalidTenantState if the tenant is suspended or deleted.
	ResolveTenant(ctx context.Context, apiKey string) (*SimpleTenant, error)

	// GetDefaultTenant returns a restrictive default tenant for requests without API keys.
	// The default tenant has "free" tier quotas and limited access.
	GetDefaultTenant(ctx context.Context) *SimpleTenant

	// Close releases any resources held by the resolver
	Close() error
}
