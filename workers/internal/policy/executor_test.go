package policy

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestExecutor_GetCacheStats(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil) // nil loader for this test

	// Add some entries
	cache.Set(&CacheEntry{Name: "policy1", Version: "1.0.0", WASM: []byte("test")})
	cache.Set(&CacheEntry{Name: "policy2", Version: "1.0.0", WASM: []byte("test")})

	stats := executor.GetCacheStats()

	size, ok := stats["size"].(int)
	if !ok || size != 2 {
		t.Errorf("expected size 2, got %v", stats["size"])
	}

	keys, ok := stats["keys"].([]string)
	if !ok || len(keys) != 2 {
		t.Errorf("expected 2 keys, got %v", stats["keys"])
	}

	_, ok = stats["timestamp"].(time.Time)
	if !ok {
		t.Error("expected timestamp in stats")
	}
}

func TestExecutor_Execute_CacheHit(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil)

	// Pre-populate cache
	cache.Set(&CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "abc123",
		WASM:    []byte("fake wasm"),
	})

	// Execute should find it in cache
	ctx := context.Background()
	result, err := executor.Execute(ctx, "test-policy", "1.0.0", map[string]interface{}{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Placeholder returns true
	if !result {
		t.Error("expected result true")
	}
}

func TestExecutor_Execute_CacheMiss(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil)

	// Don't populate cache - should get cache miss error
	ctx := context.Background()
	_, err := executor.Execute(ctx, "missing-policy", "1.0.0", map[string]interface{}{})

	if err == nil {
		t.Error("expected error for cache miss, got nil")
	}

	expectedMsg := "cache miss: policy missing-policy@1.0.0 not found (Firestore integration needed)"
	if err.Error() != "failed to get policy: "+expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExecutor_ExecuteWithHash_CacheHit(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil)

	// Pre-populate cache
	cache.Set(&CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "abc123",
		WASM:    []byte("fake wasm"),
	})

	ctx := context.Background()
	result, err := executor.ExecuteWithHash(ctx, "test-policy", "1.0.0", "abc123", map[string]interface{}{})

	if err != nil {
		t.Fatalf("ExecuteWithHash failed: %v", err)
	}

	if !result {
		t.Error("expected result true")
	}
}

func TestExecutor_ExecuteWithHash_CacheMiss_NoLoader(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil) // no loader

	ctx := context.Background()
	_, err := executor.ExecuteWithHash(ctx, "missing-policy", "1.0.0", "hash123", map[string]interface{}{})

	if err == nil {
		t.Error("expected error for cache miss without loader")
	}

	expectedMsg := "no loader configured for cache miss"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%v'", expectedMsg, err)
	}
}

func TestExecutor_EvictExpired(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)
	executor := NewExecutor(cache, nil)

	// Add entries
	cache.Set(&CacheEntry{Name: "policy1", Version: "1.0.0", WASM: []byte("test")})
	cache.Set(&CacheEntry{Name: "policy2", Version: "1.0.0", WASM: []byte("test")})

	// Wait for expiry
	time.Sleep(75 * time.Millisecond)

	// Evict
	evicted := executor.EvictExpired()
	if evicted != 2 {
		t.Errorf("expected 2 evictions, got %d", evicted)
	}

	stats := executor.GetCacheStats()
	size := stats["size"].(int)
	if size != 0 {
		t.Errorf("expected size 0 after eviction, got %d", size)
	}
}

func TestExecutor_GetVersionStats(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil)

	// Add multiple versions
	cache.Set(&CacheEntry{Name: "policy1", Version: "1.0.0", WASM: []byte("test")})
	cache.Set(&CacheEntry{Name: "policy1", Version: "2.0.0", WASM: []byte("test")})
	cache.Set(&CacheEntry{Name: "policy2", Version: "1.0.0", WASM: []byte("test")})

	stats := executor.GetVersionStats()

	totalCached, ok := stats["total_cached"].(int)
	if !ok || totalCached != 3 {
		t.Errorf("expected total_cached 3, got %v", stats["total_cached"])
	}

	versions, ok := stats["versions"].(map[string]int)
	if !ok {
		t.Fatalf("expected versions map, got %v", stats["versions"])
	}

	if versions["policy1@1.0.0"] != 1 {
		t.Errorf("expected policy1@1.0.0 count 1, got %d", versions["policy1@1.0.0"])
	}

	if versions["policy1@2.0.0"] != 1 {
		t.Errorf("expected policy1@2.0.0 count 1, got %d", versions["policy1@2.0.0"])
	}

	if versions["policy2@1.0.0"] != 1 {
		t.Errorf("expected policy2@1.0.0 count 1, got %d", versions["policy2@1.0.0"])
	}
}

