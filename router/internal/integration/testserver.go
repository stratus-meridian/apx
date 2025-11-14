package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/stratus-meridian/apx/router/internal/middleware"
	"go.uber.org/zap"
)

// CacheEntry represents a cached policy version
type CacheEntry struct {
	Name       string
	Version    string
	Hash       string
	WASM       []byte
	LoadedAt   time.Time
	LastUsedAt time.Time
}

// Cache manages policy versions with N/N-1 support
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry // key: {name}@{version}
	ttl     time.Duration          // TTL for N-1 versions
}

// NewCache creates a new policy cache
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a policy from cache
func (c *Cache) Get(name, version string) (*CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s@%s", name, version)
	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Update last used time (need write lock since we're modifying entry)
	entry.LastUsedAt = time.Now()
	return entry, true
}

// Set adds or updates a policy in cache
func (c *Cache) Set(entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s@%s", entry.Name, entry.Version)
	entry.LoadedAt = time.Now()
	entry.LastUsedAt = time.Now()
	c.entries[key] = entry
}

// Evict removes expired entries (N-1 versions older than TTL)
func (c *Cache) Evict() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	evicted := 0

	for key, entry := range c.entries {
		if now.Sub(entry.LoadedAt) > c.ttl {
			delete(c.entries, key)
			evicted++
		}
	}

	return evicted
}

// Size returns the number of cached entries
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Keys returns all cache keys
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for k := range c.entries {
		keys = append(keys, k)
	}
	return keys
}

// PolicyLoader defines the interface for loading policies
type PolicyLoader interface {
	Load(ctx context.Context, name, version, hash string) (*CacheEntry, error)
	LoadLatest(ctx context.Context, name string) (*CacheEntry, error)
	Close() error
}

// Executor executes policies
type Executor struct {
	cache  *Cache
	loader PolicyLoader
}

// MockFirestoreClient simulates Firestore policy metadata queries
type MockFirestoreClient struct {
	mu       sync.RWMutex
	versions map[string]*PolicyMetadata // key: policyName@version
}

// PolicyMetadata represents policy metadata stored in Firestore
type PolicyMetadata struct {
	Name        string
	Version     string
	Hash        string
	CreatedAt   time.Time
	Description string
}

// NewMockFirestoreClient creates a new mock Firestore client
func NewMockFirestoreClient() *MockFirestoreClient {
	return &MockFirestoreClient{
		versions: make(map[string]*PolicyMetadata),
	}
}

// AddVersion adds a policy version to mock Firestore
func (m *MockFirestoreClient) AddVersion(metadata *PolicyMetadata) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s@%s", metadata.Name, metadata.Version)
	m.versions[key] = metadata
}

// GetVersion retrieves policy metadata by name and version
func (m *MockFirestoreClient) GetVersion(name, version string) (*PolicyMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s@%s", name, version)
	metadata, found := m.versions[key]
	if !found {
		return nil, fmt.Errorf("policy version not found: %s", key)
	}
	return metadata, nil
}

