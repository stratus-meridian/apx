package policy

import (
	"context"
	"fmt"
	"time"
)

// PolicyLoader defines the interface for loading policies
type PolicyLoader interface {
	Load(ctx context.Context, name, version, hash string) (*CacheEntry, error)
	LoadLatest(ctx context.Context, name string) (*CacheEntry, error)
	Close() error
}

// Executor executes policies using OPA engine
type Executor struct {
	cache  *Cache
	loader PolicyLoader
}

// NewExecutor creates a new policy executor
func NewExecutor(cache *Cache, loader PolicyLoader) *Executor {
	return &Executor{
		cache:  cache,
		loader: loader,
	}
}

// Execute runs a policy evaluation
func (e *Executor) Execute(ctx context.Context, policyName, version string, input interface{}) (bool, error) {
	// Get from cache or load
	entry, err := e.getOrLoad(ctx, policyName, version)
	if err != nil {
		return false, fmt.Errorf("failed to get policy: %w", err)
	}

	// For now, return success (full OPA integration in next phase)
	// TODO: Integrate with OPA engine from M2-T1-001
	// This will involve:
	// 1. Creating an OPA engine instance from the WASM bytes
	// 2. Evaluating the policy with the provided input
	// 3. Returning the evaluation result
	_ = entry // Use entry to avoid unused variable error

	// Placeholder: return true for demonstration
	return true, nil
}

// ExecuteWithHash runs a policy evaluation with explicit hash for cache lookup
func (e *Executor) ExecuteWithHash(ctx context.Context, policyName, version, hash string, input interface{}) (bool, error) {
	// Try cache first
	entry, found := e.cache.Get(policyName, version)
	if found && entry.Hash == hash {
		// Cache hit with matching hash
		_ = entry // Will be used for OPA evaluation
		return true, nil
	}

	// Cache miss or hash mismatch - load from GCS
	if e.loader == nil {
		return false, fmt.Errorf("no loader configured for cache miss")
	}

	loadedEntry, err := e.loader.Load(ctx, policyName, version, hash)
	if err != nil {
		return false, fmt.Errorf("failed to load policy: %w", err)
	}

	// Add to cache
	e.cache.Set(loadedEntry)

	// Execute (placeholder)
	return true, nil
}

// getOrLoad retrieves policy from cache or loads from GCS
func (e *Executor) getOrLoad(ctx context.Context, name, version string) (*CacheEntry, error) {
	// Check cache first
	if entry, found := e.cache.Get(name, version); found {
		return entry, nil
	}

	// Cache miss - need to load from GCS
	// For MVP, we'll need the hash to construct the path
	// In production, this would query Firestore first to get the hash
	return nil, fmt.Errorf("cache miss: policy %s@%s not found (Firestore integration needed)", name, version)
}

// Preload loads N and N-1 versions into cache
func (e *Executor) Preload(ctx context.Context, name string, nVersion, nMinus1Version, nHash, nMinus1Hash string) error {
	// This would be called on worker startup to preload common policies
	if e.loader == nil {
		return fmt.Errorf("no loader configured for preload")
	}

	// Load N version
	nEntry, err := e.loader.Load(ctx, name, nVersion, nHash)
	if err != nil {
		return fmt.Errorf("failed to preload N version: %w", err)
	}
	e.cache.Set(nEntry)

	// Load N-1 version
	nMinus1Entry, err := e.loader.Load(ctx, name, nMinus1Version, nMinus1Hash)
	if err != nil {
		return fmt.Errorf("failed to preload N-1 version: %w", err)
	}
	e.cache.Set(nMinus1Entry)

	return nil
}

// GetCacheStats returns cache statistics
func (e *Executor) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"size":      e.cache.Size(),
		"keys":      e.cache.Keys(),
		"timestamp": time.Now(),
	}
}

// EvictExpired removes expired cache entries
func (e *Executor) EvictExpired() int {
	return e.cache.Evict()
}

// GetVersionStats returns statistics about version usage
func (e *Executor) GetVersionStats() map[string]interface{} {
	keys := e.cache.Keys()

	versionCounts := make(map[string]int)
	for _, key := range keys {
		// Keys are in format "name@version"
		versionCounts[key]++
	}

	return map[string]interface{}{
		"total_cached": len(keys),
		"versions":     versionCounts,
		"timestamp":    time.Now(),
	}
}
