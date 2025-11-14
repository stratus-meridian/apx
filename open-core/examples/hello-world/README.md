# Hello World Example

This example demonstrates a complete working setup of APX Router with a simple backend.

## What's Included

- **backend.go**: Simple HTTP server that echoes request details
- **run.sh**: Script to start both backend and router
- **test.sh**: Script to test the setup with sample requests

## Quick Start

### 1. Start the services

```bash
./run.sh
```

This will start:
- Backend server on port 9000
- APX Router on port 8080 (routing `/api/**` to backend)

### 2. Test the setup

In another terminal:

```bash
./test.sh
```

Or manually:

```bash
# Basic request
curl http://localhost:8080/api/hello

# With API key header (demo mode accepts any key)
curl -H "Authorization: Bearer demo-key-123" \
  http://localhost:8080/api/hello

# Check health
curl http://localhost:8080/health

# Check metrics
curl http://localhost:8080/metrics
```

## Expected Output

**Request**: `curl http://localhost:8080/api/hello`

**Response**:
```json
{
  "message": "Hello from APX Router Open-Core!",
  "path": "/api/hello",
  "method": "GET",
  "timestamp": "2025-01-15T10:30:00Z",
  "headers": {
    "X-Request-Id": ["req_abc123..."],
    "X-Policy-Version": ["default"],
    "User-Agent": ["curl/7.79.1"]
  }
}
```

## Architecture

```
Client --> Router (8080) --> Backend (9000)
           └─ Middleware:
              1. RequestID
              2. TenantContext (demo tenant)
              3. RateLimit (10 req/min)
              4. Logging
              5. Metrics
```

## Rate Limiting Test

The demo tenant has a 10 req/min limit. Test it:

```bash
# Send 15 requests rapidly
for i in {1..15}; do
  curl -s http://localhost:8080/api/test
  echo ""
done
```

After ~10 requests, you should see:

```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Rate limit exceeded. Please retry after 60 seconds."
  }
}
```

## Cleanup

```bash
# Stop the processes (Ctrl+C in the terminal running run.sh)
# Or:
pkill -f "backend.go"
pkill -f "router"
```
