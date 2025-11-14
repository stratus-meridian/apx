package auth

// SimpleTenant represents a minimal tenant for the open-core edition
// This is a demonstration implementation - the commercial version uses
// a full multi-tenant model with organizations, products, and environments.
type SimpleTenant struct {
	ID   string
	Name string
	Tier string
	RPM  int // Requests per minute limit
}

// TenantOrganization provides basic organization info
type TenantOrganization struct {
	ID   string
	Name string
	Tier string
}

// TenantProduct provides basic product info
type TenantProduct struct {
	ID   string
	Name string
}

// TenantEnvironment provides basic environment info
type TenantEnvironment struct {
	ID   string
	Name string
}
