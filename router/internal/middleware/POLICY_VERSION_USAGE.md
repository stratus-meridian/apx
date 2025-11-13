# Policy Version Middleware Usage

## Overview

The PolicyVersion middleware extracts and validates policy versions from incoming HTTP requests, ensuring that requests are processed with the correct policy version throughout the APX system.

## Features

- Extracts policy version from `X-Apx-Policy-Version` header
- Defaults to "latest" when header is not present
- Validates semantic versioning format (X.Y.Z)
- Supports prerelease versions (e.g., 1.0.0-beta.1)
- Supports build metadata (e.g., 1.0.0+build.123)
- Adds version to request context for downstream use
- Returns version used in response header for debugging
- Case-insensitive header matching

## Usage in Router

```go
package main

import (
    "net/http"
    "github.com/apx/router/internal/middleware"
)

func main() {
    // Create middleware
    policyVersion := middleware.NewPolicyVersion()

    // Wrap your handler
    mux := http.NewServeMux()
    mux.HandleFunc("/api/v1/test", handleRequest)

    // Apply middleware
    handler := policyVersion.Handler(mux)

    http.ListenAndServe(":8080", handler)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Get version from context
    version := middleware.GetVersionFromContext(r.Context())

    // Use version when forwarding to workers
    log.Printf("Processing request with policy version: %s", version)

    // Version is now available for Pub/Sub attributes, tracing, etc.
}
```

## Request Examples

### With Semantic Version Header

```bash
curl -H "X-Apx-Policy-Version: 1.2.3" http://localhost:8080/api/v1/test
```

**Response Headers:**
```
X-Apx-Policy-Version-Used: 1.2.3
```

### With "latest" Version

```bash
curl -H "X-Apx-Policy-Version: latest" http://localhost:8080/api/v1/test
```

**Response Headers:**
```
X-Apx-Policy-Version-Used: latest
```

### Without Header (defaults to "latest")

```bash
curl http://localhost:8080/api/v1/test
```

**Response Headers:**
```
X-Apx-Policy-Version-Used: latest
```

### With Prerelease Version

```bash
curl -H "X-Apx-Policy-Version: 2.0.0-beta.1" http://localhost:8080/api/v1/test
```

**Response Headers:**
```
X-Apx-Policy-Version-Used: 2.0.0-beta.1
```

### Invalid Version (Returns 400)

```bash
curl -H "X-Apx-Policy-Version: invalid" http://localhost:8080/api/v1/test
```

**Response:**
```
HTTP/1.1 400 Bad Request
Invalid policy version format
```

## Valid Version Formats

### Accepted Formats

- `latest` - Special keyword for the latest version
- `1.0.0` - Standard semantic version
- `0.1.0` - Zero major version (pre-1.0 releases)
- `1.0.0-alpha` - Prerelease version
- `1.0.0-beta.1` - Prerelease with numeric suffix
- `1.0.0+build.123` - Version with build metadata
- `1.0.0-rc.1+build.123` - Full semver with prerelease and metadata

### Rejected Formats

- `invalid` - Not a valid semver
- `1.0` - Missing patch version
- `v1.0.0` - "v" prefix not allowed
- `1.0.0.0` - Too many version parts
- Empty string - Must provide a value

## Integration with Pub/Sub

When forwarding requests to workers via Pub/Sub, include version in message attributes:

```go
import (
    "cloud.google.com/go/pubsub"
    "github.com/apx/router/internal/middleware"
)

func forwardToWorker(r *http.Request, topic *pubsub.Topic) error {
    // Extract version from context
    version := middleware.GetVersionFromContext(r.Context())

    // Create message with version attribute
    message := &pubsub.Message{
        Data: requestData,
        Attributes: map[string]string{
            "policy_version": version,
            "request_id":     requestID,
            "tenant_id":      tenantID,
        },
    }

    // Publish message
    result := topic.Publish(r.Context(), message)
    _, err := result.Get(r.Context())
    return err
}
```

## Integration with Tracing

Add version to trace spans for observability:

```go
import (
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    "github.com/apx/router/internal/middleware"
)

func addTracingAttributes(r *http.Request) {
    version := middleware.GetVersionFromContext(r.Context())

    span := trace.SpanFromContext(r.Context())
    span.SetAttributes(
        attribute.String("policy.version", version),
        attribute.String("apx.version", version),
    )
}
```

## Integration with Metrics

Include version in metrics for monitoring:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/apx/router/internal/middleware"
)

var requestCounter = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "apx_requests_total",
        Help: "Total number of requests",
    },
    []string{"policy_version", "status"},
)

func recordMetric(r *http.Request, status int) {
    version := middleware.GetVersionFromContext(r.Context())
    requestCounter.WithLabelValues(version, fmt.Sprintf("%d", status)).Inc()
}
```

## Middleware Chaining

The policy version middleware should be added early in the middleware chain, after request ID but before policy loading:

```go
handler := middleware.Chain(
    middleware.RequestID(),          // Generate request ID
    middleware.PolicyVersion(),      // Extract policy version (NEW)
    middleware.Logging(),            // Log requests
    middleware.Tracing(),            // Add tracing
    middleware.RateLimit(),          // Rate limiting
    middleware.PolicyLoader(),       // Load policy (uses version)
)(yourHandler)
```

## Custom Default Version

You can customize the default version if needed:

```go
policyVersion := &middleware.PolicyVersion{
    DefaultVersion: "1.0.0", // Custom default instead of "latest"
}
handler := policyVersion.Handler(mux)
```

## Constants Reference

```go
// Header name for policy version
middleware.HeaderPolicyVersion = "X-Apx-Policy-Version"

// Context key for policy version
middleware.ContextKeyPolicyVersion = "apx.policy.version"

// Default policy version
middleware.DefaultPolicyVersion = "latest"
```

## Testing

The middleware includes comprehensive tests covering:

- Valid semver versions (1.0.0, 2.5.10, etc.)
- "latest" keyword
- Prerelease versions (1.0.0-beta.1)
- Build metadata (1.0.0+build.123)
- Invalid formats (rejected with 400)
- Missing header (defaults to "latest")
- Case-insensitive header matching
- Context propagation
- Response header setting
- Custom default versions

Run tests:

```bash
cd /Users/agentsy/APILEE/router
go test ./internal/middleware/policy_version*.go -v
go test ./internal/middleware/policy_version*.go -cover
```

## Error Handling

The middleware returns `400 Bad Request` for invalid version formats. The response body contains:

```
Invalid policy version format
```

Clients should handle this error and provide a valid version format.

## Security Considerations

- Version validation prevents injection attacks through malformed headers
- Only alphanumeric characters, dots, hyphens, and plus signs are allowed
- Maximum reasonable length is enforced by regex complexity limits
- "latest" is the only special keyword accepted

## Future Enhancements

Potential future additions:

- Version range support (e.g., ">=1.0.0 <2.0.0")
- Version deprecation warnings
- Version-specific rate limiting
- Version usage analytics
- Automatic version resolution from tenant configuration
