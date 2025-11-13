package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPolicyVersion_Integration tests the middleware in a chain with other middlewares
func TestPolicyVersion_Integration(t *testing.T) {
	// Create a simple logging middleware for testing
	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In a real scenario, this would log the request
			next.ServeHTTP(w, r)
		})
	}

	// Create policy version middleware
	policyVersionMiddleware := NewPolicyVersion().Handler

	// Create final handler
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := GetVersionFromContext(r.Context())
		w.Header().Set("X-Test-Version", version)
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware chain: logging -> policy version -> handler
	handler := Chain(finalHandler, loggingMiddleware, policyVersionMiddleware)

	// Test with version header
	t.Run("with version in chain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Apx-Policy-Version", "1.5.0")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		// Check both headers are set
		if used := rr.Header().Get("X-Apx-Policy-Version-Used"); used != "1.5.0" {
			t.Errorf("expected X-Apx-Policy-Version-Used '1.5.0', got '%s'", used)
		}

		if test := rr.Header().Get("X-Test-Version"); test != "1.5.0" {
			t.Errorf("expected X-Test-Version '1.5.0', got '%s'", test)
		}
	})

	// Test without version header (should default to latest)
	t.Run("without version in chain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		if used := rr.Header().Get("X-Apx-Policy-Version-Used"); used != "latest" {
			t.Errorf("expected X-Apx-Policy-Version-Used 'latest', got '%s'", used)
		}

		if test := rr.Header().Get("X-Test-Version"); test != "latest" {
			t.Errorf("expected X-Test-Version 'latest', got '%s'", test)
		}
	})
}

// TestPolicyVersion_MiddlewareSignature ensures PolicyVersion.Handler matches Middleware type
func TestPolicyVersion_MiddlewareSignature(t *testing.T) {
	pv := NewPolicyVersion()

	// This will fail to compile if the signature doesn't match
	var _ Middleware = pv.Handler

	t.Log("PolicyVersion.Handler matches Middleware signature")
}
