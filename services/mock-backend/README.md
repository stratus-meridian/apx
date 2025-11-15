# APX Mock Backend

A lightweight mock backend service similar to [mocktarget.apigee.net](https://mocktarget.apigee.net) for testing APX gateway functionality.

## Features

- ✅ Echo endpoints (return request details)
- ✅ JSON/XML responses
- ✅ Delay simulation
- ✅ Status code testing
- ✅ SSE streaming
- ✅ Header inspection
- ✅ IP detection
- ✅ Health checks

## Available Endpoints

```
GET  /              - Landing page with endpoint list
GET  /health        - Health check (Cloud Run ready)
ANY  /echo          - Echo request details (method, headers, body)
GET  /json          - Return sample JSON
GET  /xml           - Return sample XML
GET  /ip            - Return client IP address
GET  /headers       - Return all request headers
GET  /user-agent    - Return User-Agent header
GET  /delay/:N      - Delay response by N seconds (max 10)
ANY  /status/:code  - Return specified HTTP status code
GET  /uuid          - Generate a UUID
GET  /stream/:N     - SSE stream N events
GET  /bytes/:N      - Return N random bytes (max 10KB)
```

## Local Development

```bash
cd services/mock-backend

# Run locally
go run main.go

# Test endpoints
curl http://localhost:8080/
curl http://localhost:8080/echo
curl http://localhost:8080/json
curl http://localhost:8080/delay/2
curl http://localhost:8080/status/404
```

## Docker Build

```bash
# From repo root
docker build --platform linux/amd64 -f services/mock-backend/Dockerfile -t mock-backend .

# Run container
docker run -p 8080:8080 mock-backend
```

## Cloud Run Deployment

```bash
# Build and push
docker build --platform linux/amd64 \
  -f services/mock-backend/Dockerfile \
  -t us-central1-docker.pkg.dev/apx-build-478003/apx-containers/mock-backend:latest .
  
docker push us-central1-docker.pkg.dev/apx-build-478003/apx-containers/mock-backend:latest

# Deploy to Cloud Run
gcloud run deploy apx-mock-backend-dev \
  --image us-central1-docker.pkg.dev/apx-build-478003/apx-containers/mock-backend:latest \
  --region us-central1 \
  --project apx-build-478003 \
  --platform managed \
  --allow-unauthenticated \
  --memory 256Mi \
  --cpu 1 \
  --min-instances 1 \
  --max-instances 10
```

## Use with APX Router

Configure APX router to proxy to this backend:

```bash
# Set environment variable on router
ROUTES_CONFIG="/mock/**=https://apx-mock-backend-dev-xxx.run.app:sync"

# Test through APX
curl https://api.apx.build/mock/echo \
  -H "Authorization: Bearer apx_test_..."
```

## Environment Variables

- `PORT` - HTTP port (default: 8080)
- `SERVICE_NAME` - Service name for logging (default: apx-mock-backend)

## Similar to mocktarget.apigee.net

This service provides similar functionality to Apigee's mock target for testing API gateways, with these endpoints matching their patterns:
- Root landing page
- Echo/reflection endpoints
- JSON/XML responses
- Delay and status code testing
- Header inspection

