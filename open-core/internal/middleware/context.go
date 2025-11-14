package middleware

import (
	"context"
)

// Context keys for propagating metadata through middleware chain
const (
	tenantIDKey      contextKey = "tenant_id"
	tenantTierKey    contextKey = "tenant_tier"
	policyVersionKey contextKey = "policy_version"
	regionKey        contextKey = "region"
)

// GetTenantID retrieves the tenant ID from context
func GetTenantID(ctx context.Context) string {
	// Try to get from tenant object first
	if tenant, ok := GetTenant(ctx); ok && tenant != nil {
		return tenant.ID
	}
	// Fallback to direct context value
	if val := ctx.Value(tenantIDKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetTenantTier retrieves the tenant tier from context
func GetTenantTier(ctx context.Context) string {
	// Try to get from tenant object first
	if tenant, ok := GetTenant(ctx); ok && tenant != nil {
		return tenant.Tier
	}
	// Fallback to direct context value
	if val := ctx.Value(tenantTierKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetPolicyVersion retrieves the policy version from context
func GetPolicyVersion(ctx context.Context) string {
	if val := ctx.Value(policyVersionKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return "default"
}

// GetRegion retrieves the region from context
func GetRegion(ctx context.Context) string {
	if val := ctx.Value(regionKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return "us-central1" // Default region
}
