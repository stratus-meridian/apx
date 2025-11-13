package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPolicyVersion_Handler(t *testing.T) {
	tests := []struct {
		name           string
		headerValue    string
		expectedStatus int
		expectedInCtx  string
	}{
		{
			name:           "valid semver version",
			headerValue:    "1.0.0",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "1.0.0",
		},
		{
			name:           "latest version",
			headerValue:    "latest",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "latest",
		},
		{
			name:           "no header defaults to latest",
			headerValue:    "",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "latest",
		},
		{
			name:           "semver with prerelease",
			headerValue:    "1.0.0-beta.1",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "1.0.0-beta.1",
		},
		{
			name:           "invalid version format",
			headerValue:    "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedInCtx:  "",
		},
		{
			name:           "semver with metadata",
			headerValue:    "1.0.0+build.123",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "1.0.0+build.123",
		},
		{
			name:           "major.minor.patch standard format",
			headerValue:    "2.5.10",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "2.5.10",
		},
		{
			name:           "zero major version",
			headerValue:    "0.1.0",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "0.1.0",
		},
		{
			name:           "complex prerelease",
			headerValue:    "1.0.0-alpha.beta.1",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "1.0.0-alpha.beta.1",
		},
		{
			name:           "full semver with prerelease and metadata",
			headerValue:    "1.0.0-rc.1+build.123",
			expectedStatus: http.StatusOK,
			expectedInCtx:  "1.0.0-rc.1+build.123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := NewPolicyVersion()

			// Create test handler that captures context
			var capturedVersion string
			handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedVersion = GetVersionFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set(HeaderPolicyVersion, tt.headerValue)
			}

			// Record response
			rr := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(rr, req)

			// Verify status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Verify context value (if success expected)
			if tt.expectedStatus == http.StatusOK {
				if capturedVersion != tt.expectedInCtx {
					t.Errorf("expected version in context '%s', got '%s'", tt.expectedInCtx, capturedVersion)
				}

				// Verify response header
				usedVersion := rr.Header().Get("X-Apx-Policy-Version-Used")
				if usedVersion != tt.expectedInCtx {
					t.Errorf("expected response header '%s', got '%s'", tt.expectedInCtx, usedVersion)
				}
			}
		})
	}
}

func TestPolicyVersion_isValidVersion(t *testing.T) {
	pv := NewPolicyVersion()

	tests := []struct {
		version string
		valid   bool
	}{
		{"latest", true},
		{"1.0.0", true},
		{"0.0.1", true},
		{"10.20.30", true},
		{"1.0.0-alpha", true},
		{"1.0.0-beta.1", true},
		{"1.0.0+build", true},
		{"1.0.0-rc.1+build.123", true},
		{"999.999.999", true},
		{"0.0.0", true},
		{"1.2.3-alpha.1.2.3", true},
		{"1.2.3+metadata.info", true},
		{"invalid", false},
		{"1.0", false},
		{"v1.0.0", false},
		{"1.0.0.0", false},
		{"", false},
		{"1", false},
		{"1.2.3.4", false},
		{"a.b.c", false},
		{" 1.0.0", false},
		{"1.0.0 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := pv.isValidVersion(tt.version)
			if result != tt.valid {
				t.Errorf("version '%s': expected valid=%v, got %v", tt.version, tt.valid, result)
			}
		})
	}
}

func TestGetVersionFromContext(t *testing.T) {
	// Test with version in context
	ctx := context.WithValue(context.Background(), ContextKeyPolicyVersion, "1.2.3")
	version := GetVersionFromContext(ctx)
	if version != "1.2.3" {
		t.Errorf("expected '1.2.3', got '%s'", version)
	}

	// Test with empty context
	version = GetVersionFromContext(context.Background())
	if version != DefaultPolicyVersion {
		t.Errorf("expected default '%s', got '%s'", DefaultPolicyVersion, version)
	}

	// Test with wrong type in context
	ctx = context.WithValue(context.Background(), ContextKeyPolicyVersion, 123)
	version = GetVersionFromContext(ctx)
	if version != DefaultPolicyVersion {
		t.Errorf("expected default '%s' for wrong type, got '%s'", DefaultPolicyVersion, version)
	}
}

func TestPolicyVersion_CustomDefault(t *testing.T) {
	// Create middleware with custom default
	pv := &PolicyVersion{
		DefaultVersion: "1.0.0",
	}

	handler := pv.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := GetVersionFromContext(r.Context())
		if version != "1.0.0" {
			t.Errorf("expected custom default '1.0.0', got '%s'", version)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestPolicyVersion_HeaderCaseInsensitive(t *testing.T) {
	middleware := NewPolicyVersion()

	testCases := []string{
		"X-Apx-Policy-Version",
		"x-apx-policy-version",
		"X-APX-POLICY-VERSION",
	}

	for _, headerName := range testCases {
		t.Run(headerName, func(t *testing.T) {
			var capturedVersion string
			handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedVersion = GetVersionFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set(headerName, "1.2.3")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if capturedVersion != "1.2.3" {
				t.Errorf("header %s: expected version '1.2.3', got '%s'", headerName, capturedVersion)
			}
		})
	}
}
