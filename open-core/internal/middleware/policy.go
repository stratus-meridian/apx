package middleware

import (
	"net/http"

	"github.com/stratus-meridian/apx-router-open-core/internal/policy"
	"go.uber.org/zap"
)

// PolicyVersionTag adds policy version metadata to responses.
// This is useful for debugging and tracking which policy version processed a request.
func PolicyVersionTag(store policy.Store, logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get tenant from context
			tenant, ok := GetTenant(r.Context())
			if ok && tenant != nil && store != nil {
				version := store.GetPolicyVersion(tenant.ID)
				w.Header().Set("X-Policy-Version", version)
			}

			next.ServeHTTP(w, r)
		})
	}
}
