package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// RateLimitHeaders contains parsed rate limit header values
type RateLimitHeaders struct {
	Limit     int64
	Remaining int64
	Reset     int64
}

// ParseRateLimitHeaders extracts rate limit headers from an HTTP response
func ParseRateLimitHeaders(headers http.Header) (*RateLimitHeaders, error) {
	limit, err := parseIntHeader(headers, "X-RateLimit-Limit")
	if err != nil {
		return nil, fmt.Errorf("invalid X-RateLimit-Limit header: %w", err)
	}

	remaining, err := parseIntHeader(headers, "X-RateLimit-Remaining")
	if err != nil {
		return nil, fmt.Errorf("invalid X-RateLimit-Remaining header: %w", err)
	}

	reset, err := parseIntHeader(headers, "X-RateLimit-Reset")
	if err != nil {
		return nil, fmt.Errorf("invalid X-RateLimit-Reset header: %w", err)
	}

	return &RateLimitHeaders{
		Limit:     limit,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}

// GetRetryAfter extracts the Retry-After header value (seconds)
func GetRetryAfter(headers http.Header) (int64, error) {
	retryAfter := headers.Get("Retry-After")
	if retryAfter == "" {
		return 0, fmt.Errorf("Retry-After header not found")
	}

	seconds, err := strconv.ParseInt(retryAfter, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid Retry-After header: %w", err)
	}

	return seconds, nil
}

// ResetTime returns the reset time as a time.Time object
func (h *RateLimitHeaders) ResetTime() time.Time {
	return time.Unix(h.Reset, 0)
}

// TimeUntilReset returns the duration until the rate limit resets
func (h *RateLimitHeaders) TimeUntilReset() time.Duration {
	return time.Until(h.ResetTime())
}

// IsExceeded checks if the rate limit has been exceeded
func (h *RateLimitHeaders) IsExceeded() bool {
	return h.Remaining == 0
}

// PercentageUsed returns the percentage of the rate limit used (0-100)
func (h *RateLimitHeaders) PercentageUsed() float64 {
	if h.Limit == 0 {
		return 0
	}
	used := h.Limit - h.Remaining
	return (float64(used) / float64(h.Limit)) * 100
}

// parseIntHeader parses an integer header value
func parseIntHeader(headers http.Header, key string) (int64, error) {
	value := headers.Get(key)
	if value == "" {
		return 0, fmt.Errorf("header not found: %s", key)
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse header %s: %w", key, err)
	}

	return parsed, nil
}

// SetRateLimitHeaders sets rate limit headers on an HTTP response
func SetRateLimitHeaders(w http.ResponseWriter, limit, remaining int64, resetAt time.Time) {
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))
}

// SetRetryAfter sets the Retry-After header on an HTTP response
func SetRetryAfter(w http.ResponseWriter, seconds int64) {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
}
