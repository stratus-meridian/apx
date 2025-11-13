# APX Portal Backend Connectors

**Version:** 1.0
**Last Updated:** 2025-11-12
**Status:** Production Ready

---

## Overview

The APX Portal backend connectors provide production-ready integration with Google Cloud Platform services and the APX Router. All connectors use real GCP SDK clients and include:

- BigQuery for analytics queries
- Firestore for configuration and data storage
- Cloud Storage for file management
- Pub/Sub for event streaming
- WebSocket Server for real-time updates
- Router Sync for API key synchronization

---

## Table of Contents

1. [Architecture](#architecture)
2. [BigQuery Connector](#bigquery-connector)
3. [Firestore Connector](#firestore-connector)
4. [Cloud Storage Connector](#cloud-storage-connector)
5. [Pub/Sub Connector](#pubsub-connector)
6. [WebSocket Server](#websocket-server)
7. [API Key Management](#api-key-management)
8. [Router Sync](#router-sync)
9. [Organization Management](#organization-management)
10. [Analytics Pipeline](#analytics-pipeline)
11. [Auth Adapter](#auth-adapter)
12. [Health Checks](#health-checks)
13. [Configuration](#configuration)
14. [Error Handling](#error-handling)
15. [Testing](#testing)
16. [Troubleshooting](#troubleshooting)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     APX Portal (Next.js)                    │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │           Backend Connectors Layer                  │   │
│  │                                                      │   │
│  │  BigQuery │ Firestore │ Storage │ Pub/Sub │ WS    │   │
│  └────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ Real GCP SDK Clients
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                  Google Cloud Platform                      │
│                                                             │
│  BigQuery  │  Firestore  │  Cloud Storage  │  Pub/Sub    │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ Router Sync API
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      APX Router (Go)                        │
└─────────────────────────────────────────────────────────────┘
```

---

## BigQuery Connector

**Location:** `.private/portal/lib/backend/bigquery.ts`

### Features

- Query execution with parameter binding
- Streaming insert for real-time data
- Connection pooling and retry logic
- Health checks and monitoring
- Query cost estimation

### Usage

#### Execute Query

```typescript
import * as bigquery from '@/lib/backend/bigquery';

const results = await bigquery.executeQuery({
  query: 'SELECT * FROM `dataset.table` WHERE user_id = @userId LIMIT 100',
  params: { userId: 'user_123' },
  maxResults: 100,
  timeoutMs: 30000,
});
```

#### Streaming Insert

```typescript
await bigquery.streamInsertSingle(
  {
    datasetId: 'apx_analytics',
    tableId: 'api_requests',
  },
  {
    request_id: 'req_123',
    timestamp: new Date().toISOString(),
    user_id: 'user_123',
    status_code: 200,
    latency_ms: 150,
  }
);
```

#### Health Check

```typescript
const health = await bigquery.checkHealth();
console.log(health.status); // 'healthy', 'degraded', or 'unhealthy'
```

### Configuration

```bash
# Environment Variables
GOOGLE_CLOUD_PROJECT=apx-build-478003
BIGQUERY_PROJECT_ID=apx-build-478003
BIGQUERY_LOCATION=US
BIGQUERY_DATASET_ID=apx_analytics
BIGQUERY_TABLE_ID=api_requests

# Credentials (choose one)
GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json
# OR
BIGQUERY_CLIENT_EMAIL=...
BIGQUERY_PRIVATE_KEY=...
```

---

## Firestore Connector

**Location:** `.private/portal/lib/backend/firestore.ts`

### Features

- CRUD operations with type safety
- Query building with filters and sorting
- Batch operations for atomic updates
- Real-time listeners
- Health checks

### Usage

#### Create Document

```typescript
import * as firestore from '@/lib/backend/firestore';

const doc = await firestore.createDocument('api_keys', {
  name: 'Production Key',
  user_id: 'user_123',
  scopes: ['read', 'write'],
  status: 'active',
});
```

#### Query Documents

```typescript
const keys = await firestore.queryDocuments('api_keys', {
  filters: [
    { field: 'user_id', operator: '==', value: 'user_123' },
    { field: 'status', operator: '==', value: 'active' },
  ],
  orderBy: [{ field: 'created_at', direction: 'desc' }],
  limit: 10,
});
```

#### Real-time Listener

```typescript
const unsubscribe = firestore.subscribeToDocument(
  'api_keys',
  'key_123',
  (data) => {
    console.log('Key updated:', data);
  },
  (error) => {
    console.error('Listener error:', error);
  }
);

// Cleanup
unsubscribe();
```

#### Batch Operations

```typescript
await firestore.executeBatch([
  {
    type: 'set',
    collection: 'api_keys',
    docId: 'key_1',
    data: { status: 'active' },
  },
  {
    type: 'update',
    collection: 'api_keys',
    docId: 'key_2',
    data: { last_used_at: new Date() },
  },
  {
    type: 'delete',
    collection: 'api_keys',
    docId: 'key_3',
  },
]);
```

### Configuration

```bash
NEXT_PUBLIC_FIREBASE_PROJECT_ID=apx-build-478003
FIREBASE_CLIENT_EMAIL=...
FIREBASE_PRIVATE_KEY=...
```

---

## Cloud Storage Connector

**Location:** `.private/portal/lib/backend/storage.ts`

### Features

- File upload/download operations
- Signed URLs for temporary access
- Bucket management
- Metadata operations

### Usage

#### Upload File

```typescript
import * as storage from '@/lib/backend/storage';

const metadata = await storage.uploadFile(
  fileBuffer,
  {
    destination: 'policies/rate-limiting-v1.wasm',
    contentType: 'application/wasm',
    public: false,
  },
  'apx-prod-artifacts'
);
```

#### Generate Signed URL

```typescript
const url = await storage.generateSignedUrl(
  'policies/rate-limiting-v1.wasm',
  {
    action: 'read',
    expires: Date.now() + 3600 * 1000, // 1 hour
  },
  'apx-prod-artifacts'
);
```

#### Download File

```typescript
const buffer = await storage.downloadFile('policies/rate-limiting-v1.wasm', 'apx-prod-artifacts');
```

### Configuration

```bash
GOOGLE_CLOUD_PROJECT=apx-build-478003
GCS_BUCKET=apx-uploads
GCS_CLIENT_EMAIL=...
GCS_PRIVATE_KEY=...
```

---

## Pub/Sub Connector

**Location:** `.private/portal/lib/backend/pubsub.ts`

### Features

- Publish messages to topics
- Subscribe to topics with message handlers
- Ordered message delivery
- Batch publishing

### Usage

#### Publish Message

```typescript
import * as pubsub from '@/lib/backend/pubsub';

const messageId = await pubsub.publishJSON('api-key-updates', {
  action: 'created',
  key_id: 'key_123',
  user_id: 'user_123',
});
```

#### Subscribe to Topic

```typescript
const unsubscribe = await pubsub.subscribe(
  'portal-updates-sub',
  async (message) => {
    const data = pubsub.parseMessageData(message);
    console.log('Received:', data);
  },
  (error) => {
    console.error('Subscription error:', error);
  }
);

// Cleanup
unsubscribe();
```

### Configuration

```bash
GOOGLE_CLOUD_PROJECT=apx-build-478003
PUBSUB_CLIENT_EMAIL=...
PUBSUB_PRIVATE_KEY=...
```

---

## WebSocket Server

**Location:** `.private/websocket-server/`

### Features

- Real-time connection management
- JWT authentication
- Pub/Sub integration for broadcasting
- Room/channel subscriptions

### Deployment

```bash
# Build Docker image
cd .private/websocket-server
docker build -t apx-websocket-server .

# Run locally
npm install
npm run dev

# Deploy to Cloud Run
gcloud run deploy apx-websocket-server \
  --image gcr.io/apx-build-478003/apx-websocket-server \
  --port 3001 \
  --set-env-vars NEXTAUTH_SECRET=...,GOOGLE_CLOUD_PROJECT=apx-build-478003
```

### Client Usage

```typescript
import { useEffect, useState } from 'react';

function useWebSocket(token: string) {
  const [ws, setWs] = useState<WebSocket | null>(null);

  useEffect(() => {
    const socket = new WebSocket(`ws://localhost:3001?token=${token}`);

    socket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      console.log('Received:', message);
    };

    socket.onopen = () => {
      // Subscribe to channels
      socket.send(JSON.stringify({ type: 'subscribe', channel: 'api-keys' }));
    };

    setWs(socket);

    return () => {
      socket.close();
    };
  }, [token]);

  return ws;
}
```

### Configuration

```bash
WS_PORT=3001
NEXTAUTH_SECRET=your-secret-key
GOOGLE_CLOUD_PROJECT=apx-build-478003
PUBSUB_SUBSCRIPTION=portal-updates-sub
```

---

## API Key Management

**Location:** `.private/portal/lib/backend/api-keys.ts`

### Features

- Store keys in Firestore
- Sync with APX Router
- Real-time key status updates
- Usage tracking

### Usage

#### Create API Key

```typescript
import * as apiKeys from '@/lib/backend/api-keys';

const key = await apiKeys.createAPIKey({
  user_id: 'user_123',
  org_id: 'org_456',
  name: 'Production Key',
  scopes: ['read', 'write'],
  rate_limit: {
    requests_per_second: 10,
    requests_per_minute: 600,
    requests_per_hour: 36000,
    requests_per_day: 864000,
  },
  expires_in_days: 365,
});

// Save the key value - this is the only time it's visible
console.log('API Key:', key.key_value);
```

#### Revoke API Key

```typescript
await apiKeys.revokeAPIKey('key_123', 'user_123');
// Automatically syncs to router
```

---

## Router Sync

**Location:** `.private/portal/lib/backend/router-sync.ts`

### Features

- HTTP client for router admin API
- Automatic retries for failed syncs
- Health check monitoring
- Sync status tracking

### Usage

#### Sync Key to Router

```typescript
import { syncKeyToRouter } from '@/lib/backend/router-sync';

const result = await syncKeyToRouter({
  key_id: 'key_123',
  key_hash: 'sha256_hash...',
  scopes: ['read', 'write'],
  status: 'active',
  rate_limit: {
    requests_per_second: 10,
  },
  user_id: 'user_123',
});

if (result.success) {
  console.log('Synced at:', result.synced_at);
} else {
  console.error('Sync failed:', result.message);
}
```

### Configuration

```bash
NEXT_PUBLIC_APX_ROUTER_URL=https://router.apx.dev
APX_INTERNAL_API_KEY=router-admin-key
```

---

## Health Checks

**Endpoint:** `GET /api/health`

### Response Format

```json
{
  "status": "healthy",
  "timestamp": "2025-11-12T10:00:00Z",
  "version": "1.0.0",
  "uptime": 3600,
  "checks": {
    "bigquery": {
      "status": "ok",
      "latency": 250,
      "metadata": {
        "datasetId": "apx_analytics",
        "location": "US"
      }
    },
    "firestore": {
      "status": "ok",
      "latency": 150
    },
    "storage": {
      "status": "ok",
      "latency": 180
    },
    "pubsub": {
      "status": "ok",
      "latency": 120
    },
    "websocket": {
      "status": "ok"
    },
    "router": {
      "status": "ok",
      "latency": 100,
      "metadata": {
        "version": "1.0.0"
      }
    }
  }
}
```

### Status Codes

- `200` - Healthy or degraded (service operational)
- `503` - Unhealthy (service should not receive traffic)

---

## Configuration

### Complete Environment Variables

```bash
# ============================================================================
# Google Cloud Platform
# ============================================================================
GOOGLE_CLOUD_PROJECT=apx-build-478003
GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json

# BigQuery
BIGQUERY_PROJECT_ID=apx-build-478003
BIGQUERY_LOCATION=US
BIGQUERY_DATASET_ID=apx_analytics
BIGQUERY_TABLE_ID=api_requests

# Firestore
NEXT_PUBLIC_FIREBASE_PROJECT_ID=apx-build-478003
FIREBASE_CLIENT_EMAIL=firebase-adminsdk-xxxxx@project.iam.gserviceaccount.com
FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"

# Cloud Storage
GCS_BUCKET=apx-uploads
GCS_CLIENT_EMAIL=...
GCS_PRIVATE_KEY=...

# Pub/Sub
PUBSUB_CLIENT_EMAIL=...
PUBSUB_PRIVATE_KEY=...

# ============================================================================
# APX Router
# ============================================================================
NEXT_PUBLIC_APX_ROUTER_URL=https://router.apx.dev
APX_INTERNAL_API_KEY=router-admin-key

# ============================================================================
# WebSocket Server
# ============================================================================
WS_PORT=3001
NEXTAUTH_SECRET=your-secret-key
PUBSUB_SUBSCRIPTION=portal-updates-sub

# ============================================================================
# NextAuth
# ============================================================================
NEXTAUTH_URL=https://portal.apx.dev
NEXTAUTH_SECRET=your-nextauth-secret
```

---

## Error Handling

All connectors use structured error types:

```typescript
import { BackendError, ValidationError, TimeoutError } from '@/lib/backend/types';

try {
  await bigquery.executeQuery({ query: '...' });
} catch (error) {
  if (error instanceof TimeoutError) {
    console.error('Query timed out:', error.message);
  } else if (error instanceof ValidationError) {
    console.error('Validation failed:', error.details);
  } else if (error instanceof BackendError) {
    console.error('Backend error:', error.service, error.code);
  }
}
```

---

## Testing

### Integration Tests

```bash
# Run all integration tests
npm test -- __tests__/integration/backend

# Run specific connector tests
npm test -- __tests__/integration/backend/bigquery.test.ts
npm test -- __tests__/integration/backend/firestore.test.ts
npm test -- __tests__/integration/backend/api-keys.test.ts
```

### Test Configuration

```bash
# Use emulators for local testing
BIGQUERY_PROJECT_ID=test-project
FIRESTORE_EMULATOR_HOST=localhost:8080
PUBSUB_EMULATOR_HOST=localhost:8085
```

---

## Troubleshooting

### Common Issues

#### BigQuery: "Query execution timed out"

**Solution:**
- Increase timeout: `{ timeoutMs: 60000 }`
- Optimize query with proper indexes
- Reduce data scanned with WHERE clauses

#### Firestore: "Client not configured"

**Solution:**
- Verify environment variables are set
- Check credentials are not placeholder values
- Ensure Firebase Admin is initialized

#### Router Sync: "Request failed"

**Solution:**
- Check `NEXT_PUBLIC_APX_ROUTER_URL` is correct
- Verify `APX_INTERNAL_API_KEY` is valid
- Ensure router is reachable from portal

#### WebSocket: "Connection refused"

**Solution:**
- Verify WebSocket server is running
- Check JWT token is valid
- Ensure firewall allows port 3001

### Debug Logging

Enable debug logging for connectors:

```typescript
import { logger } from '@/lib/logger';

logger.setLevel('debug');
```

### Health Check Debugging

```bash
# Check individual services
curl https://portal.apx.dev/api/health

# Check specific connector
curl https://portal.apx.dev/api/health | jq '.checks.bigquery'
```

---

## Best Practices

1. **Always use health checks** before critical operations
2. **Implement retry logic** for transient failures
3. **Use batch operations** when updating multiple documents
4. **Enable query caching** in BigQuery for repeated queries
5. **Monitor connector metrics** in Cloud Monitoring
6. **Use signed URLs** for temporary file access
7. **Sync API keys immediately** after creation/revocation
8. **Subscribe to Pub/Sub topics** for real-time updates
9. **Clean up WebSocket connections** on component unmount
10. **Test with emulators** before deploying to production

---

## Support

For issues or questions:

1. Check [Troubleshooting](#troubleshooting) section
2. Review integration tests for usage examples
3. Check GCP service status: https://status.cloud.google.com/
4. Review APX Router logs for sync issues

---

**Document Version:** 1.0
**Last Updated:** 2025-11-12
**Maintained By:** APX Team
