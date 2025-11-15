package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/stratus-meridian/apx-private/control/pkg/ratelimit"
	"github.com/stratus-meridian/apx-private/control/tenant"
	"github.com/stratus-meridian/apx/router/pkg/responses"
	"go.uber.org/zap"
)

// QuotaMiddleware enforces monthly quotas per tenant and surfaces HTTP 402 responses.
type QuotaMiddleware struct {
	enforcer *ratelimit.QuotaEnforcer
	logger   *zap.Logger
}

// NewQuotaMiddleware constructs the quota enforcement middleware.
func NewQuotaMiddleware(enforcer *ratelimit.QuotaEnforcer, logger *zap.Logger) *QuotaMiddleware {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &QuotaMiddleware{
		enforcer: enforcer,
		logger:   logger,
	}
}

// Handler returns the middleware handler.
func (m *QuotaMiddleware) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.enforcer == nil {
				next.ServeHTTP(w, r)
				return
			}

			tenantCtx, ok := GetTenant(r.Context())
			if !ok || tenantCtx == nil {
				next.ServeHTTP(w, r)
				return
			}

			tier := ratelimit.Tier(tenantCtx.Organization.Tier)
			allowed, status, err := m.enforcer.ShouldAllowRequest(r.Context(), tenantCtx.ResourceID, tier)
			if err != nil {
				m.logger.Error("quota check failed",
					zap.Error(err),
					zap.String("tenant_id", tenantCtx.ResourceID))
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				m.sendPaymentRequired(w, tenantCtx, status)
				return
			}

			next.ServeHTTP(w, r)

			if updatedStatus, err := m.enforcer.IncrementQuota(r.Context(), tenantCtx.ResourceID, tier, 1); err != nil {
				m.logger.Warn("failed to increment quota",
					zap.Error(err),
					zap.String("tenant_id", tenantCtx.ResourceID))
			} else {
				status = updatedStatus
			}

			m.writeQuotaHeaders(w, status)
		})
	}
}

// QuotaEnforcement provides a functional helper mirroring other middleware constructors.
func QuotaEnforcement(enforcer *ratelimit.QuotaEnforcer, logger *zap.Logger) Middleware {
	return NewQuotaMiddleware(enforcer, logger).Handler()
}

func (m *QuotaMiddleware) sendPaymentRequired(w http.ResponseWriter, tenantCtx *tenant.Tenant, status *ratelimit.QuotaStatus) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)

	response := responses.NewPaymentRequired(tenantCtx, status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.Error("failed to write payment required response", zap.Error(err))
	}
}

func (m *QuotaMiddleware) writeQuotaHeaders(w http.ResponseWriter, status *ratelimit.QuotaStatus) {
	if status == nil {
		return
	}

	if status.Limit >= 0 {
		w.Header().Set("X-Quota-Limit", strconv.FormatInt(status.Limit, 10))
		w.Header().Set("X-Quota-Remaining", strconv.FormatInt(status.Remaining, 10))
	} else {
		w.Header().Set("X-Quota-Limit", "unlimited")
		w.Header().Set("X-Quota-Remaining", "unlimited")
	}

	w.Header().Set("X-Quota-Used", strconv.FormatInt(status.Used, 10))
	if !status.ResetAt.IsZero() {
		w.Header().Set("X-Quota-Reset", strconv.FormatInt(status.ResetAt.Unix(), 10))
	}
}
