package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		expectKey   string
		expectError error
	}{
		{
			name:        "valid live key",
			authHeader:  "Bearer apx_live_0123456789abcdef0123456789abcdef",
			expectKey:   "apx_live_0123456789abcdef0123456789abcdef",
			expectError: nil,
		},
		{
			name:        "valid test key",
			authHeader:  "Bearer apx_test_0123456789abcdef0123456789abcdef",
			expectKey:   "apx_test_0123456789abcdef0123456789abcdef",
			expectError: nil,
		},
		{
			name:        "missing header",
			authHeader:  "",
			expectKey:   "",
			expectError: ErrNoAuthHeader,
		},
		{
			name:        "invalid format - no Bearer",
			authHeader:  "apx_live_0123456789abcdef0123456789abcdef",
			expectKey:   "",
			expectError: ErrInvalidAuthFormat,
		},
		{
			name:        "invalid format - wrong scheme",
			authHeader:  "Basic apx_live_0123456789abcdef0123456789abcdef",
			expectKey:   "",
			expectError: ErrInvalidAuthFormat,
		},
		{
			name:        "empty token",
			authHeader:  "Bearer ",
			expectKey:   "",
			expectError: ErrEmptyAPIKey,
		},
		{
			name:        "whitespace token",
			authHeader:  "Bearer    ",
			expectKey:   "",
			expectError: ErrEmptyAPIKey,
		},
		{
			name:        "case insensitive Bearer",
			authHeader:  "bearer apx_live_0123456789abcdef0123456789abcdef",
			expectKey:   "apx_live_0123456789abcdef0123456789abcdef",
			expectError: nil,
		},
		{
			name:        "Bearer with extra spaces",
			authHeader:  "Bearer   apx_live_0123456789abcdef0123456789abcdef  ",
			expectKey:   "apx_live_0123456789abcdef0123456789abcdef",
			expectError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			key, err := ExtractAPIKey(req)
			if tt.expectError != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.expectError)
				}
				if err != tt.expectError {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if key != tt.expectKey {
					t.Errorf("expected key %q, got %q", tt.expectKey, key)
				}
			}
		})
	}
}

func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectError bool
	}{
		{
			name:        "valid live key",
			apiKey:      "apx_live_0123456789abcdef0123456789abcdef",
			expectError: false,
		},
		{
			name:        "valid test key",
			apiKey:      "apx_test_0123456789abcdef0123456789abcdef",
			expectError: false,
		},
		{
			name:        "valid uppercase hex",
			apiKey:      "apx_live_0123456789ABCDEF0123456789ABCDEF",
			expectError: false,
		},
		{
			name:        "valid mixed case hex",
			apiKey:      "apx_live_0123456789AbCdEf0123456789aBcDeF",
			expectError: false,
		},
		{
			name:        "empty key",
			apiKey:      "",
			expectError: true,
		},
		{
			name:        "invalid prefix",
			apiKey:      "invalid_0123456789abcdef0123456789abcdef",
			expectError: true,
		},
		{
			name:        "too short",
			apiKey:      "apx_live_0123456789abcdef",
			expectError: true,
		},
		{
			name:        "too long",
			apiKey:      "apx_live_0123456789abcdef0123456789abcdef00",
			expectError: true,
		},
		{
			name:        "invalid hex character - space",
			apiKey:      "apx_live_0123456789abcdef 123456789abcdef",
			expectError: true,
		},
		{
			name:        "invalid hex character - g",
			apiKey:      "apx_live_0123456789abcdefg123456789abcdef",
			expectError: true,
		},
		{
			name:        "invalid hex character - special",
			apiKey:      "apx_live_0123456789abcdef-123456789abcdef",
			expectError: true,
		},
		{
			name:        "no prefix underscore",
			apiKey:      "apxlive_0123456789abcdef0123456789abcdef",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKeyFormat(tt.apiKey)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		expectPrefix   string
		expectProd     bool
		expectError    bool
	}{
		{
			name:         "live key",
			apiKey:       "apx_live_0123456789abcdef0123456789abcdef",
			expectPrefix: "apx_live_",
			expectProd:   true,
			expectError:  false,
		},
		{
			name:         "test key",
			apiKey:       "apx_test_0123456789abcdef0123456789abcdef",
			expectPrefix: "apx_test_",
			expectProd:   false,
			expectError:  false,
		},
		{
			name:         "invalid key",
			apiKey:       "invalid_key",
			expectPrefix: "",
			expectProd:   false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseAPIKey(tt.apiKey)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.RawKey != tt.apiKey {
				t.Errorf("expected RawKey %q, got %q", tt.apiKey, info.RawKey)
			}
			if info.Prefix != tt.expectPrefix {
				t.Errorf("expected Prefix %q, got %q", tt.expectPrefix, info.Prefix)
			}
			if info.IsProduction != tt.expectProd {
				t.Errorf("expected IsProduction %v, got %v", tt.expectProd, info.IsProduction)
			}
		})
	}
}

func TestExtractAndValidateAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		authHeader   string
		expectPrefix string
		expectProd   bool
		expectError  bool
	}{
		{
			name:         "valid live key",
			authHeader:   "Bearer apx_live_0123456789abcdef0123456789abcdef",
			expectPrefix: "apx_live_",
			expectProd:   true,
			expectError:  false,
		},
		{
			name:         "valid test key",
			authHeader:   "Bearer apx_test_0123456789abcdef0123456789abcdef",
			expectPrefix: "apx_test_",
			expectProd:   false,
			expectError:  false,
		},
		{
			name:         "missing header",
			authHeader:   "",
			expectPrefix: "",
			expectProd:   false,
			expectError:  true,
		},
		{
			name:         "invalid format",
			authHeader:   "Bearer invalid_key",
			expectPrefix: "",
			expectProd:   false,
			expectError:  true,
		},
		{
			name:         "invalid Bearer format",
			authHeader:   "apx_live_0123456789abcdef0123456789abcdef",
			expectPrefix: "",
			expectProd:   false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			info, err := ExtractAndValidateAPIKey(req)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.Prefix != tt.expectPrefix {
				t.Errorf("expected Prefix %q, got %q", tt.expectPrefix, info.Prefix)
			}
			if info.IsProduction != tt.expectProd {
				t.Errorf("expected IsProduction %v, got %v", tt.expectProd, info.IsProduction)
			}
		})
	}
}

func TestIsHexChar(t *testing.T) {
	tests := []struct {
		char   rune
		expect bool
	}{
		{'0', true},
		{'9', true},
		{'a', true},
		{'f', true},
		{'A', true},
		{'F', true},
		{'g', false},
		{'G', false},
		{' ', false},
		{'-', false},
		{'_', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isHexChar(tt.char)
			if result != tt.expect {
				t.Errorf("isHexChar(%c) = %v, expected %v", tt.char, result, tt.expect)
			}
		})
	}
}

// Benchmark tests
func BenchmarkExtractAPIKey(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer apx_live_0123456789abcdef0123456789abcdef")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractAPIKey(req)
	}
}

func BenchmarkValidateAPIKeyFormat(b *testing.B) {
	apiKey := "apx_live_0123456789abcdef0123456789abcdef"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateAPIKeyFormat(apiKey)
	}
}

func BenchmarkParseAPIKey(b *testing.B) {
	apiKey := "apx_live_0123456789abcdef0123456789abcdef"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseAPIKey(apiKey)
	}
}

func BenchmarkExtractAndValidateAPIKey(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer apx_live_0123456789abcdef0123456789abcdef")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractAndValidateAPIKey(req)
	}
}

// Fuzz test for API key validation
func FuzzValidateAPIKeyFormat(f *testing.F) {
	// Seed corpus with valid and invalid keys
	f.Add("apx_live_0123456789abcdef0123456789abcdef")
	f.Add("apx_test_0123456789abcdef0123456789abcdef")
	f.Add("invalid_key")
	f.Add("")
	f.Add("apx_live_")

	f.Fuzz(func(t *testing.T, apiKey string) {
		// Just ensure it doesn't panic
		_ = ValidateAPIKeyFormat(apiKey)
	})
}
