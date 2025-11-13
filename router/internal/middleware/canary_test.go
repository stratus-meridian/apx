package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCanary_Handler(t *testing.T) {
	decider := func(ctx context.Context, policyName, tenantID string) (string, bool, error) {
		// Simple mock: tenant-canary uses canary
		if tenantID == "tenant-canary" {
			return "1.1.0", true, nil
		}
		return "1.0.0", false, nil
	}

	canary := NewCanary(decider)

	tests := []struct {
		name           string
		tenantID       string
		expectedCanary bool
		expectedVer    string
	}{
		{"stable tenant", "tenant-stable", false, "1.0.0"},
		{"canary tenant", "tenant-canary", true, "1.1.0"},
		{"no tenant header", "", false, "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := canary.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				isCanary := IsCanary(r.Context())
				if isCanary != tt.expectedCanary {
					t.Errorf("expected canary=%v, got %v", tt.expectedCanary, isCanary)
				}

				version := GetCanaryVersion(r.Context())
				if version != tt.expectedVer {
					t.Errorf("expected version=%s, got %s", tt.expectedVer, version)
				}

				// Check response headers
				canaryHeader := w.Header().Get("X-Apx-Canary")
				expectedHeader := "false"
				if tt.expectedCanary {
					expectedHeader = "true"
				}
				if canaryHeader != expectedHeader {
					t.Errorf("expected X-Apx-Canary=%s, got %s", expectedHeader, canaryHeader)
				}
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.tenantID != "" {
				req.Header.Set(HeaderTenantID, tt.tenantID)
			}

			// Set policy version to "latest" to trigger canary check
			ctx := context.WithValue(req.Context(), ContextKeyPolicyVersion, "latest")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		})
	}
}

func TestCanary_NoDecider(t *testing.T) {
	// Test with nil decider - should pass through without error
	canary := NewCanary(nil)

	handler := canary.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should reach here without panic
		isCanary := IsCanary(r.Context())
		if isCanary {
			t.Error("expected isCanary=false with nil decider")
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), ContextKeyPolicyVersion, "latest")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestCanary_NonLatestVersion(t *testing.T) {
	// Test that canary decider is not called for specific versions
	deciderCalled := false
	decider := func(ctx context.Context, policyName, tenantID string) (string, bool, error) {
		deciderCalled = true
		return "1.1.0", true, nil
	}

	canary := NewCanary(decider)

	handler := canary.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deciderCalled {
			t.Error("decider should not be called for specific version")
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(HeaderTenantID, "tenant-1")

	// Set specific version (not "latest")
	ctx := context.WithValue(req.Context(), ContextKeyPolicyVersion, "1.0.0")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestCanary_IsCanary(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected bool
	}{
		{"canary true", context.WithValue(context.Background(), ContextKeyCanary, true), true},
		{"canary false", context.WithValue(context.Background(), ContextKeyCanary, false), false},
		{"no value", context.Background(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCanary(tt.ctx)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCanary_GetCanaryVersion(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{"with version", context.WithValue(context.Background(), ContextKeyCanaryVersion, "1.1.0"), "1.1.0"},
		{"no version", context.Background(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCanaryVersion(tt.ctx)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCanary_TenantStickiness(t *testing.T) {
	// Mock decider that uses consistent hashing
	decider := func(ctx context.Context, policyName, tenantID string) (string, bool, error) {
		// Simple hash: if tenant ID ends with even number, use canary
		lastChar := tenantID[len(tenantID)-1]
		isEven := (lastChar-'0')%2 == 0
		if isEven {
			return "1.1.0", true, nil
		}
		return "1.0.0", false, nil
	}

	canary := NewCanary(decider)

	// Test multiple requests for same tenants
	tenants := []string{"tenant-1", "tenant-2", "tenant-3", "tenant-4"}
	results := make(map[string]bool)

	// First pass: record results
	for _, tenant := range tenants {
		handler := canary.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results[tenant] = IsCanary(r.Context())
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(HeaderTenantID, tenant)
		ctx := context.WithValue(req.Context(), ContextKeyPolicyVersion, "latest")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// Second pass: verify consistency
	for _, tenant := range tenants {
		handler := canary.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isCanary := IsCanary(r.Context())
			expected := results[tenant]
			if isCanary != expected {
				t.Errorf("tenant %s got inconsistent result: expected %v, got %v",
					tenant, expected, isCanary)
			}
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(HeaderTenantID, tenant)
		ctx := context.WithValue(req.Context(), ContextKeyPolicyVersion, "latest")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
