package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseRateLimitHeaders tests parsing of rate limit headers
func TestParseRateLimitHeaders(t *testing.T) {
	tests := []struct {
		name          string
		headers       http.Header
		expectedLimit int64
		expectedRem   int64
		expectedReset int64
		expectError   bool
	}{
		{
			name: "valid headers",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("X-RateLimit-Limit", "100")
				h.Set("X-RateLimit-Remaining", "50")
				h.Set("X-RateLimit-Reset", "1699564800")
				return h
			}(),
			expectedLimit: 100,
			expectedRem:   50,
			expectedReset: 1699564800,
			expectError:   false,
		},
		{
			name: "missing limit header",
			headers: http.Header{
				"X-RateLimit-Remaining": []string{"50"},
				"X-RateLimit-Reset":     []string{"1699564800"},
			},
			expectError: true,
		},
		{
			name: "missing remaining header",
			headers: http.Header{
				"X-RateLimit-Limit": []string{"100"},
				"X-RateLimit-Reset": []string{"1699564800"},
			},
			expectError: true,
		},
		{
			name: "missing reset header",
			headers: http.Header{
				"X-RateLimit-Limit":     []string{"100"},
				"X-RateLimit-Remaining": []string{"50"},
			},
			expectError: true,
		},
		{
			name: "invalid limit value",
			headers: http.Header{
				"X-RateLimit-Limit":     []string{"invalid"},
				"X-RateLimit-Remaining": []string{"50"},
				"X-RateLimit-Reset":     []string{"1699564800"},
			},
			expectError: true,
		},
		{
			name: "zero values",
			headers: func() http.Header {
				h := http.Header{}
				h.Set("X-RateLimit-Limit", "0")
				h.Set("X-RateLimit-Remaining", "0")
				h.Set("X-RateLimit-Reset", "0")
				return h
			}(),
			expectedLimit: 0,
			expectedRem:   0,
			expectedReset: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRateLimitHeaders(tt.headers)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedLimit, result.Limit)
				assert.Equal(t, tt.expectedRem, result.Remaining)
				assert.Equal(t, tt.expectedReset, result.Reset)
			}
		})
	}
}

