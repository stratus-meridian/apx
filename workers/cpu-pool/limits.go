package cpupool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// TenantLimits manages per-tenant concurrency and resource limits
type TenantLimits struct {
	// Per-tenant concurrency semaphores
	semaphores map[string]*semaphore.Weighted

	// Per-tenant active request counters
	activeRequests map[string]int

	// Lock for thread-safe map access
	mu sync.RWMutex

	// Product configuration for tenant limits
	config *TenantLimitConfig
}

// TenantLimitConfig defines limits per tenant tier
type TenantLimitConfig struct {
	// Default limits by tenant tier
	TierLimits map[string]*TierLimit

	// Per-tenant overrides (for custom contracts)
	TenantOverrides map[string]*TierLimit

	// Global defaults
	DefaultConcurrency int
	DefaultTimeout     time.Duration
}

// TierLimit defines resource limits for a tier
type TierLimit struct {
	// Maximum concurrent requests per tenant
	MaxConcurrency int

	// Request timeout
	Timeout time.Duration

	// Maximum request size in bytes
	MaxRequestSize int64

	// Maximum response size in bytes
	MaxResponseSize int64

	// Priority (higher = processed first when queue is full)
	Priority int
}

// NewTenantLimits creates a new tenant limits manager
func NewTenantLimits(config *TenantLimitConfig) *TenantLimits {
	if config == nil {
		config = DefaultTenantLimitConfig()
	}

	return &TenantLimits{
		semaphores:     make(map[string]*semaphore.Weighted),
		activeRequests: make(map[string]int),
		config:         config,
	}
}

// DefaultTenantLimitConfig returns sensible defaults
func DefaultTenantLimitConfig() *TenantLimitConfig {
	return &TenantLimitConfig{
		TierLimits: map[string]*TierLimit{
			"free": {
				MaxConcurrency:  5,
				Timeout:         30 * time.Second,
				MaxRequestSize:  1 * 1024 * 1024,  // 1MB
				MaxResponseSize: 5 * 1024 * 1024,  // 5MB
				Priority:        1,
			},
			"pro": {
				MaxConcurrency:  50,
				Timeout:         60 * time.Second,
				MaxRequestSize:  10 * 1024 * 1024,  // 10MB
				MaxResponseSize: 50 * 1024 * 1024,  // 50MB
				Priority:        5,
			},
			"enterprise": {
				MaxConcurrency:  500,
				Timeout:         120 * time.Second,
				MaxRequestSize:  100 * 1024 * 1024,  // 100MB
				MaxResponseSize: 500 * 1024 * 1024,  // 500MB
				Priority:        10,
			},
		},
		TenantOverrides:    make(map[string]*TierLimit),
		DefaultConcurrency: 10,
		DefaultTimeout:     30 * time.Second,
	}
}

// AcquireSlot attempts to acquire a processing slot for a tenant
// Returns error if tenant has exceeded concurrency limit
func (tl *TenantLimits) AcquireSlot(ctx context.Context, tenantID string, tenantTier string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	// Get or create semaphore for tenant
	sem := tl.getSemaphore(tenantID, tenantTier)

	// CRITICAL: TryAcquire ensures we don't block other tenants
	// If this tenant is at their limit, fail fast
	if !sem.TryAcquire(1) {
		limit := tl.getLimit(tenantID, tenantTier)
		return &ConcurrencyLimitExceededError{
			TenantID:   tenantID,
			TenantTier: tenantTier,
			Limit:      limit.MaxConcurrency,
			Current:    tl.getActiveRequests(tenantID),
		}
	}

	// Track active request
	tl.incrementActiveRequests(tenantID)

	return nil
}

// ReleaseSlot releases a processing slot for a tenant
func (tl *TenantLimits) ReleaseSlot(tenantID string) {
	if tenantID == "" {
		return
	}

	tl.mu.RLock()
	sem, exists := tl.semaphores[tenantID]
	tl.mu.RUnlock()

	if exists {
		sem.Release(1)
		tl.decrementActiveRequests(tenantID)
	}
}

