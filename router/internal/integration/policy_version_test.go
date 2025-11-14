package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stratus-meridian/apx/router/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLatestVersionNoHeader tests that requests without X-Apx-Policy-Version use latest version
func TestLatestVersionNoHeader(t *testing.T) {
	// Setup test server
	ts := NewTestServer()
	defer ts.Close()

	// Add test policy versions to mock Firestore
	ts.Firestore.AddVersion(&PolicyMetadata{
		Name:        "default",
		Version:     "2.1.0",
		Hash:        "hash-v2.1.0",
		CreatedAt:   time.Now(),
		Description: "Latest version",
	})

	// Add corresponding WASM bundle to mock GCS
	wasmBytes := []byte("mock-wasm-bundle-v2.1.0")
	ts.GCS.AddBundle("default", "2.1.0", "hash-v2.1.0", wasmBytes)

	// Make request without X-Apx-Policy-Version header
	req, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that "latest" was used (default)
	assert.Equal(t, "latest", ts.GetVersionUsed())

	// Verify response header shows version used
	assert.Equal(t, "latest", resp.Header.Get("X-Apx-Policy-Version-Used"))

	// Decode response body
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify that latest version (2.1.0) was resolved and used
	assert.Equal(t, "2.1.0", result["policy_version"])
	assert.Equal(t, "hash-v2.1.0", result["policy_hash"])
	assert.True(t, result["allowed"].(bool))

	// Verify Firestore was queried for latest version
	assert.True(t, ts.WasFetchedFromFirestore())
}

// TestExplicitNVersion tests that requests with explicit N version header use that version
func TestExplicitNVersion(t *testing.T) {
	// Setup test server
	ts := NewTestServer()
	defer ts.Close()

	// Add version 2.1.0 (current N version)
	ts.Firestore.AddVersion(&PolicyMetadata{
		Name:        "default",
		Version:     "2.1.0",
		Hash:        "hash-v2.1.0",
		CreatedAt:   time.Now(),
		Description: "Current version",
	})
	ts.GCS.AddBundle("default", "2.1.0", "hash-v2.1.0", []byte("mock-wasm-bundle-v2.1.0"))

	// Make request with explicit version header
	req, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.HeaderPolicyVersion, "2.1.0")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that exact version was used
	assert.Equal(t, "2.1.0", ts.GetVersionUsed())

	// Verify response header
	assert.Equal(t, "2.1.0", resp.Header.Get("X-Apx-Policy-Version-Used"))

	// Decode response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "2.1.0", result["policy_version"])
	assert.Equal(t, "hash-v2.1.0", result["policy_hash"])
	assert.True(t, result["allowed"].(bool))
}

// TestNMinus1VersionCacheHit tests that N-1 version requests use cached version
func TestNMinus1VersionCacheHit(t *testing.T) {
	// Setup test server
	ts := NewTestServer()
	defer ts.Close()

	// Add N and N-1 versions to Firestore
	ts.Firestore.AddVersion(&PolicyMetadata{
		Name:        "default",
		Version:     "2.1.0",
		Hash:        "hash-v2.1.0",
		CreatedAt:   time.Now(),
		Description: "N version",
	})
	ts.Firestore.AddVersion(&PolicyMetadata{
		Name:        "default",
		Version:     "2.0.0",
		Hash:        "hash-v2.0.0",
		CreatedAt:   time.Now().Add(-48 * time.Hour), // N-1 is older
		Description: "N-1 version",
	})

	// Add WASM bundles to GCS
	ts.GCS.AddBundle("default", "2.1.0", "hash-v2.1.0", []byte("mock-wasm-bundle-v2.1.0"))
	ts.GCS.AddBundle("default", "2.0.0", "hash-v2.0.0", []byte("mock-wasm-bundle-v2.0.0"))

	// First request: Load N-1 version into cache
	req1, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req1.Header.Set(middleware.HeaderPolicyVersion, "2.0.0")

	resp1, err := http.DefaultClient.Do(req1)
	require.NoError(t, err)
	resp1.Body.Close()

	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.True(t, ts.WasFetchedFromFirestore()) // First request should fetch

	// Reset stats
	ts.ResetStats()

	// Second request: Should hit cache
	req2, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req2.Header.Set(middleware.HeaderPolicyVersion, "2.0.0")

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Decode response
	var result map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&result)
	require.NoError(t, err)

	// Verify cache hit
	assert.True(t, result["cache_hit"].(bool))
	assert.Equal(t, "2.0.0", result["policy_version"])
	assert.Equal(t, "hash-v2.0.0", result["policy_hash"])

	// Verify Firestore was NOT queried (cache hit)
	assert.False(t, ts.WasFetchedFromFirestore())
}