// GetLatestVersion retrieves the latest version of a policy
func (m *MockFirestoreClient) GetLatestVersion(name string) (*PolicyMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest *PolicyMetadata
	for _, metadata := range m.versions {
		if metadata.Name == name {
			if latest == nil || metadata.CreatedAt.After(latest.CreatedAt) {
				latest = metadata
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no versions found for policy: %s", name)
	}
	return latest, nil
}

// MockGCSClient simulates GCS WASM bundle storage
type MockGCSClient struct {
	mu      sync.RWMutex
	bundles map[string][]byte // key: policies/{name}/{version}/{hash}.wasm
}

// NewMockGCSClient creates a new mock GCS client
func NewMockGCSClient() *MockGCSClient {
	return &MockGCSClient{
		bundles: make(map[string][]byte),
	}
}

// AddBundle adds a WASM bundle to mock GCS
func (m *MockGCSClient) AddBundle(name, version, hash string, wasmBytes []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("policies/%s/%s/%s.wasm", name, version, hash)
	m.bundles[key] = wasmBytes
}

// GetBundle retrieves a WASM bundle from mock GCS
func (m *MockGCSClient) GetBundle(name, version, hash string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("policies/%s/%s/%s.wasm", name, version, hash)
	bundle, found := m.bundles[key]
	if !found {
		return nil, fmt.Errorf("bundle not found: %s", key)
	}
	return bundle, nil
}

// MockPolicyLoader implements PolicyLoader interface with mocks
type MockPolicyLoader struct {
	firestore *MockFirestoreClient
	gcs       *MockGCSClient
}

// NewMockPolicyLoader creates a new mock policy loader
func NewMockPolicyLoader(firestore *MockFirestoreClient, gcs *MockGCSClient) *MockPolicyLoader {
	return &MockPolicyLoader{
		firestore: firestore,
		gcs:       gcs,
	}
}

// Load implements PolicyLoader.Load
func (m *MockPolicyLoader) Load(ctx context.Context, name, version, hash string) (*CacheEntry, error) {
	// Get WASM bytes from mock GCS
	wasmBytes, err := m.gcs.GetBundle(name, version, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to load WASM bundle: %w", err)
	}

	return &CacheEntry{
		Name:    name,
		Version: version,
		Hash:    hash,
		WASM:    wasmBytes,
	}, nil
}

// LoadLatest implements PolicyLoader.LoadLatest
func (m *MockPolicyLoader) LoadLatest(ctx context.Context, name string) (*CacheEntry, error) {
	// Get latest metadata from mock Firestore
	metadata, err := m.firestore.GetLatestVersion(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	// Load the WASM bundle
	return m.Load(ctx, name, metadata.Version, metadata.Hash)
}

// Close implements PolicyLoader.Close
func (m *MockPolicyLoader) Close() error {
	return nil
}

// TestServer represents the full integration test server
type TestServer struct {
	Router        *httptest.Server
	Cache         *Cache
	Executor      *Executor
	Firestore     *MockFirestoreClient
	GCS           *MockGCSClient
	Loader        *MockPolicyLoader
	Logger        *zap.Logger
	VersionUsed   string // Captured version from last request
	FetchedFromFS bool   // Flag to track if Firestore was queried
	mu            sync.Mutex
}

// NewTestServer creates a new test server with full integration stack
func NewTestServer() *TestServer {
	// Create logger (silent for tests)
	logger := zap.NewNop()

	// Create mock storage
	firestore := NewMockFirestoreClient()
	gcs := NewMockGCSClient()
	loader := NewMockPolicyLoader(firestore, gcs)

	// Create policy cache with 24h TTL
	cache := NewCache(24 * time.Hour)

	// Create policy executor
	executor := &Executor{
		cache:  cache,
		loader: loader,
	}

	// Create test server
	ts := &TestServer{
		Cache:     cache,
		Executor:  executor,
		Firestore: firestore,
		GCS:       gcs,
		Loader:    loader,
		Logger:    logger,
	}

	// Create HTTP handler with middleware stack
	handler := ts.createHandler()

	// Create test HTTP server
	ts.Router = httptest.NewServer(handler)

	return ts
}

// createHandler creates the HTTP handler with middleware
func (ts *TestServer) createHandler() http.Handler {
	// Create policy version middleware
	policyVersion := middleware.NewPolicyVersion()

	// Create handler that simulates worker policy evaluation
	workerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract policy version from context (set by middleware)
		version := middleware.GetVersionFromContext(ctx)

		// Capture version for test verification
		ts.mu.Lock()
		ts.VersionUsed = version
		ts.mu.Unlock()

		// Resolve "latest" to actual version
		if version == "latest" {
			// Query Firestore for latest version
			ts.mu.Lock()
			ts.FetchedFromFS = true
			ts.mu.Unlock()

			metadata, err := ts.Firestore.GetLatestVersion("default")
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get latest version: %v", err), http.StatusInternalServerError)
				return
			}
			version = metadata.Version
		}

		// Try to get policy from cache
		entry, found := ts.Cache.Get("default", version)
		if !found {
			// Cache miss - fetch from Firestore/GCS
			ts.mu.Lock()
			ts.FetchedFromFS = true
			ts.mu.Unlock()

			metadata, err := ts.Firestore.GetVersion("default", version)
			if err != nil {
				http.Error(w, fmt.Sprintf("Policy version not found: %v", err), http.StatusNotFound)
				return
			}

			// Load from GCS
			loadedEntry, err := ts.Loader.Load(ctx, "default", version, metadata.Hash)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to load policy: %v", err), http.StatusInternalServerError)
				return
			}

			// Add to cache
			ts.Cache.Set(loadedEntry)
			entry = loadedEntry
		}

		// Simulate policy evaluation (OPA would run here)
		// For tests, just return success with version info
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed":        true,
			"policy_version": version,
			"policy_hash":    entry.Hash,
			"cache_hit":      found,
		})
	})

	// Apply middleware: PolicyVersion -> Worker Handler
	return policyVersion.Handler(workerHandler)
}

// Close closes the test server
func (ts *TestServer) Close() {
	ts.Router.Close()
}

// ResetStats resets test tracking stats
func (ts *TestServer) ResetStats() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.VersionUsed = ""
	ts.FetchedFromFS = false
}

// GetVersionUsed returns the captured policy version
func (ts *TestServer) GetVersionUsed() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.VersionUsed
}

// WasFetchedFromFirestore returns whether Firestore was queried
func (ts *TestServer) WasFetchedFromFirestore() bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.FetchedFromFS
}
