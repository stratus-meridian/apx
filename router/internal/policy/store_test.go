package policy

import (
	"context"
	"testing"
	"time"

	"github.com/stratus-meridian/apx/router/internal/config"
	"go.uber.org/zap"
)

// TestCanarySelection verifies canary traffic distribution logic
func TestCanarySelection(t *testing.T) {
	// Create a test store with in-memory cache
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		PolicyStoreType: "memory",
	}

	store := &Store{
		cfg:    cfg,
		logger: logger,
		cache:  make(map[string]*PolicyBundle),
		ready:  true,
	}

	// Add stable version (100%)
	stableBundle := &PolicyBundle{
		Name:             "test-api",
		Version:          "v1.0.0",
		CanaryPercentage: 100,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	store.cache["test-api@v1.0.0"] = stableBundle

	// Add canary version (20%)
	canaryBundle := &PolicyBundle{
		Name:             "test-api",
		Version:          "v2.0.0",
		CanaryPercentage: 20,
		StableVersion:    "v1.0.0",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	store.cache["test-api@v2.0.0"] = canaryBundle

	ctx := context.Background()

	t.Run("CanaryWeight0ShouldUseCanary", func(t *testing.T) {
		// Weight 0 should go to canary (0 < 20)
		policy, ref, err := store.GetForRequest(ctx, "test-api", 0)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if policy.Version != "v2.0.0" {
			t.Errorf("Expected canary version v2.0.0, got: %s", policy.Version)
		}
		if ref != "test-api@v2.0.0" {
			t.Errorf("Expected ref test-api@v2.0.0, got: %s", ref)
		}
	})

	t.Run("CanaryWeight19ShouldUseCanary", func(t *testing.T) {
		// Weight 19 should go to canary (19 < 20)
		policy, ref, err := store.GetForRequest(ctx, "test-api", 19)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if policy.Version != "v2.0.0" {
			t.Errorf("Expected canary version v2.0.0, got: %s", policy.Version)
		}
		if ref != "test-api@v2.0.0" {
			t.Errorf("Expected ref test-api@v2.0.0, got: %s", ref)
		}
	})

	t.Run("CanaryWeight20ShouldUseStable", func(t *testing.T) {
		// Weight 20 should go to stable (20 >= 20)
		policy, ref, err := store.GetForRequest(ctx, "test-api", 20)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if policy.Version != "v1.0.0" {
			t.Errorf("Expected stable version v1.0.0, got: %s", policy.Version)
		}
		if ref != "test-api@v1.0.0" {
			t.Errorf("Expected ref test-api@v1.0.0, got: %s", ref)
		}
	})

	t.Run("CanaryWeight99ShouldUseStable", func(t *testing.T) {
		// Weight 99 should go to stable (99 >= 20)
		policy, ref, err := store.GetForRequest(ctx, "test-api", 99)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if policy.Version != "v1.0.0" {
			t.Errorf("Expected stable version v1.0.0, got: %s", policy.Version)
		}
		if ref != "test-api@v1.0.0" {
			t.Errorf("Expected ref test-api@v1.0.0, got: %s", ref)
		}
	})
}

// TestCanaryDistribution verifies traffic distribution accuracy
func TestCanaryDistribution(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		PolicyStoreType: "memory",
	}

	store := &Store{
		cfg:    cfg,
		logger: logger,
		cache:  make(map[string]*PolicyBundle),
		ready:  true,
	}

	// Add stable and canary versions
	store.cache["test-api@v1.0.0"] = &PolicyBundle{
		Name:             "test-api",
		Version:          "v1.0.0",
		CanaryPercentage: 100,
	}
	store.cache["test-api@v2.0.0"] = &PolicyBundle{
		Name:             "test-api",
		Version:          "v2.0.0",
		CanaryPercentage: 30,
		StableVersion:    "v1.0.0",
	}

	ctx := context.Background()
	numRequests := 1000
	canaryHits := 0

	// Simulate requests with uniform distribution of weights
	for weight := 0; weight < 100; weight++ {
		for i := 0; i < numRequests/100; i++ {
			policy, _, err := store.GetForRequest(ctx, "test-api", weight)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if policy.Version == "v2.0.0" {
				canaryHits++
			}
		}
	}

	// Expected: 30% of 1000 = 300
	// Allow ±2% tolerance = 280-320
	expectedCanaryHits := 300
	tolerance := 20 // 2% of 1000

	if canaryHits < expectedCanaryHits-tolerance || canaryHits > expectedCanaryHits+tolerance {
		t.Errorf("Canary distribution outside tolerance. Expected: %d±%d, Got: %d",
			expectedCanaryHits, tolerance, canaryHits)
	}

	actualPercent := float64(canaryHits) / float64(numRequests) * 100
	t.Logf("Canary hit rate: %.2f%% (expected: 30%%)", actualPercent)
}

// TestNoCanaryDeployment verifies behavior when no canary is active
func TestNoCanaryDeployment(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		PolicyStoreType: "memory",
	}

	store := &Store{
		cfg:    cfg,
		logger: logger,
		cache:  make(map[string]*PolicyBundle),
		ready:  true,
	}

	// Only stable version
	store.cache["test-api@v1.0.0"] = &PolicyBundle{
		Name:             "test-api",
		Version:          "v1.0.0",
		CanaryPercentage: 100,
	}

	ctx := context.Background()

	// All weights should return stable version
	for weight := 0; weight < 100; weight++ {
		policy, ref, err := store.GetForRequest(ctx, "test-api", weight)
		if err != nil {
			t.Fatalf("Unexpected error at weight %d: %v", weight, err)
		}
		if policy.Version != "v1.0.0" {
			t.Errorf("Expected stable v1.0.0 at weight %d, got: %s", weight, policy.Version)
		}
		if ref != "test-api@v1.0.0" {
			t.Errorf("Expected ref test-api@v1.0.0 at weight %d, got: %s", weight, ref)
		}
	}
}

// TestListVersions verifies version listing
func TestListVersions(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		PolicyStoreType: "memory",
	}

	store := &Store{
		cfg:    cfg,
		logger: logger,
		cache:  make(map[string]*PolicyBundle),
		ready:  true,
	}

	// Add multiple versions
	store.cache["test-api@v1.0.0"] = &PolicyBundle{
		Name:    "test-api",
		Version: "v1.0.0",
	}
	store.cache["test-api@v2.0.0"] = &PolicyBundle{
		Name:    "test-api",
		Version: "v2.0.0",
	}
	store.cache["other-api@v1.0.0"] = &PolicyBundle{
		Name:    "other-api",
		Version: "v1.0.0",
	}

	ctx := context.Background()

	versions, err := store.ListVersions(ctx, "test-api")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got: %d", len(versions))
	}

	// Verify both versions are for test-api
	for _, v := range versions {
		if v.Name != "test-api" {
			t.Errorf("Expected name test-api, got: %s", v.Name)
		}
	}
}

// TestPolicyNotFound verifies error handling
func TestPolicyNotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		PolicyStoreType: "memory",
	}

	store := &Store{
		cfg:    cfg,
		logger: logger,
		cache:  make(map[string]*PolicyBundle),
		ready:  true,
	}

	ctx := context.Background()

	_, _, err := store.GetForRequest(ctx, "nonexistent-api", 50)
	if err == nil {
		t.Error("Expected error for nonexistent policy, got nil")
	}
}