// TestCacheMissExpiredVersion tests that expired N-1 versions are refetched
func TestCacheMissExpiredVersion(t *testing.T) {
	// Setup test server
	ts := NewTestServer()
	defer ts.Close()

	// Add test policy version
	ts.Firestore.AddVersion(&PolicyMetadata{
		Name:        "default",
		Version:     "2.0.0",
		Hash:        "hash-v2.0.0",
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		Description: "N-1 version",
	})
	ts.GCS.AddBundle("default", "2.0.0", "hash-v2.0.0", []byte("mock-wasm-bundle-v2.0.0"))

	// First request: Load into cache
	req1, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req1.Header.Set(middleware.HeaderPolicyVersion, "2.0.0")

	resp1, err := http.DefaultClient.Do(req1)
	require.NoError(t, err)
	resp1.Body.Close()

	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Verify cache size
	assert.Equal(t, 1, ts.Cache.Size())

	// Simulate TTL expiration (25 hours)
	// Note: In real cache implementation, we'd need to mock time.Now()
	// For this test, we'll manually evict and verify refetch behavior
	evicted := ts.Cache.Evict()
	assert.Equal(t, 0, evicted) // Should not evict yet (within 24h)

	// Force eviction by manually clearing cache to simulate TTL expiration
	// In production, this would happen after 24h TTL
	ts.Cache = NewCache(24 * time.Hour) // Create fresh cache (simulates expiration)

	// Reset stats
	ts.ResetStats()

	// Second request after "expiration": Should refetch
	req2, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req2.Header.Set(middleware.HeaderPolicyVersion, "2.0.0")

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Decode response
	var result map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&result)
	require.NoError(t, err)

	// Verify cache miss (refetched)
	assert.False(t, result["cache_hit"].(bool))
	assert.Equal(t, "2.0.0", result["policy_version"])

	// Verify Firestore was queried again
	assert.True(t, ts.WasFetchedFromFirestore())
}

// TestInvalidVersionFormat tests that malformed version headers return 400
func TestInvalidVersionFormat(t *testing.T) {
	// Setup test server
	ts := NewTestServer()
	defer ts.Close()

	testCases := []struct {
		name          string
		version       string
		expectedError string
	}{
		{
			name:          "invalid_semver",
			version:       "invalid.version",
			expectedError: "Invalid policy version format",
		},
		{
			name:          "missing_patch",
			version:       "2.1",
			expectedError: "Invalid policy version format",
		},
		{
			name:          "negative_version",
			version:       "2.1.-1",
			expectedError: "Invalid policy version format",
		},
		{
			name:          "non_numeric",
			version:       "abc.def.ghi",
			expectedError: "Invalid policy version format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
			require.NoError(t, err)
			req.Header.Set(middleware.HeaderPolicyVersion, tc.version)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return 400 Bad Request
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Read response body
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			bodyText := buf.String()

			// Verify error message
			assert.Contains(t, bodyText, tc.expectedError)
		})
	}
}

