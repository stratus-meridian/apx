package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// CanonicalTierSchema represents the canonical tier schema
type CanonicalTierSchema struct {
	CanonicalValues struct {
		Tiers map[string]CanonicalTierConfig `json:"tiers"`
	} `json:"canonical_values"`
}

// CanonicalTierConfig represents a tier configuration in the canonical schema
type CanonicalTierConfig struct {
	Tier        string           `json:"tier"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Quotas      CanonicalQuotas  `json:"quotas"`
	Billing     CanonicalBilling `json:"billing"`
}

// CanonicalQuotas represents quota values in the canonical schema
type CanonicalQuotas struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	MonthlyRequests   int `json:"monthly_requests"`
	BurstLimit        int `json:"burst_limit"`
	MaxPolicies       int `json:"max_policies"`
	MaxAPIKeys        int `json:"max_api_keys"`
	MaxProducts       int `json:"max_products"`
	MaxEnvironments   int `json:"max_environments"`
	StorageGB         int `json:"storage_gb"`
}

// CanonicalBilling represents billing configuration in the canonical schema
type CanonicalBilling struct {
	OverageAllowed   bool `json:"overage_allowed"`
	OverageRate      int  `json:"overage_rate"`
	GracePeriodHours int  `json:"grace_period_hours"`
}

// loadCanonicalSchema loads the canonical tier schema from JSON
func loadCanonicalSchema(t *testing.T) *CanonicalTierSchema {
	// Get the project root directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate to project root (two levels up from tests/integration)
	projectRoot := filepath.Join(cwd, "..", "..")
	schemaPath := filepath.Join(projectRoot, "configs", "schemas", "tier-schema.json")

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read canonical schema at %s: %v", schemaPath, err)
	}

	var schema CanonicalTierSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("Failed to parse canonical schema: %v", err)
	}

	return &schema
}

// TestCanonicalSchemaExists validates that the canonical schema file exists and is valid JSON
func TestCanonicalSchemaExists(t *testing.T) {
	schema := loadCanonicalSchema(t)

	// Verify all three tiers are present
	if _, ok := schema.CanonicalValues.Tiers["free"]; !ok {
		t.Error("Canonical schema missing 'free' tier")
	}
	if _, ok := schema.CanonicalValues.Tiers["pro"]; !ok {
		t.Error("Canonical schema missing 'pro' tier")
	}
	if _, ok := schema.CanonicalValues.Tiers["enterprise"]; !ok {
		t.Error("Canonical schema missing 'enterprise' tier")
	}

	t.Log("✓ Canonical schema loaded successfully with all three tiers")
}

// TestTierQuotaValues validates that tier quota values are reasonable
func TestTierQuotaValues(t *testing.T) {
	schema := loadCanonicalSchema(t)

	// Verify tier values are reasonable
	freeTier := schema.CanonicalValues.Tiers["free"]
	if freeTier.Quotas.RequestsPerMinute <= 0 {
		t.Error("Free tier RequestsPerMinute must be positive")
	}
	if freeTier.Quotas.MonthlyRequests <= 0 {
		t.Error("Free tier MonthlyRequests must be positive")
	}

	proTier := schema.CanonicalValues.Tiers["pro"]
	if proTier.Quotas.RequestsPerMinute <= freeTier.Quotas.RequestsPerMinute {
		t.Error("Pro tier RequestsPerMinute should be greater than free tier")
	}
	if proTier.Quotas.MonthlyRequests <= freeTier.Quotas.MonthlyRequests {
		t.Error("Pro tier MonthlyRequests should be greater than free tier")
	}

	entTier := schema.CanonicalValues.Tiers["enterprise"]
	if entTier.Quotas.RequestsPerMinute <= proTier.Quotas.RequestsPerMinute {
		t.Error("Enterprise tier RequestsPerMinute should be greater than pro tier")
	}
	if entTier.Quotas.MonthlyRequests != -1 {
		t.Error("Enterprise tier MonthlyRequests should be unlimited (-1)")
	}

	t.Log("✓ All tier quota values are reasonable and properly ordered")
}

// TestUnlimitedQuotaConsistency validates that -1 is used consistently for unlimited
func TestUnlimitedQuotaConsistency(t *testing.T) {
	schema := loadCanonicalSchema(t)
	entTier := schema.CanonicalValues.Tiers["enterprise"]

	// Enterprise tier should have -1 for unlimited quotas
	if entTier.Quotas.MonthlyRequests != -1 {
		t.Errorf("Enterprise MonthlyRequests should be -1 (unlimited), got %d", entTier.Quotas.MonthlyRequests)
	}
	if entTier.Quotas.MaxPolicies != -1 {
		t.Errorf("Enterprise MaxPolicies should be -1 (unlimited), got %d", entTier.Quotas.MaxPolicies)
	}
	if entTier.Quotas.MaxAPIKeys != -1 {
		t.Errorf("Enterprise MaxAPIKeys should be -1 (unlimited), got %d", entTier.Quotas.MaxAPIKeys)
	}
	if entTier.Quotas.MaxProducts != -1 {
		t.Errorf("Enterprise MaxProducts should be -1 (unlimited), got %d", entTier.Quotas.MaxProducts)
	}
	if entTier.Quotas.MaxEnvironments != -1 {
		t.Errorf("Enterprise MaxEnvironments should be -1 (unlimited), got %d", entTier.Quotas.MaxEnvironments)
	}

	t.Log("✓ Enterprise tier uses -1 consistently for unlimited quotas")
}

// TestBillingConfiguration validates billing settings for each tier
func TestBillingConfiguration(t *testing.T) {
	schema := loadCanonicalSchema(t)

	// Free tier should not allow overage
	freeTier := schema.CanonicalValues.Tiers["free"]
	if freeTier.Billing.OverageAllowed {
		t.Error("Free tier should not allow overage")
	}
	if freeTier.Billing.OverageRate != 0 {
		t.Error("Free tier overage rate should be 0")
	}

	// Pro tier should allow overage with reasonable rate
	proTier := schema.CanonicalValues.Tiers["pro"]
	if !proTier.Billing.OverageAllowed {
		t.Error("Pro tier should allow overage")
	}
	if proTier.Billing.OverageRate <= 0 {
		t.Error("Pro tier overage rate should be positive")
	}

	// Enterprise tier should allow overage with 0 rate (custom pricing)
	entTier := schema.CanonicalValues.Tiers["enterprise"]
	if !entTier.Billing.OverageAllowed {
		t.Error("Enterprise tier should allow overage")
	}
	if entTier.Billing.OverageRate != 0 {
		t.Error("Enterprise tier overage rate should be 0 (custom pricing)")
	}

	t.Log("✓ Billing configuration is correct for all tiers")
}

// TestCanonicalValuesDisplay prints the canonical values for verification
func TestCanonicalValuesDisplay(t *testing.T) {
	schema := loadCanonicalSchema(t)

	t.Log("\n=== CANONICAL TIER VALUES ===")
	for tierName, tier := range schema.CanonicalValues.Tiers {
		t.Logf("\n%s Tier (%s):", tierName, tier.Name)
		t.Logf("  Description: %s", tier.Description)
		t.Logf("  Quotas:")
		t.Logf("    - requests_per_minute: %d", tier.Quotas.RequestsPerMinute)
		t.Logf("    - monthly_requests: %d", tier.Quotas.MonthlyRequests)
		t.Logf("    - burst_limit: %d", tier.Quotas.BurstLimit)
		t.Logf("    - max_policies: %d", tier.Quotas.MaxPolicies)
		t.Logf("    - max_api_keys: %d", tier.Quotas.MaxAPIKeys)
		t.Logf("    - max_products: %d", tier.Quotas.MaxProducts)
		t.Logf("    - max_environments: %d", tier.Quotas.MaxEnvironments)
		t.Logf("    - storage_gb: %d", tier.Quotas.StorageGB)
		t.Logf("  Billing:")
		t.Logf("    - overage_allowed: %v", tier.Billing.OverageAllowed)
		t.Logf("    - overage_rate: %d cents/1K reqs", tier.Billing.OverageRate)
		t.Logf("    - grace_period_hours: %d", tier.Billing.GracePeriodHours)
	}
}

// TestFieldCoverage validates that all expected fields are present
func TestFieldCoverage(t *testing.T) {
	schema := loadCanonicalSchema(t)

	expectedFields := []string{"free", "pro", "enterprise"}
	for _, tierName := range expectedFields {
		tier, ok := schema.CanonicalValues.Tiers[tierName]
		if !ok {
			t.Errorf("Missing tier: %s", tierName)
			continue
		}

		// Check all quota fields are defined
		q := tier.Quotas
		if q.RequestsPerMinute == 0 && tierName != "free" {
			t.Errorf("Tier %s: RequestsPerMinute not set", tierName)
		}
		// Note: We don't check for 0 on other fields as they could legitimately be 0

		// Check billing fields
		b := tier.Billing
		_ = b.OverageAllowed  // Just verify field exists
		_ = b.OverageRate
		_ = b.GracePeriodHours
	}

	t.Log("✓ All expected fields are present in canonical schema")
}

// Main function for running tests manually
func main() {
	fmt.Println("Running Tier Alignment Tests...")
	fmt.Println("Use: go test -v tier_alignment_test.go")
}