// TestGetRetryAfter tests parsing of Retry-After header
func TestGetRetryAfter(t *testing.T) {
	tests := []struct {
		name        string
		headers     http.Header
		expected    int64
		expectError bool
	}{
		{
			name:        "valid retry after",
			headers:     http.Header{"Retry-After": []string{"60"}},
			expected:    60,
			expectError: false,
		},
		{
			name:        "missing retry after",
			headers:     http.Header{},
			expectError: true,
		},
		{
			name:        "invalid retry after",
			headers:     http.Header{"Retry-After": []string{"invalid"}},
			expectError: true,
		},
		{
			name:        "zero retry after",
			headers:     http.Header{"Retry-After": []string{"0"}},
			expected:    0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetRetryAfter(tt.headers)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestRateLimitHeaders_ResetTime tests ResetTime method
func TestRateLimitHeaders_ResetTime(t *testing.T) {
	resetUnix := time.Now().Add(5 * time.Minute).Unix()
	headers := &RateLimitHeaders{
		Limit:     100,
		Remaining: 50,
		Reset:     resetUnix,
	}

	resetTime := headers.ResetTime()
	assert.Equal(t, resetUnix, resetTime.Unix())
}

// TestRateLimitHeaders_TimeUntilReset tests TimeUntilReset method
func TestRateLimitHeaders_TimeUntilReset(t *testing.T) {
	resetAt := time.Now().Add(5 * time.Minute)
	headers := &RateLimitHeaders{
		Limit:     100,
		Remaining: 50,
		Reset:     resetAt.Unix(),
	}

	duration := headers.TimeUntilReset()

	// Should be approximately 5 minutes (allow 1 second margin for test execution)
	assert.InDelta(t, 5*time.Minute, duration, float64(1*time.Second))
}

// TestRateLimitHeaders_IsExceeded tests IsExceeded method
func TestRateLimitHeaders_IsExceeded(t *testing.T) {
	tests := []struct {
		name      string
		remaining int64
		expected  bool
	}{
		{
			name:      "not exceeded",
			remaining: 50,
			expected:  false,
		},
		{
			name:      "exceeded",
			remaining: 0,
			expected:  true,
		},
		{
			name:      "one remaining",
			remaining: 1,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := &RateLimitHeaders{
				Limit:     100,
				Remaining: tt.remaining,
				Reset:     time.Now().Add(time.Minute).Unix(),
			}

			assert.Equal(t, tt.expected, headers.IsExceeded())
		})
	}
}

// TestRateLimitHeaders_PercentageUsed tests PercentageUsed method
func TestRateLimitHeaders_PercentageUsed(t *testing.T) {
	tests := []struct {
		name      string
		limit     int64
		remaining int64
		expected  float64
	}{
		{
			name:      "50% used",
			limit:     100,
			remaining: 50,
			expected:  50.0,
		},
		{
			name:      "0% used",
			limit:     100,
			remaining: 100,
			expected:  0.0,
		},
		{
			name:      "100% used",
			limit:     100,
			remaining: 0,
			expected:  100.0,
		},
		{
			name:      "25% used",
			limit:     100,
			remaining: 75,
			expected:  25.0,
		},
		{
			name:      "zero limit",
			limit:     0,
			remaining: 0,
			expected:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := &RateLimitHeaders{
				Limit:     tt.limit,
				Remaining: tt.remaining,
				Reset:     time.Now().Add(time.Minute).Unix(),
			}

			assert.Equal(t, tt.expected, headers.PercentageUsed())
		})
	}
}

// TestSetRateLimitHeaders tests SetRateLimitHeaders function
func TestSetRateLimitHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	resetAt := time.Now().Add(time.Minute)

	SetRateLimitHeaders(w, 100, 75, resetAt)

	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "75", w.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, fmt.Sprintf("%d", resetAt.Unix()), w.Header().Get("X-RateLimit-Reset"))
}

// TestSetRetryAfter tests SetRetryAfter function
func TestSetRetryAfter(t *testing.T) {
	w := httptest.NewRecorder()

	SetRetryAfter(w, 60)

	assert.Equal(t, "60", w.Header().Get("Retry-After"))
}

// TestRateLimitHeaders_RoundTrip tests full round trip of setting and parsing headers
func TestRateLimitHeaders_RoundTrip(t *testing.T) {
	w := httptest.NewRecorder()
	resetAt := time.Now().Add(5 * time.Minute)

	// Set headers
	SetRateLimitHeaders(w, 1000, 750, resetAt)
	SetRetryAfter(w, 300)

	// Parse headers
	parsed, err := ParseRateLimitHeaders(w.Header())
	require.NoError(t, err)

	assert.Equal(t, int64(1000), parsed.Limit)
	assert.Equal(t, int64(750), parsed.Remaining)
	assert.Equal(t, resetAt.Unix(), parsed.Reset)

	retryAfter, err := GetRetryAfter(w.Header())
	require.NoError(t, err)
	assert.Equal(t, int64(300), retryAfter)
}

// TestRateLimitHeaders_Integration tests headers in actual HTTP response
func TestRateLimitHeaders_Integration(t *testing.T) {
	resetAt := time.Now().Add(time.Minute)

	// Create a test server that sets rate limit headers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetRateLimitHeaders(w, 100, 95, resetAt)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Parse the headers
	parsed, err := ParseRateLimitHeaders(rr.Header())
	require.NoError(t, err)

	assert.Equal(t, int64(100), parsed.Limit)
	assert.Equal(t, int64(95), parsed.Remaining)
	assert.Equal(t, resetAt.Unix(), parsed.Reset)
	assert.False(t, parsed.IsExceeded())
	assert.Equal(t, 5.0, parsed.PercentageUsed())
}
