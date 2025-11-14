package responses

import (
	"fmt"
	"time"

	"github.com/stratus-meridian/apx-private/control/pkg/ratelimit"
	"github.com/stratus-meridian/apx-private/control/tenant"
)

// PaymentRequiredResponse represents an HTTP 402 Payment Required response
// This is returned when a tenant's quota is exhausted and overage is not allowed
type PaymentRequiredResponse struct {
	Error         string        `json:"error"`                    // Error code (e.g., "quota_exceeded")
	Message       string        `json:"message"`                  // Human-readable error message
	Tier          string        `json:"tier"`                     // Current subscription tier
	CurrentUsage  int64         `json:"current_usage"`            // Current usage this month
	Limit         int64         `json:"limit"`                    // Monthly quota limit
	Overage       int64         `json:"overage"`                  // Amount over quota
	ResetAt       time.Time     `json:"reset_at"`                 // When quota resets (end of month)
	UpgradeURL    string        `json:"upgrade_url"`              // URL to upgrade subscription
	PaymentURL    string        `json:"payment_url,omitempty"`    // URL to add payment method (for P2.5-W2-T10)
	SuggestedTier string        `json:"suggested_tier,omitempty"` // Recommended tier upgrade
	Pricing       *TierPricing  `json:"pricing,omitempty"`        // Pricing info for suggested tier
	Balance       *BalanceInfo  `json:"balance,omitempty"`        // Account balance info (for P2.5-W2-T8)
	GracePeriod   *GracePeriod  `json:"grace_period,omitempty"`   // Grace period info (if applicable)
}

// TierPricing represents pricing information for a tier
type TierPricing struct {
	Tier          string `json:"tier"`            // Tier name
	MonthlyPrice  int64  `json:"monthly_price"`   // Monthly price in cents
	MonthlyQuota  int64  `json:"monthly_quota"`   // Monthly request quota
	OverageRate   int    `json:"overage_rate"`    // Overage rate per 1000 requests (cents)
	RequestsPerMin int64 `json:"requests_per_min"` // Rate limit (RPM)
}

// BalanceInfo represents account balance information
type BalanceInfo struct {
	AvailableCents int64  `json:"available_cents"` // Available balance in cents
	AvailableUSD   string `json:"available_usd"`   // Available balance formatted as USD
}

// GracePeriod represents grace period information
type GracePeriod struct {
	InGracePeriod bool      `json:"in_grace_period"` // Whether currently in grace period
	GraceEndsAt   time.Time `json:"grace_ends_at"`   // When grace period ends
	Message       string    `json:"message"`         // Grace period message
}

// NewPaymentRequired creates a new HTTP 402 Payment Required response
func NewPaymentRequired(tenant *tenant.Tenant, quotaStatus *ratelimit.QuotaStatus) *PaymentRequiredResponse {
	tier := string(tenant.Organization.Tier)
	overage := int64(0)
	if quotaStatus.Used > quotaStatus.Limit {
		overage = quotaStatus.Used - quotaStatus.Limit
	}

	// Build the error message
	message := fmt.Sprintf("Monthly quota of %s requests exceeded for %s tier. Please upgrade to continue.",
		formatNumber(quotaStatus.Limit),
		formatTierName(tier))

	response := &PaymentRequiredResponse{
		Error:        "quota_exceeded",
		Message:      message,
		Tier:         tier,
		CurrentUsage: quotaStatus.Used,
		Limit:        quotaStatus.Limit,
		Overage:      overage,
		ResetAt:      quotaStatus.ResetAt,
		UpgradeURL:   buildUpgradeURL(tier, quotaStatus.Used),
	}

	// Add suggested tier and pricing
	suggestedTier := SuggestTier(quotaStatus.Used, ratelimit.Tier(tier))
	if suggestedTier != ratelimit.Tier(tier) {
		response.SuggestedTier = string(suggestedTier)
		response.Pricing = getTierPricing(suggestedTier)
	}

	// Add grace period info if applicable
	if quotaStatus.InGracePeriod && quotaStatus.GraceEndsAt != nil {
		response.GracePeriod = &GracePeriod{
			InGracePeriod: true,
			GraceEndsAt:   *quotaStatus.GraceEndsAt,
			Message:       fmt.Sprintf("Grace period ends in %s", formatDuration(time.Until(*quotaStatus.GraceEndsAt))),
		}
	}

	// Add balance info if available (placeholder for P2.5-W2-T8)
	if tenant.Organization.Billing.BalanceCents > 0 {
		response.Balance = &BalanceInfo{
			AvailableCents: tenant.Organization.Billing.BalanceCents,
			AvailableUSD:   formatCentsToUSD(tenant.Organization.Billing.BalanceCents),
		}
		// Update message to include balance
		response.Message += fmt.Sprintf(" You have %s in account balance.", response.Balance.AvailableUSD)
	}

	return response
}

