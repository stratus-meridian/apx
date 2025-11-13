package policy

import (
	"fmt"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	entry := &CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "abc123",
		WASM:    []byte("fake wasm"),
	}

	// Set entry
	cache.Set(entry)

	// Get entry
	retrieved, found := cache.Get("test-policy", "1.0.0")
	if !found {
		t.Fatal("entry should be found in cache")
	}

	if retrieved.Name != "test-policy" {
		t.Errorf("expected name 'test-policy', got '%s'", retrieved.Name)
	}

	if retrieved.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", retrieved.Version)
	}

	if retrieved.Hash != "abc123" {
		t.Errorf("expected hash 'abc123', got '%s'", retrieved.Hash)
	}
}

func TestCache_GetNonExistent(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	// Try to get non-existent entry
	_, found := cache.Get("non-existent", "1.0.0")
	if found {
		t.Error("non-existent entry should not be found")
	}
}

func TestCache_Evict(t *testing.T) {
	cache := NewCache(100 * time.Millisecond) // Short TTL for testing

	entry := &CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "abc123",
		WASM:    []byte("fake wasm"),
	}

	cache.Set(entry)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Evict expired entries
	evicted := cache.Evict()
	if evicted != 1 {
		t.Errorf("expected 1 eviction, got %d", evicted)
	}

	// Entry should be gone
	_, found := cache.Get("test-policy", "1.0.0")
	if found {
		t.Error("entry should not be found after eviction")
	}
}

func TestCache_EvictNoExpired(t *testing.T) {
	cache := NewCache(24 * time.Hour) // Long TTL

	entry := &CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "abc123",
		WASM:    []byte("fake wasm"),
	}

	cache.Set(entry)

	// Evict immediately - nothing should be evicted
	evicted := cache.Evict()
	if evicted != 0 {
		t.Errorf("expected 0 evictions, got %d", evicted)
	}

	// Entry should still be there
	_, found := cache.Get("test-policy", "1.0.0")
	if !found {
		t.Error("entry should still be found")
	}
}

func TestCache_Size(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	if cache.Size() != 0 {
		t.Errorf("expected size 0, got %d", cache.Size())
	}

	cache.Set(&CacheEntry{Name: "policy1", Version: "1.0.0", WASM: []byte("test")})

	if cache.Size() != 1 {
		t.Errorf("expected size 1, got %d", cache.Size())
	}

	cache.Set(&CacheEntry{Name: "policy2", Version: "1.0.0", WASM: []byte("test")})

	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}
}

func TestCache_Keys(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	cache.Set(&CacheEntry{Name: "policy1", Version: "1.0.0", WASM: []byte("test")})
	cache.Set(&CacheEntry{Name: "policy2", Version: "2.0.0", WASM: []byte("test")})

	keys := cache.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	// Check keys exist (order doesn't matter)
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	if !keyMap["policy1@1.0.0"] {
		t.Error("expected key 'policy1@1.0.0' not found")
	}
	if !keyMap["policy2@2.0.0"] {
		t.Error("expected key 'policy2@2.0.0' not found")
	}
}

func TestCache_NAndNMinus1Versions(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	// Simulate N and N-1 versions
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

	// Both should be accessible
	n, found := cache.Get("payment-policy", "2.0.0")
	if !found {
		t.Fatal("N version should be found")
	}
	if string(n.WASM) != "wasm-n" {
		t.Error("N version has wrong WASM")
	}

	nMinus1, found := cache.Get("payment-policy", "1.9.0")
	if !found {
		t.Fatal("N-1 version should be found")
	}
	if string(nMinus1.WASM) != "wasm-n-1" {
		t.Error("N-1 version has wrong WASM")
	}
}

func TestCache_UpdateEntry(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	// Set initial entry
	entry1 := &CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "hash1",
		WASM:    []byte("wasm1"),
	}
	cache.Set(entry1)

	// Update with new hash (same name/version)
	entry2 := &CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "hash2",
		WASM:    []byte("wasm2"),
	}
	cache.Set(entry2)

	// Should have the updated version
	retrieved, found := cache.Get("test-policy", "1.0.0")
	if !found {
		t.Fatal("entry should be found")
	}

	if retrieved.Hash != "hash2" {
		t.Errorf("expected updated hash 'hash2', got '%s'", retrieved.Hash)
	}

	if string(retrieved.WASM) != "wasm2" {
		t.Errorf("expected updated WASM 'wasm2', got '%s'", string(retrieved.WASM))
	}

	// Size should still be 1 (update, not addition)
	if cache.Size() != 1 {
		t.Errorf("expected size 1 after update, got %d", cache.Size())
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			entry := &CacheEntry{
				Name:    fmt.Sprintf("policy-%d", idx),
				Version: "1.0.0",
				Hash:    fmt.Sprintf("hash-%d", idx),
				WASM:    []byte(fmt.Sprintf("wasm-%d", idx)),
			}
			cache.Set(entry)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(idx int) {
			_, _ = cache.Get(fmt.Sprintf("policy-%d", idx), "1.0.0")
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify size
	if cache.Size() != 10 {
		t.Errorf("expected size 10 after concurrent writes, got %d", cache.Size())
	}
}

func TestCache_LastUsedAtUpdates(t *testing.T) {
	cache := NewCache(24 * time.Hour)

	entry := &CacheEntry{
		Name:    "test-policy",
		Version: "1.0.0",
		Hash:    "abc123",
		WASM:    []byte("fake wasm"),
	}

	cache.Set(entry)

	// Get initial entry
	retrieved1, _ := cache.Get("test-policy", "1.0.0")
	firstAccess := retrieved1.LastUsedAt

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Get again
	retrieved2, _ := cache.Get("test-policy", "1.0.0")
	secondAccess := retrieved2.LastUsedAt

	// LastUsedAt should be updated
	if !secondAccess.After(firstAccess) {
		t.Error("LastUsedAt should be updated on each Get")
	}
}
