package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stratus-meridian/apx-private/control/tenant"
	"github.com/stratus-meridian/apx/router/internal/middleware"
	"go.uber.org/zap"
)

const (
	defaultCacheTTL = 5 * time.Minute

	cacheKeyPrefix = "tenant-resolver"
)

// FirestoreTenantResolver implements middleware.TenantResolver by looking up tenant
// metadata from the control-plane Firestore repository with an optional Redis cache.
type FirestoreTenantResolver struct {
	repo          tenant.Repository
	cache         *redis.Client
	logger        *zap.Logger
	cacheTTL      time.Duration
	defaultTenant *tenant.Tenant
}

// NewFirestoreTenantResolver builds a resolver backed by the shared tenant repository.
// A Redis client is optional; when provided it is used to cache API key lookups for
// defaultCacheTTL to minimize Firestore reads in Cloud Run/GKE.
func NewFirestoreTenantResolver(repo tenant.Repository, cache *redis.Client, logger *zap.Logger) *FirestoreTenantResolver {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &FirestoreTenantResolver{
		repo:          repo,
		cache:         cache,
		logger:        logger,
		cacheTTL:      defaultCacheTTL,
		defaultTenant: buildDefaultTenant(),
	}
}

// ResolveTenant fetches the tenant hierarchy for the provided API key.
func (r *FirestoreTenantResolver) ResolveTenant(ctx context.Context, apiKey string) (*tenant.Tenant, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api key is required")
	}

	start := time.Now()
	var tenantCtx *tenant.Tenant
	var fromCache bool

	defer func() {
		if !middleware.IsDebugEnabled() {
			return
		}

		tenantID := ""
		if tenantCtx != nil {
			tenantID = tenantCtx.ResourceID
		}

		r.logger.Info("tenant.resolve",
			zap.String("tenant_id", tenantID),
			zap.Bool("cache_hit", fromCache),
			zap.Duration("duration", time.Since(start)),
		)
	}()

	if err := tenant.ValidateAPIKey(apiKey); err != nil {
		return nil, err
	}

	if cached := r.getFromCache(ctx, apiKey); cached != nil {
		fromCache = true
		tenantCtx = cached
		return cached, nil
	}

	var err error
	tenantCtx, err = r.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	if err := validateTenantState(tenantCtx); err != nil {
		return nil, err
	}

	r.setCache(ctx, apiKey, tenantCtx)
	return tenantCtx, nil
}

// GetDefaultTenant returns the restrictive default tenant.
func (r *FirestoreTenantResolver) GetDefaultTenant(ctx context.Context) *tenant.Tenant {
	return r.defaultTenant
}

// Close releases resolver resources. Currently this is a no-op but it keeps the
// surface area symmetrical with other infrastructure components.
func (r *FirestoreTenantResolver) Close() error {
	return nil
}

func (r *FirestoreTenantResolver) cacheKey(apiKey string) string {
	return fmt.Sprintf("%s:%s", cacheKeyPrefix, apiKey)
}

func (r *FirestoreTenantResolver) getFromCache(ctx context.Context, apiKey string) *tenant.Tenant {
	if r.cache == nil {
		return nil
	}

	result, err := r.cache.Get(ctx, r.cacheKey(apiKey)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			r.logger.Warn("tenant cache get failed", zap.Error(err))
		}
		return nil
	}

	if result == "" {
		return nil
	}

	var cached tenant.Tenant
	if err := json.Unmarshal([]byte(result), &cached); err != nil {
		r.logger.Warn("failed to unmarshal cached tenant", zap.Error(err))
		_ = r.cache.Del(ctx, r.cacheKey(apiKey)).Err()
		return nil
	}

	return &cached
}

func (r *FirestoreTenantResolver) setCache(ctx context.Context, apiKey string, value *tenant.Tenant) {
	if r.cache == nil || value == nil {
		return
	}

	data, err := json.Marshal(value)
	if err != nil {
		r.logger.Warn("failed to marshal tenant for cache", zap.Error(err))
		return
	}

	if err := r.cache.Set(ctx, r.cacheKey(apiKey), data, r.cacheTTL).Err(); err != nil {
		r.logger.Warn("tenant cache set failed", zap.Error(err))
	}
}

func validateTenantState(t *tenant.Tenant) error {
	if t == nil {
		return tenant.ErrNotFound
	}

	if t.Organization.Status != tenant.StatusActive {
		return fmt.Errorf("organization %s is %s", t.Organization.ID, t.Organization.Status)
	}

	if t.Environment.Status != tenant.StatusActive {
		return fmt.Errorf("environment %s is %s", t.Environment.ResourceID, t.Environment.Status)
	}

	return nil
}

func buildDefaultTenant() *tenant.Tenant {
	return &tenant.Tenant{
		Organization: tenant.Organization{
			ID:     "apx_default_org",
			Name:   "Default",
			Tier:   tenant.TierFree,
			Status: tenant.StatusActive,
			Quotas: tenant.GetDefaultQuotas(tenant.TierFree),
		},
		Product: tenant.Product{
			ID:     "default",
			OrgID:  "apx_default_org",
			Name:   "Default",
			Status: tenant.StatusActive,
		},
		Environment: tenant.Environment{
			ID:         "default",
			OrgID:      "apx_default_org",
			ProductID:  "default",
			ResourceID: tenant.BuildResourceID("apx", "default", "default"),
			Name:       "Default",
			Type:       tenant.EnvDevelopment,
			Status:     tenant.StatusActive,
			APIKey:     "apx_test_default",
		},
		ResourceID: tenant.BuildResourceID("apx", "default", "default"),
	}
}
