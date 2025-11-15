# APX Router Health Check

This package implements a comprehensive health check endpoint for the APX router that matches the portal's expected schema.

## Overview

The health endpoint returns detailed information about the router's operational status, including individual component health for Firestore, Pub/Sub, and BigQuery.

## Response Schema

```json
{
  "status": "healthy" | "degraded" | "down",
  "version": "string",
  "timestamp": "ISO 8601 timestamp",
  "components": {
    "firestore": "healthy" | "degraded" | "down",
    "pubsub": "healthy" | "degraded" | "down",
    "bigquery": "healthy" | "degraded" | "down"
  }
}
```

## Component Health Checks

### Firestore
- **Healthy**: Policy store is initialized and ready
- **Down**: Policy store is nil or not ready

Firestore is used for storing and retrieving access policies. The health check verifies the policy store is operational.

### Pub/Sub
- **Healthy**: Topic exists and responds to config requests within 1 second
- **Degraded**: Topic exists but config request times out or fails
- **Down**: Topic is nil

Pub/Sub is critical for async message routing. The health check performs a lightweight config check with a 1-second timeout.

### BigQuery
- **Healthy**: Observability system initialized successfully
- **Degraded**: Observability failed to initialize

BigQuery is used indirectly through observability for metrics and logs. Since it's optional for router operation, failures result in degraded rather than down status.

## Overall Status Logic

```
overall_status =
  if (firestore == down OR pubsub == down) -> "down"
  else if (any component == degraded) -> "degraded"
  else -> "healthy"
```

Critical components (Firestore, Pub/Sub) being down will mark the entire service as down. Any component degradation results in overall degraded status.

## HTTP Status Codes

- **200 OK**: Service is healthy or degraded
- **503 Service Unavailable**: Service is down

## Version Information

The version is read from the `VERSION` environment variable. If not set, defaults to `"dev"`.

## Usage

```go
import (
    "github.com/apx/router/pkg/health"
)

// Create health checker with component clients
healthChecker := health.NewChecker(policyStore, pubsubTopic, observabilityInit, logger)

// Register HTTP handler
r.HandleFunc("/health", healthChecker.Handler()).Methods(http.MethodGet)
```

## Performance

Health checks are designed to be fast and non-blocking:
- Firestore: Simple ready state check (<1ms)
- Pub/Sub: Config request with 1-second timeout
- BigQuery: Boolean flag check (<1ms)

Total health check time: <100ms in normal conditions

## Portal Integration

This implementation matches the Zod schema defined in `.private/portal/lib/apx-client.ts`:

```typescript
export const HealthSchema = z.object({
  status: z.enum(['healthy', 'degraded', 'down']),
  version: z.string(),
  timestamp: z.string(),
  components: z.object({
    firestore: z.enum(['healthy', 'degraded', 'down']),
    pubsub: z.enum(['healthy', 'degraded', 'down']),
    bigquery: z.enum(['healthy', 'degraded', 'down']),
  }),
})
```

## Testing

Run tests with:
```bash
cd router && go test ./pkg/health -v
```

Test coverage includes:
- JSON schema validation
- Overall status logic
- Individual component checks
- HTTP handler behavior
- Version and timestamp formatting
