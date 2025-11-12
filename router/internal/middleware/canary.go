package middleware

import (
	"context"
	"crypto/rand"
	"math/big"
	"net/http"

	"github.com/apx/router/internal/policy"
	"go.uber.org/zap"
)

type contextKey string

const (
	// CanaryWeightKey is the context key for the canary weight
	CanaryWeightKey contextKey = "canary_weight"
	// PolicyVersionKey is the context key for the selected policy version
	PolicyVersionKey contextKey = "policy_version"
)

// CanarySelector adds canary traffic routing logic
// It assigns each request a random weight (0-100) and stores it in context
// This weight is used by the policy store to select the appropriate policy version
func CanarySelector(store *policy.Store, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate a cryptographically secure random number between 0-100
			canaryWeight := generateCanaryWeight()

			// Store the canary weight in context for downstream use
			ctx := context.WithValue(r.Context(), CanaryWeightKey, canaryWeight)

			// Note: Policy version selection happens later in the request chain
			// when we know which policy to apply (based on tenant, route, etc.)
			// The middleware just generates and stores the canary weight

			logger.Debug("canary weight assigned",
				zap.Int("weight", canaryWeight),
				zap.String("request_id", r.Header.Get("X-Request-ID")),
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// generateCanaryWeight generates a random number between 0 and 100
func generateCanaryWeight() int {
	// Use crypto/rand for secure random number generation
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		// Fall back to a safe default if crypto random fails
		return 50
	}
	return int(n.Int64())
}

// GetCanaryWeight extracts the canary weight from request context
func GetCanaryWeight(ctx context.Context) int {
	if weight, ok := ctx.Value(CanaryWeightKey).(int); ok {
		return weight
	}
	// Default to 50 if not set (middle of the range)
	return 50
}

// GetPolicyVersion extracts the selected policy version from request context
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
