package middleware

import (
	"context"
	"net/http"
	"regexp"
)

const (
	// HeaderPolicyVersion is the header name for policy version
	HeaderPolicyVersion = "X-Apx-Policy-Version"

	// ContextKeyPolicyVersion is the context key for policy version
	ContextKeyPolicyVersion = "apx.policy.version"

	// DefaultPolicyVersion is used when no version is specified
	DefaultPolicyVersion = "latest"
)

var (
	// Semantic version regex: X.Y.Z or X.Y.Z-prerelease
	semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
)

// PolicyVersion extracts and validates the policy version from request headers
type PolicyVersion struct {
	// DefaultVersion is used when header is not present
	DefaultVersion string
}

// NewPolicyVersion creates a new policy version middleware
func NewPolicyVersion() *PolicyVersion {
	return &PolicyVersion{
		DefaultVersion: DefaultPolicyVersion,
	}
}

// Handler returns an HTTP middleware function
func (pv *PolicyVersion) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract version from header
		version := r.Header.Get(HeaderPolicyVersion)

		// Use default if not specified
		if version == "" {
			version = pv.DefaultVersion
		}

		// Validate version format (semver or "latest")
		if !pv.isValidVersion(version) {
			http.Error(w, "Invalid policy version format", http.StatusBadRequest)
			return
		}

		// Add version to request context
		ctx := context.WithValue(r.Context(), ContextKeyPolicyVersion, version)

		// Add version to response header for debugging
		w.Header().Set("X-Apx-Policy-Version-Used", version)

		// Continue with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isValidVersion checks if version is valid semver or "latest"
func (pv *PolicyVersion) isValidVersion(version string) bool {
	// "latest" is always valid
	if version == "latest" {
		return true
	}

	// Check if it's valid semver
	return semverRegex.MatchString(version)
}

// GetVersionFromContext extracts policy version from request context
func GetVersionFromContext(ctx context.Context) string {
	if version, ok := ctx.Value(ContextKeyPolicyVersion).(string); ok {
		return version
	}
	return DefaultPolicyVersion
}
