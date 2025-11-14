package responses

import (
	"testing"
	"time"

	"github.com/apx/control/pkg/ratelimit"
	"github.com/apx/control/tenant"
)

func TestNewPaymentRequired(t *testing.T) {
	tests := []struct {
		name         string
		tenant       *tenant.Tenant
		quotaStatus  *ratelimit.QuotaStatus
		wantError    string
		wantTier     string
		wantOverage  int64
		wantSuggested string
	}{
		{
			name: "Free tier quota exceeded",
			tenant: &tenant.Tenant{
				Organization: tenant.Organization{
					ID:   "test-org",
					Tier: tenant.TierFree,
					Billing: tenant.Billing{
						BalanceCents:   0,
						OverageAllowed: false,
					},
				},
			},
			quotaStatus: &ratelimit.QuotaStatus{
				TenantID:    "test-org",
				Tier:        ratelimit.TierFree,
				Limit:       10000,
				Used:        10147,
				Remaining:   0,
				IsExhausted: true,
				ResetAt:     time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			},
			wantError:     "quota_exceeded",
			wantTier:      "free",
			wantOverage:   147,
			wantSuggested: "pro",
		},
		{
			name: "Pro tier quota exceeded",
			tenant: &tenant.Tenant{
				Organization: tenant.Organization{
					ID:   "test-org",
					Tier: tenant.TierPro,
					Billing: tenant.Billing{
						BalanceCents:   0,
						OverageAllowed: true,
					},
				},
			},
			quotaStatus: &ratelimit.QuotaStatus{
				TenantID:    "test-org",
				Tier:        ratelimit.TierPro,
				Limit:       1000000,
				Used:        1050000,
				Remaining:   0,
				IsExhausted: true,
				ResetAt:     time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			},
			wantError:     "quota_exceeded",
			wantTier:      "pro",
			wantOverage:   50000,
			wantSuggested: "enterprise",
		},
		{
			name: "Free tier with balance",
			tenant: &tenant.Tenant{
				Organization: tenant.Organization{
					ID:   "test-org",
					Tier: tenant.TierFree,
					Billing: tenant.Billing{
						BalanceCents:   5000, // $50.00
						OverageAllowed: false,
					},
				},
			},
			quotaStatus: &ratelimit.QuotaStatus{
				TenantID:    "test-org",
				Tier:        ratelimit.TierFree,
				Limit:       10000,
				Used:        10500,
				Remaining:   0,
				IsExhausted: true,
				ResetAt:     time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			},
			wantError:     "quota_exceeded",
			wantTier:      "free",
			wantOverage:   500,
			wantSuggested: "pro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := NewPaymentRequired(tt.tenant, tt.quotaStatus)

			if response.Error != tt.wantError {
				t.Errorf("Error = %v, want %v", response.Error, tt.wantError)
			}

			if response.Tier != tt.wantTier {
				t.Errorf("Tier = %v, want %v", response.Tier, tt.wantTier)
			}

			if response.Overage != tt.wantOverage {
				t.Errorf("Overage = %v, want %v", response.Overage, tt.wantOverage)
			}

			if response.SuggestedTier != tt.wantSuggested {
				t.Errorf("SuggestedTier = %v, want %v", response.SuggestedTier, tt.wantSuggested)
			}

			if response.CurrentUsage != tt.quotaStatus.Used {
				t.Errorf("CurrentUsage = %v, want %v", response.CurrentUsage, tt.quotaStatus.Used)
			}

			if response.Limit != tt.quotaStatus.Limit {
				t.Errorf("Limit = %v, want %v", response.Limit, tt.quotaStatus.Limit)
			}

			if response.UpgradeURL == "" {
				t.Error("UpgradeURL should not be empty")
			}

			// Check balance info
			if tt.tenant.Organization.Billing.BalanceCents > 0 {
				if response.Balance == nil {
					t.Error("Balance should not be nil when tenant has balance")
				} else {
					if response.Balance.AvailableCents != tt.tenant.Organization.Billing.BalanceCents {
						t.Errorf("Balance.AvailableCents = %v, want %v",
							response.Balance.AvailableCents, tt.tenant.Organization.Billing.BalanceCents)
					}
				}
			}
		})
	}
}

func TestNewPaymentRequiredWithGracePeriod(t *testing.T) {
	graceEnds := time.Now().Add(12 * time.Hour)
	exhaustedAt := time.Now().Add(-12 * time.Hour)

	tenant := &tenant.Tenant{
		Organization: tenant.Organization{
			ID:   "test-org",
			Tier: tenant.TierPro,
			Billing: tenant.Billing{
				BalanceCents:   0,
				OverageAllowed: true,
			},
		},
	}

	quotaStatus := &ratelimit.QuotaStatus{
		TenantID:      "test-org",
		Tier:          ratelimit.TierPro,
		Limit:         1000000,
		Used:          1050000,
		Remaining:     0,
		IsExhausted:   true,
		ExhaustedAt:   &exhaustedAt,
		InGracePeriod: true,
		GraceEndsAt:   &graceEnds,
		ResetAt:       time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
	}

	response := NewPaymentRequired(tenant, quotaStatus)

	if response.GracePeriod == nil {
		t.Fatal("GracePeriod should not be nil")
	}

	if !response.GracePeriod.InGracePeriod {
		t.Error("InGracePeriod should be true")
	}

	if !response.GracePeriod.GraceEndsAt.Equal(graceEnds) {
		t.Errorf("GraceEndsAt = %v, want %v", response.GracePeriod.GraceEndsAt, graceEnds)
	}

	if response.GracePeriod.Message == "" {
		t.Error("Grace period message should not be empty")
	}
}

