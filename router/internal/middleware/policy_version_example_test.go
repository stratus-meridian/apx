package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/stratus-meridian/apx/router/internal/middleware"
)

// Example demonstrates basic usage of the PolicyVersion middleware
func Example_policyVersion_basic() {
	// Create middleware
	policyVersion := middleware.NewPolicyVersion()

	// Create a handler that uses the version
	handler := policyVersion.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := middleware.GetVersionFromContext(r.Context())
		fmt.Fprintf(w, "Using policy version: %s", version)
	}))

	// Make request with version header
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Apx-Policy-Version", "1.2.3")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	fmt.Println(rr.Body.String())
	// Output: Using policy version: 1.2.3
}

// Example demonstrates default version behavior
func Example_policyVersion_default() {
	policyVersion := middleware.NewPolicyVersion()

	handler := policyVersion.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := middleware.GetVersionFromContext(r.Context())
		fmt.Fprintf(w, "Using policy version: %s", version)
	}))

	// Make request WITHOUT version header
	req := httptest.NewRequest("GET", "/api/test", nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	fmt.Println(rr.Body.String())
	// Output: Using policy version: latest
}

// Example demonstrates invalid version handling
func Example_policyVersion_invalid() {
	policyVersion := middleware.NewPolicyVersion()

	handler := policyVersion.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make request with invalid version
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Apx-Policy-Version", "invalid-version")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	fmt.Printf("Status: %d\n", rr.Code)
	fmt.Printf("Body: %s", rr.Body.String())
	// Output:
	// Status: 400
	// Body: Invalid policy version format
}

// Example demonstrates prerelease version support
func Example_policyVersion_prerelease() {
	policyVersion := middleware.NewPolicyVersion()

	handler := policyVersion.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := middleware.GetVersionFromContext(r.Context())
		fmt.Fprintf(w, "Using policy version: %s", version)
	}))

	// Make request with prerelease version
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Apx-Policy-Version", "2.0.0-beta.1")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	fmt.Println(rr.Body.String())
	// Output: Using policy version: 2.0.0-beta.1
}