// getSemaphore gets or creates a semaphore for a tenant
func (tl *TenantLimits) getSemaphore(tenantID string, tenantTier string) *semaphore.Weighted {
	tl.mu.RLock()
	sem, exists := tl.semaphores[tenantID]
	tl.mu.RUnlock()

	if exists {
		return sem
	}

	// Create new semaphore with tenant's limit
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// Double-check after acquiring write lock
	if sem, exists := tl.semaphores[tenantID]; exists {
		return sem
	}

	limit := tl.getLimit(tenantID, tenantTier)
	sem = semaphore.NewWeighted(int64(limit.MaxConcurrency))
	tl.semaphores[tenantID] = sem

	return sem
}

// getLimit returns the limit configuration for a tenant
func (tl *TenantLimits) getLimit(tenantID string, tenantTier string) *TierLimit {
	// Check for tenant-specific override
	if override, exists := tl.config.TenantOverrides[tenantID]; exists {
		return override
	}

	// Check tier limits
	if limit, exists := tl.config.TierLimits[tenantTier]; exists {
		return limit
	}

	// Return default
	return &TierLimit{
		MaxConcurrency:  tl.config.DefaultConcurrency,
		Timeout:         tl.config.DefaultTimeout,
		MaxRequestSize:  10 * 1024 * 1024,
		MaxResponseSize: 50 * 1024 * 1024,
		Priority:        1,
	}
}

// incrementActiveRequests increments the active request counter for a tenant
func (tl *TenantLimits) incrementActiveRequests(tenantID string) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.activeRequests[tenantID]++
}

// decrementActiveRequests decrements the active request counter for a tenant
func (tl *TenantLimits) decrementActiveRequests(tenantID string) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	if tl.activeRequests[tenantID] > 0 {
		tl.activeRequests[tenantID]--
	}
}

// getActiveRequests returns the number of active requests for a tenant
func (tl *TenantLimits) getActiveRequests(tenantID string) int {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	return tl.activeRequests[tenantID]
}

// GetStats returns current stats for a tenant
func (tl *TenantLimits) GetStats(tenantID string, tenantTier string) *TenantStats {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	limit := tl.getLimit(tenantID, tenantTier)
	active := tl.activeRequests[tenantID]

	return &TenantStats{
		TenantID:       tenantID,
		TenantTier:     tenantTier,
		MaxConcurrency: limit.MaxConcurrency,
		ActiveRequests: active,
		Available:      limit.MaxConcurrency - active,
	}
}

// GetAllStats returns stats for all active tenants
func (tl *TenantLimits) GetAllStats() []*TenantStats {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	stats := make([]*TenantStats, 0, len(tl.activeRequests))
	for tenantID, active := range tl.activeRequests {
		if active > 0 {
			// We don't know the tier here, so use empty string
			limit := tl.getLimit(tenantID, "")
			stats = append(stats, &TenantStats{
				TenantID:       tenantID,
				TenantTier:     "unknown",
				MaxConcurrency: limit.MaxConcurrency,
				ActiveRequests: active,
				Available:      limit.MaxConcurrency - active,
			})
		}
	}

	return stats
}

// TenantStats represents current tenant resource usage
type TenantStats struct {
	TenantID       string
	TenantTier     string
	MaxConcurrency int
	ActiveRequests int
	Available      int
}

// ConcurrencyLimitExceededError represents a concurrency limit violation
type ConcurrencyLimitExceededError struct {
	TenantID   string
	TenantTier string
	Limit      int
	Current    int
}

func (e *ConcurrencyLimitExceededError) Error() string {
	return fmt.Sprintf("concurrency limit exceeded for tenant %s (%s tier): %d/%d",
		e.TenantID, e.TenantTier, e.Current, e.Limit)
}

// IsConcurrencyLimitError checks if an error is a concurrency limit error
func IsConcurrencyLimitError(err error) bool {
	_, ok := err.(*ConcurrencyLimitExceededError)
	return ok
}