func TestSuggestTier(t *testing.T) {
	tests := []struct {
		name         string
		currentUsage int64
		currentTier  ratelimit.Tier
		wantTier     ratelimit.Tier
	}{
		{
			name:         "Free tier low usage - no upgrade",
			currentUsage: 5000,
			currentTier:  ratelimit.TierFree,
			wantTier:     ratelimit.TierFree,
		},
		{
			name:         "Free tier exceeded - suggest Pro",
			currentUsage: 15000,
			currentTier:  ratelimit.TierFree,
			wantTier:     ratelimit.TierPro,
		},
		{
			name:         "Pro tier low usage - no upgrade",
			currentUsage: 500000,
			currentTier:  ratelimit.TierPro,
			wantTier:     ratelimit.TierPro,
		},
		{
			name:         "Pro tier high usage - suggest Enterprise",
			currentUsage: 850000,
			currentTier:  ratelimit.TierPro,
			wantTier:     ratelimit.TierEnterprise,
		},
		{
			name:         "Enterprise tier - no upgrade",
			currentUsage: 5000000,
			currentTier:  ratelimit.TierEnterprise,
			wantTier:     ratelimit.TierEnterprise,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestTier(tt.currentUsage, tt.currentTier)
			if got != tt.wantTier {
				t.Errorf("SuggestTier() = %v, want %v", got, tt.wantTier)
			}
		})
	}
}

func TestGetTierPricing(t *testing.T) {
	tests := []struct {
		name            string
		tier            ratelimit.Tier
		wantMonthlyPrice int64
		wantMonthlyQuota int64
	}{
		{
			name:            "Free tier",
			tier:            ratelimit.TierFree,
			wantMonthlyPrice: 0,
			wantMonthlyQuota: 10000,
		},
		{
			name:            "Pro tier",
			tier:            ratelimit.TierPro,
			wantMonthlyPrice: 4900,
			wantMonthlyQuota: 1000000,
		},
		{
			name:            "Enterprise tier",
			tier:            ratelimit.TierEnterprise,
			wantMonthlyPrice: 0,
			wantMonthlyQuota: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := getTierPricing(tt.tier)

			if pricing.MonthlyPrice != tt.wantMonthlyPrice {
				t.Errorf("MonthlyPrice = %v, want %v", pricing.MonthlyPrice, tt.wantMonthlyPrice)
			}

			if pricing.MonthlyQuota != tt.wantMonthlyQuota {
				t.Errorf("MonthlyQuota = %v, want %v", pricing.MonthlyQuota, tt.wantMonthlyQuota)
			}

			if pricing.Tier != string(tt.tier) {
				t.Errorf("Tier = %v, want %v", pricing.Tier, string(tt.tier))
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		want string
	}{
		{"Small number", 100, "100"},
		{"Thousands", 1000, "1,000"},
		{"Millions", 1000000, "1,000,000"},
		{"Unlimited", -1, "unlimited"},
		{"Zero", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNumber(tt.n)
			if got != tt.want {
				t.Errorf("formatNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatCentsToUSD(t *testing.T) {
	tests := []struct {
		name  string
		cents int64
		want  string
	}{
		{"Zero", 0, "$0.00"},
		{"Dollars", 4900, "$49.00"},
		{"Cents", 50, "$0.50"},
		{"Large amount", 123456, "$1234.56"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCentsToUSD(tt.cents)
			if got != tt.want {
				t.Errorf("formatCentsToUSD() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"Seconds", 45 * time.Second, "45s"},
		{"Minutes", 5 * time.Minute, "5m 0s"},
		{"Hours", 2 * time.Hour, "2h 0m"},
		{"Mixed", 2*time.Hour + 30*time.Minute + 15*time.Second, "2h 30m"},
		{"Negative", -10 * time.Second, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.d)
			if got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildUpgradeURL(t *testing.T) {
	tests := []struct {
		name         string
		currentTier  string
		currentUsage int64
		wantContains []string
	}{
		{
			name:         "Free tier",
			currentTier:  "free",
			currentUsage: 15000,
			wantContains: []string{"current=free", "suggested=pro", "from=quota_exceeded"},
		},
		{
			name:         "Pro tier",
			currentTier:  "pro",
			currentUsage: 900000,
			wantContains: []string{"current=pro", "suggested=enterprise", "from=quota_exceeded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildUpgradeURL(tt.currentTier, tt.currentUsage)
			for _, want := range tt.wantContains {
				if !contains(got, want) {
					t.Errorf("buildUpgradeURL() = %v, want to contain %v", got, want)
				}
			}
		})
	}
}

func TestNewPaymentRequiredWithBalance(t *testing.T) {
	tenant := &tenant.Tenant{
		Organization: tenant.Organization{
			ID:   "test-org",
			Tier: tenant.TierFree,
			Billing: tenant.Billing{
				BalanceCents:   1000, // Original balance
				OverageAllowed: false,
			},
		},
	}

	quotaStatus := &ratelimit.QuotaStatus{
		TenantID:    "test-org",
		Tier:        ratelimit.TierFree,
		Limit:       10000,
		Used:        10500,
		Remaining:   0,
		IsExhausted: true,
		ResetAt:     time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
	}

	// Override balance from ledger service
	ledgerBalance := int64(5000)
	response := NewPaymentRequiredWithBalance(tenant, quotaStatus, ledgerBalance)

	if response.Balance == nil {
		t.Fatal("Balance should not be nil")
	}

	if response.Balance.AvailableCents != ledgerBalance {
		t.Errorf("Balance.AvailableCents = %v, want %v", response.Balance.AvailableCents, ledgerBalance)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
