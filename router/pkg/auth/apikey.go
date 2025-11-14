package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Common errors
var (
	ErrNoAuthHeader      = errors.New("missing Authorization header")
	ErrInvalidAuthFormat = errors.New("invalid Authorization format: expected 'Bearer <token>'")
	ErrInvalidAPIKey     = errors.New("invalid API key format")
	ErrEmptyAPIKey       = errors.New("API key cannot be empty")
)

// APIKey prefix constants
const (
	PrefixLive = "apx_live_"
	PrefixTest = "apx_test_"
)

// APIKeyInfo contains parsed information from an API key
type APIKeyInfo struct {
	RawKey      string // Full API key
	Prefix      string // "apx_live_" or "apx_test_"
	IsProduction bool  // true if live key, false if test key
}

// ExtractAPIKey extracts the API key from the Authorization header.
// Expects format: "Authorization: Bearer apx_live_abc123..."
func ExtractAPIKey(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeader
	}

	// Parse "Bearer <token>" format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", ErrInvalidAuthFormat
	}

	apiKey := strings.TrimSpace(parts[1])
	if apiKey == "" {
		return "", ErrEmptyAPIKey
	}

	return apiKey, nil
}

// ValidateAPIKeyFormat validates the format of an API key.
// Valid formats:
//   - apx_live_<32 hex characters>
//   - apx_test_<32 hex characters>
func ValidateAPIKeyFormat(apiKey string) error {
	if apiKey == "" {
		return ErrEmptyAPIKey
	}

	// Check prefix
	if !strings.HasPrefix(apiKey, PrefixLive) && !strings.HasPrefix(apiKey, PrefixTest) {
		return fmt.Errorf("%w: must start with %s or %s", ErrInvalidAPIKey, PrefixLive, PrefixTest)
	}

	// Check total length: prefix (9 chars) + hex string (32 chars) = 41 chars
	if len(apiKey) != 41 {
		return fmt.Errorf("%w: invalid length %d, expected 41", ErrInvalidAPIKey, len(apiKey))
	}

	// Extract hex portion and validate it's all hex characters
	hexPortion := apiKey[9:] // Skip the 9-char prefix
	for i, c := range hexPortion {
		if !isHexChar(c) {
			return fmt.Errorf("%w: invalid character at position %d: %c", ErrInvalidAPIKey, i+9, c)
		}
	}

	return nil
}

// ParseAPIKey parses an API key and returns structured information.
func ParseAPIKey(apiKey string) (*APIKeyInfo, error) {
	if err := ValidateAPIKeyFormat(apiKey); err != nil {
		return nil, err
	}

	info := &APIKeyInfo{
		RawKey: apiKey,
	}

	if strings.HasPrefix(apiKey, PrefixLive) {
		info.Prefix = PrefixLive
		info.IsProduction = true
	} else {
		info.Prefix = PrefixTest
		info.IsProduction = false
	}

	return info, nil
}

// ExtractAndValidateAPIKey extracts and validates an API key from the request.
// This is a convenience function that combines ExtractAPIKey and ValidateAPIKeyFormat.
func ExtractAndValidateAPIKey(r *http.Request) (*APIKeyInfo, error) {
	apiKey, err := ExtractAPIKey(r)
	if err != nil {
		return nil, err
	}

	return ParseAPIKey(apiKey)
}

// isHexChar checks if a rune is a valid hexadecimal character (0-9, a-f, A-F)
func isHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