// NewPaymentRequiredWithBalance creates a payment required response with balance checking
// This is a placeholder for P2.5-W2-T8 (Unified Ledger Service)
func NewPaymentRequiredWithBalance(tenant *tenant.Tenant, quotaStatus *ratelimit.QuotaStatus, balanceCents int64) *PaymentRequiredResponse {
	response := NewPaymentRequired(tenant, quotaStatus)

	// Override balance with provided value (from ledger service)
	if balanceCents > 0 {
		response.Balance = &BalanceInfo{
			AvailableCents: balanceCents,
			AvailableUSD:   formatCentsToUSD(balanceCents),
		}
	}

	return response
}

// SuggestTier suggests an appropriate tier based on usage
func SuggestTier(currentUsage int64, currentTier ratelimit.Tier) ratelimit.Tier {
	// If Free tier and over limit, suggest Pro
	if currentTier == ratelimit.TierFree && currentUsage > 10000 {
		return ratelimit.TierPro
	}

	// If Pro tier and consistently high usage (80% of quota), suggest Enterprise
	if currentTier == ratelimit.TierPro && currentUsage > 800000 {
		return ratelimit.TierEnterprise
	}

	// No upgrade needed
	return currentTier
}

// getTierPricing returns pricing information for a tier
func getTierPricing(tier ratelimit.Tier) *TierPricing {
	config := ratelimit.GetTierConfig(tier)

	// Pricing in cents per month
	var monthlyPrice int64
	switch tier {
	case ratelimit.TierFree:
		monthlyPrice = 0
	case ratelimit.TierPro:
		monthlyPrice = 4900 // $49.00/month
	case ratelimit.TierEnterprise:
		monthlyPrice = 0 // Custom pricing
	}

	return &TierPricing{
		Tier:           string(tier),
		MonthlyPrice:   monthlyPrice,
		MonthlyQuota:   config.MonthlyQuota,
		OverageRate:    config.OverageRate,
		RequestsPerMin: config.RequestsPerMinute,
	}
}

// buildUpgradeURL constructs the upgrade URL with query parameters
func buildUpgradeURL(currentTier string, currentUsage int64) string {
	suggestedTier := SuggestTier(currentUsage, ratelimit.Tier(currentTier))
	return fmt.Sprintf("https://app.apx.dev/billing/upgrade?current=%s&suggested=%s&from=quota_exceeded",
		currentTier, suggestedTier)
}

// formatNumber formats a number with thousands separators
func formatNumber(n int64) string {
	if n < 0 {
		return "unlimited"
	}

	// Simple thousands separator
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Insert commas
	var result string
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// formatTierName formats a tier name for display
func formatTierName(tier string) string {
	switch tier {
	case "free":
		return "Free"
	case "pro":
		return "Pro"
	case "enterprise":
		return "Enterprise"
	default:
		return tier
	}
}

// formatCentsToUSD formats cents to USD string
func formatCentsToUSD(cents int64) string {
	dollars := float64(cents) / 100.0
	return fmt.Sprintf("$%.2f", dollars)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