func TestExecutor_Preload_NoLoader(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil) // no loader

	ctx := context.Background()
	err := executor.Preload(ctx, "policy1", "2.0.0", "1.9.0", "hash-n", "hash-n-1")

	if err == nil {
		t.Error("expected error for preload without loader")
	}

	expectedMsg := "no loader configured for preload"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%v'", expectedMsg, err)
	}
}

func TestExecutor_NAndNMinus1Scenario(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil)

	// Simulate worker startup: preload N and N-1
	nEntry := &CacheEntry{
		Name:    "payment-policy",
		Version: "2.0.0",
		Hash:    "hash-n",
		WASM:    []byte("wasm-n"),
	}

	nMinus1Entry := &CacheEntry{
		Name:    "payment-policy",
		Version: "1.9.0",
		Hash:    "hash-n-1",
		WASM:    []byte("wasm-n-1"),
	}

	cache.Set(nEntry)
	cache.Set(nMinus1Entry)

	// Verify both versions are cached
	stats := executor.GetCacheStats()
	if stats["size"].(int) != 2 {
		t.Errorf("expected 2 cached versions, got %d", stats["size"])
	}

	// Execute with N version
	ctx := context.Background()
	resultN, err := executor.Execute(ctx, "payment-policy", "2.0.0", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute N version failed: %v", err)
	}
	if !resultN {
		t.Error("expected N version execution to succeed")
	}

	// Execute with N-1 version
	resultNMinus1, err := executor.Execute(ctx, "payment-policy", "1.9.0", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute N-1 version failed: %v", err)
	}
	if !resultNMinus1 {
		t.Error("expected N-1 version execution to succeed")
	}

	// Check version stats
	versionStats := executor.GetVersionStats()
	versions := versionStats["versions"].(map[string]int)

	if versions["payment-policy@2.0.0"] != 1 {
		t.Error("N version should be in stats")
	}

	if versions["payment-policy@1.9.0"] != 1 {
		t.Error("N-1 version should be in stats")
	}
}

func TestExecutor_ExecuteWithHash_HashMismatch(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	executor := NewExecutor(cache, nil)

	// Pre-populate cache with one hash
	cache.Set(&CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "old-hash",
		WASM:    []byte("old wasm"),
	})

	// Try to execute with different hash - should trigger cache miss
	ctx := context.Background()
	_, err := executor.ExecuteWithHash(ctx, "test-policy", "1.0.0", "new-hash", map[string]interface{}{})

	if err == nil {
		t.Error("expected error for hash mismatch without loader")
	}

	// Should get "no loader" error since we need to reload
	expectedMsg := "no loader configured for cache miss"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%v'", expectedMsg, err)
	}
}

// MockLoader is a test loader that simulates GCS loading without actual network calls
type MockLoader struct {
	entries map[string]*CacheEntry
	loadErr error
}

func (m *MockLoader) Load(ctx context.Context, name, version, hash string) (*CacheEntry, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}

	key := name + "@" + version + "@" + hash
	if entry, ok := m.entries[key]; ok {
		return entry, nil
	}

	return &CacheEntry{
		Name:    name,
		Version: version,
		Hash:    hash,
		WASM:    []byte("mock wasm"),
	}, nil
}

func (m *MockLoader) LoadLatest(ctx context.Context, name string) (*CacheEntry, error) {
	return nil, fmt.Errorf("LoadLatest not implemented in mock")
}

func (m *MockLoader) Close() error {
	return nil
}

func TestExecutor_ExecuteWithHash_LoadFromGCS(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	mockLoader := &MockLoader{
		entries: make(map[string]*CacheEntry),
	}
	executor := NewExecutor(cache, mockLoader)

	// Cache miss - should load from mock loader
	ctx := context.Background()
	result, err := executor.ExecuteWithHash(ctx, "new-policy", "1.0.0", "hash123", map[string]interface{}{})

	if err != nil {
		t.Fatalf("ExecuteWithHash with loader failed: %v", err)
	}

	if !result {
		t.Error("expected result true")
	}

	// Verify it was cached
	entry, found := cache.Get("new-policy", "1.0.0")
	if !found {
		t.Error("entry should be cached after loading")
	}

	if entry.Hash != "hash123" {
		t.Errorf("expected hash 'hash123', got '%s'", entry.Hash)
	}
}