// TestCacheEvictionTTL tests that cache eviction works correctly based on TTL
func TestCacheEvictionTTL(t *testing.T) {
	// Setup test server
	ts := NewTestServer()
	defer ts.Close()

	// Add multiple policy versions
	versions := []struct {
		version string
		hash    string
	}{
		{"2.0.0", "hash-v2.0.0"},
		{"2.1.0", "hash-v2.1.0"},
		{"1.9.0", "hash-v1.9.0"},
	}

	for _, v := range versions {
		ts.Firestore.AddVersion(&PolicyMetadata{
			Name:        "default",
			Version:     v.version,
			Hash:        v.hash,
			CreatedAt:   time.Now(),
			Description: "Test version",
		})
		ts.GCS.AddBundle("default", v.version, v.hash, []byte("mock-wasm-bundle-"+v.version))
	}

	// Load all versions into cache
	for _, v := range versions {
		req, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
		require.NoError(t, err)
		req.Header.Set(middleware.HeaderPolicyVersion, v.version)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Verify all versions are cached
	assert.Equal(t, 3, ts.Cache.Size())

	// Check cache keys
	keys := ts.Cache.Keys()
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "default@2.0.0")
	assert.Contains(t, keys, "default@2.1.0")
	assert.Contains(t, keys, "default@1.9.0")

	// Evict expired entries (none should be expired yet)
	evicted := ts.Cache.Evict()
	assert.Equal(t, 0, evicted)
	assert.Equal(t, 3, ts.Cache.Size())

	// Note: To properly test TTL-based eviction, we would need to:
	// 1. Mock time.Now() in the cache implementation
	// 2. Advance time by 25+ hours
	// 3. Verify eviction removes expired entries
	// This is demonstrated conceptually in TestCacheMissExpiredVersion
}

// TestConcurrentRequests tests thread safety with concurrent version requests
func TestConcurrentRequests(t *testing.T) {
	// Setup test server
	ts := NewTestServer()
	defer ts.Close()

	// Add test policies
	ts.Firestore.AddVersion(&PolicyMetadata{
		Name:        "default",
		Version:     "2.1.0",
		Hash:        "hash-v2.1.0",
		CreatedAt:   time.Now(),
		Description: "Current version",
	})
	ts.Firestore.AddVersion(&PolicyMetadata{
		Name:        "default",
		Version:     "2.0.0",
		Hash:        "hash-v2.0.0",
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		Description: "N-1 version",
	})

	ts.GCS.AddBundle("default", "2.1.0", "hash-v2.1.0", []byte("mock-wasm-bundle-v2.1.0"))
	ts.GCS.AddBundle("default", "2.0.0", "hash-v2.0.0", []byte("mock-wasm-bundle-v2.0.0"))

	// Run 100 concurrent requests mixing N and N-1 versions
	const numRequests = 100
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			version := "2.1.0"
			if index%2 == 0 {
				version = "2.0.0" // Alternate between N and N-1
			}

			req, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
			if err != nil {
				results <- err
				return
			}
			req.Header.Set(middleware.HeaderPolicyVersion, version)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- assert.AnError
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		err := <-results
		assert.NoError(t, err, "Request %d failed", i)
	}

	// Verify cache contains both versions
	assert.Equal(t, 2, ts.Cache.Size())
}

// TestVersionNotFound tests handling of non-existent policy versions
func TestVersionNotFound(t *testing.T) {
	// Setup test server
	ts := NewTestServer()
	defer ts.Close()

	// Add only version 2.1.0
	ts.Firestore.AddVersion(&PolicyMetadata{
		Name:        "default",
		Version:     "2.1.0",
		Hash:        "hash-v2.1.0",
		CreatedAt:   time.Now(),
		Description: "Current version",
	})
	ts.GCS.AddBundle("default", "2.1.0", "hash-v2.1.0", []byte("mock-wasm-bundle-v2.1.0"))

	// Request non-existent version
	req, err := http.NewRequest("POST", ts.Router.URL+"/proxy", bytes.NewBuffer([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set(middleware.HeaderPolicyVersion, "1.0.0") // Does not exist

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 404 Not Found
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Read response body
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	bodyText := buf.String()

	// Verify error message
	assert.Contains(t, bodyText, "Policy version not found")
}