func TestExecutor_ExecuteWithHash_LoadError(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	mockLoader := &MockLoader{
		entries: make(map[string]*CacheEntry),
		loadErr: fmt.Errorf("GCS connection failed"),
	}
	executor := NewExecutor(cache, mockLoader)

	ctx := context.Background()
	_, err := executor.ExecuteWithHash(ctx, "failing-policy", "1.0.0", "hash123", map[string]interface{}{})

	if err == nil {
		t.Error("expected error from loader failure")
	}

	if !contains(err.Error(), "failed to load policy") {
		t.Errorf("expected 'failed to load policy' in error, got: %v", err)
	}
}

func TestExecutor_Preload_Success(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	mockLoader := &MockLoader{
		entries: make(map[string]*CacheEntry),
	}
	executor := NewExecutor(cache, mockLoader)

	ctx := context.Background()
	err := executor.Preload(ctx, "payment-policy", "2.0.0", "1.9.0", "hash-n", "hash-n-1")

	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}

	// Verify N version is cached
	nEntry, found := cache.Get("payment-policy", "2.0.0")
	if !found {
		t.Error("N version should be cached after preload")
	}
	if nEntry.Hash != "hash-n" {
		t.Errorf("expected N hash 'hash-n', got '%s'", nEntry.Hash)
	}

	// Verify N-1 version is cached
	nMinus1Entry, found := cache.Get("payment-policy", "1.9.0")
	if !found {
		t.Error("N-1 version should be cached after preload")
	}
	if nMinus1Entry.Hash != "hash-n-1" {
		t.Errorf("expected N-1 hash 'hash-n-1', got '%s'", nMinus1Entry.Hash)
	}

	// Verify cache size
	if cache.Size() != 2 {
		t.Errorf("expected cache size 2, got %d", cache.Size())
	}
}

func TestExecutor_Preload_NVersionError(t *testing.T) {
	cache := NewCache(24 * time.Hour)
	mockLoader := &MockLoader{
		entries: make(map[string]*CacheEntry),
		loadErr: fmt.Errorf("failed to load N version"),
	}
	executor := NewExecutor(cache, mockLoader)

	ctx := context.Background()
	err := executor.Preload(ctx, "payment-policy", "2.0.0", "1.9.0", "hash-n", "hash-n-1")

	if err == nil {
		t.Error("expected error from N version load failure")
	}

	if !contains(err.Error(), "failed to preload N version") {
		t.Errorf("expected 'failed to preload N version' in error, got: %v", err)
	}
}

func TestExecutor_Preload_NMinus1Error(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	// Mock loader that succeeds first, then fails
	callCount := 0
	mockLoader := &MockLoader{
		entries: make(map[string]*CacheEntry),
	}

	// Create custom executor to intercept Load calls
	executor := NewExecutor(cache, mockLoader)

	// We need to test the N-1 error path, so let's create a loader that fails on second call
	specialMockLoader := &specialMockLoader{
		callCount: &callCount,
	}
	executor.loader = specialMockLoader

	ctx := context.Background()
	err := executor.Preload(ctx, "payment-policy", "2.0.0", "1.9.0", "hash-n", "hash-n-1")

	if err == nil {
		t.Error("expected error from N-1 version load failure")
	}

	if !contains(err.Error(), "failed to preload N-1 version") {
		t.Errorf("expected 'failed to preload N-1 version' in error, got: %v", err)
	}
}

// specialMockLoader for testing N-1 error path
type specialMockLoader struct {
	callCount *int
}

func (s *specialMockLoader) Load(ctx context.Context, name, version, hash string) (*CacheEntry, error) {
	*s.callCount++
	if *s.callCount == 1 {
		// First call (N version) succeeds
		return &CacheEntry{
			Name:    name,
			Version: version,
			Hash:    hash,
			WASM:    []byte("mock wasm"),
		}, nil
	}
	// Second call (N-1 version) fails
	return nil, fmt.Errorf("failed to load N-1")
}

func (s *specialMockLoader) LoadLatest(ctx context.Context, name string) (*CacheEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *specialMockLoader) Close() error {
	return nil
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
