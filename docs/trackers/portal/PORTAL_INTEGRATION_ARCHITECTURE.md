# APX Portal - Backend Integration Architecture

**Version:** 1.0
**Last Updated:** 2025-11-11

---

## Overview

The APX Developer Portal is **tightly integrated** with all APX backend services, creating a seamless experience where portal actions immediately affect the API platform and vice versa.

---

## Integration Map

```
┌─────────────────────────────────────────────────────────────────┐
│                     APX DEVELOPER PORTAL                        │
│                    (Next.js 14 App Router)                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Dashboard  │  │  API Console │  │   API Keys   │         │
│  │              │  │   "Try It"   │  │     CRUD     │         │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │
│         │                  │                  │                 │
│         │                  │                  │                 │
│  ┌──────▼──────────────────▼──────────────────▼───────┐        │
│  │           Next.js API Routes (Backend)             │        │
│  │  /api/dashboard/stats  /api/proxy  /api/keys      │        │
│  └──────┬──────────────────┬──────────────────┬───────┘        │
│         │                  │                  │                 │
└─────────┼──────────────────┼──────────────────┼─────────────────┘
          │                  │                  │
          │                  │                  │
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      APX BACKEND SERVICES                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐│
│  │  APX Router     │  │  APX Edge       │  │  BigQuery       ││
│  │  (Cloud Run)    │  │  (Cloud Run)    │  │  (Analytics)    ││
│  │                 │  │                 │  │                 ││
│  │ • Health Check  │  │ • Request Trace │  │ • Usage Data    ││
│  │ • API Proxy     │  │ • Latency Logs  │  │ • Request Logs  ││
│  │ • Policy Apply  │  │ • Error Logs    │  │ • Cost Data     ││
│  └─────────────────┘  └─────────────────┘  └─────────────────┘│
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐│
│  │  Firestore      │  │  Pub/Sub        │  │  Secret Mgr     ││
│  │  (Database)     │  │  (Events)       │  │  (Secrets)      ││
│  │                 │  │                 │  │                 ││
│  │ • API Keys      │  │ • Webhooks      │  │ • Credentials   ││
│  │ • Users/Orgs    │  │ • Real-time     │  │ • API Keys      ││
│  │ • Policies      │  │   Updates       │  │ • Tokens        ││
│  └─────────────────┘  └─────────────────┘  └─────────────────┘│
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Data Flow Diagrams

### 1. API Key Creation Flow

```
User                Portal UI           API Route          Firestore          APX Router
  │                     │                   │                   │                   │
  │  Click "Create"     │                   │                   │                   │
  ├────────────────────>│                   │                   │                   │
  │                     │                   │                   │                   │
  │                     │  POST /api/keys   │                   │                   │
  │                     ├──────────────────>│                   │                   │
  │                     │                   │                   │                   │
  │                     │                   │  Generate apx_... │                   │
  │                     │                   ├──────────────────>│                   │
  │                     │                   │                   │                   │
  │                     │                   │      Save key     │                   │
  │                     │                   │<──────────────────┤                   │
  │                     │                   │                   │                   │
  │                     │    Return key     │                   │                   │
  │                     │<──────────────────┤                   │                   │
  │                     │                   │                   │                   │
  │  Display key (once) │                   │                   │                   │
  │<────────────────────┤                   │                   │                   │
  │                     │                   │                   │                   │
  │                     │                   │                   │                   │
  │  Use key in request │                   │                   │                   │
  ├─────────────────────────────────────────────────────────────────────────────────>│
  │                     │                   │                   │                   │
  │                     │                   │                   │  Validate key     │
  │                     │                   │                   │<──────────────────┤
  │                     │                   │                   │                   │
  │                     │                   │                   │  Return valid     │
  │                     │                   │                   │──────────────────>│
  │                     │                   │                   │                   │
  │                     │                   │                   │                   │
  │<─────────────────────────────────────────────────────────── API Response ───────┤
  │                     │                   │                   │                   │
```

**Key Point:** Portal creates key → Firestore → Router validates **immediately** (no sync delay)

---

### 2. Dashboard Stats Flow

```
User             Portal UI         API Route        BigQuery       Firestore
  │                  │                  │                │              │
  │  Visit /dashboard│                  │                │              │
  ├─────────────────>│                  │                │              │
  │                  │                  │                │              │
  │                  │  Load stats      │                │              │
  │                  ├─────────────────>│                │              │
  │                  │                  │                │              │
  │                  │                  │  Get user keys │              │
  │                  │                  ├────────────────┼─────────────>│
  │                  │                  │                │              │
  │                  │                  │  Return keys   │              │
  │                  │                  │<───────────────┼──────────────┤
  │                  │                  │                │              │
  │                  │                  │  Query requests│              │
  │                  │                  ├───────────────>│              │
  │                  │                  │  WHERE key IN  │              │
  │                  │                  │  (user_keys)   │              │
  │                  │                  │                │              │
  │                  │                  │  Aggregates:   │              │
  │                  │                  │  - count       │              │
  │                  │                  │  - p95 latency │              │
  │                  │                  │  - error_rate  │              │
  │                  │                  │<───────────────┤              │
  │                  │                  │                │              │
  │                  │  Return stats    │                │              │
  │                  │<─────────────────┤                │              │
  │                  │                  │                │              │
  │  Render charts   │                  │                │              │
  │<─────────────────┤                  │                │              │
  │                  │                  │                │              │
```

**Key Point:** Portal queries BigQuery → scoped to user's keys → real-time usage data

---

### 3. API Console "Try It" Flow

```
User           Portal UI       Proxy API         APX Router      BigQuery
  │                │                │                 │               │
  │  Click "Send"  │                │                 │               │
  ├───────────────>│                │                 │               │
  │                │                │                 │               │
  │                │  POST /api/proxy                 │               │
  │                ├───────────────>│                 │               │
  │                │                │                 │               │
  │                │                │  Generate UUID  │               │
  │                │                │  (request_id)   │               │
  │                │                │                 │               │
  │                │                │  Forward with:  │               │
  │                │                │  x-apx-api-key  │               │
  │                │                │  x-apx-request-id              │
  │                │                ├────────────────>│               │
  │                │                │                 │               │
  │                │                │                 │  Log request  │
  │                │                │                 ├──────────────>│
  │                │                │                 │               │
  │                │                │  Return response│               │
  │                │                │<────────────────┤               │
  │                │                │                 │               │
  │                │  Return response                 │               │
  │                │<───────────────┤                 │               │
  │                │  + request_id  │                 │               │
  │                │                │                 │               │
  │  Display:      │                │                 │               │
  │  - Status      │                │                 │               │
  │  - Body        │                │                 │               │
  │  - Latency     │                │                 │               │
  │  - Request ID  │                │                 │               │
  │<───────────────┤                │                 │               │
  │                │                │                 │               │
  │  Click "Trace" │                │                 │               │
  ├───────────────>│                │                 │               │
  │                │                │                 │               │
  │                │  Query BigQuery for request_id   │               │
  │                ├──────────────────────────────────┼──────────────>│
  │                │                │                 │               │
  │                │  Return trace details            │               │
  │                │<─────────────────────────────────┼───────────────┤
  │                │  (policy, route, quotas, etc.)   │               │
  │                │                │                 │               │
  │  Show trace    │                │                 │               │
  │<───────────────┤                │                 │               │
  │                │                │                 │               │
```

**Key Point:** Request ID propagates → enables trace lookup → explainability

---

## Integration Points Reference

### APX Router

**Base URL:** `process.env.NEXT_PUBLIC_APX_ROUTER_URL`

| Endpoint | Method | Purpose | Headers | Response |
|----------|--------|---------|---------|----------|
| `/health` | GET | System status | `x-apx-internal-key` | `{status, version, components}` |
| `/*` (proxy) | ANY | User API calls | `x-apx-api-key`, `x-apx-request-id` | Proxied response |

**Portal Usage:**
- `PM0-T2-001`: Health check (system status badge)
- `PM1-T1-003`: API console (Try It)
- All user API requests

---

### APX Edge

**Base URL:** `process.env.NEXT_PUBLIC_APX_EDGE_URL`

| Endpoint | Method | Purpose | Response |
|----------|--------|---------|----------|
| `/tail` | GET SSE | Real-time request stream | Server-sent events |
| `/health` | GET | Edge status | `{status, region}` |

**Portal Usage:**
- `PM2-T2-001`: Real-time request viewer (dashboard widget)
- `PM2-T2-002`: Live tail for debugging

---

### BigQuery

**Dataset:** `apx_requests`
**Tables:**
- `requests`: All API requests (logged by Edge)
- `api_keys`: Key metadata (synced from Firestore)
- `usage_rollups`: Pre-aggregated usage (hourly/daily)

**Portal Queries:**

```sql
-- Dashboard stats (PM1-T1-001)
SELECT
  COUNT(*) as requests_24h,
  APPROX_QUANTILES(latency_ms, 100)[OFFSET(95)] AS p95_latency,
  COUNTIF(status_code >= 400) * 100.0 / COUNT(*) AS error_rate
FROM `apx_requests.requests`
WHERE api_key IN (SELECT key_id FROM api_keys WHERE user_id = @userId)
  AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)

-- Request explorer (PM2-T1-002)
SELECT
  request_id,
  timestamp,
  method,
  path,
  status_code,
  latency_ms,
  policy_version,
  tenant
FROM `apx_requests.requests`
WHERE api_key = @apiKey
  AND timestamp BETWEEN @start_date AND @end_date
ORDER BY timestamp DESC
LIMIT 100

-- Usage chart (PM2-T1-001)
SELECT
  TIMESTAMP_TRUNC(timestamp, HOUR) as hour,
  COUNT(*) as requests,
  AVG(latency_ms) as avg_latency_ms,
  APPROX_QUANTILES(latency_ms, 100)[OFFSET(95)] as p95_latency_ms
FROM `apx_requests.requests`
WHERE api_key = @apiKey
  AND timestamp >= @start_date
GROUP BY hour
ORDER BY hour
```

**Portal Usage:**
- `PM1-T1-001`: Dashboard stats
- `PM2-T1-001`: Usage charts (requests, latency, errors over time)
- `PM2-T1-002`: Request explorer (search by ID, filters)
- `PM3-T2-001`: Cost analytics (billing calculations)

---

### Firestore

**Collections:**

```
firestore/
├── users/
│   └── {user_id}/
│       ├── email: string
│       ├── name: string
│       └── created_at: timestamp
│
├── orgs/
│   └── {org_id}/
│       ├── name: string
│       ├── plan: string
│       └── members: array<user_id>
│
├── api_keys/
│   └── {key_id}/  # apx_...
│       ├── user_id: string
│       ├── org_id: string
│       ├── name: string
│       ├── scopes: array<string>
│       ├── rate_limit: number
│       ├── ip_allowlist: array<string>
│       ├── status: enum(active, revoked)
│       ├── created_at: timestamp
│       └── last_used_at: timestamp
│
├── policies/
│   └── {policy_id}/
│       ├── org_id: string
│       ├── version: string
│       ├── bundle: object (PolicyBundle CRD)
│       ├── status: enum(draft, active, archived)
│       └── updated_at: timestamp
│
├── webhooks/
│   └── {webhook_id}/
│       ├── org_id: string
│       ├── endpoint: string
│       ├── events: array<string>
│       ├── secret: string
│       ├── status: enum(active, paused)
│       └── deliveries: subcollection
│
└── sessions/
    └── {session_id}/
        ├── user_id: string
        ├── expires_at: timestamp
        └── ...
```

**Portal Operations:**

```typescript
// Create API key (PM1-T2-001)
await db.collection('api_keys').doc(keyId).set({
  id: keyId,
  user_id: userId,
  name: 'My API Key',
  scopes: ['product:payments'],
  status: 'active',
  created_at: new Date().toISOString(),
})

// List user's keys (PM1-T2-001)
const snapshot = await db
  .collection('api_keys')
  .where('user_id', '==', userId)
  .where('status', '==', 'active')
  .get()

// Revoke key (PM1-T2-001)
await db.collection('api_keys').doc(keyId).update({
  status: 'revoked',
})

// Get org members (PM1-T2-002)
const orgDoc = await db.collection('orgs').doc(orgId).get()
const members = orgDoc.data().members
```

**Portal Usage:**
- `PM0-T2-002`: Auth sessions (NextAuth adapter)
- `PM1-T2-001`: API keys CRUD
- `PM1-T2-002`: Organizations, team management
- `PM3-T1-001`: Webhooks configuration
- `PM4-T2-001`: Policy management

---

### Pub/Sub

**Topics:**

```
apx-events/
├── api-requests     # Every API request (for webhooks)
├── api-key-events   # Key created/revoked
├── policy-changes   # PolicyBundle updates
└── usage-alerts     # Quota warnings, SLO violations
```

**Portal Subscriptions:**

```typescript
// Real-time dashboard updates (PM2-T2-003)
const subscription = pubsub
  .topic('api-requests')
  .subscription(`portal-dashboard-${userId}`)

subscription.on('message', (message) => {
  const request = JSON.parse(message.data)
  // Update dashboard live
})

// Webhook delivery logs (PM3-T1-002)
const deliveryLogs = pubsub
  .topic('webhook-deliveries')
  .subscription('portal-webhooks')
```

**Portal Usage:**
- `PM2-T2-003`: Real-time dashboard updates (SSE to browser)
- `PM3-T1-002`: Webhook delivery logs, retries
- `PM4-T3-001`: Usage alerts, notifications

---

## Environment Variables

```bash
# APX Backend Services
NEXT_PUBLIC_APX_ROUTER_URL=https://router-abc123.run.app
NEXT_PUBLIC_APX_EDGE_URL=https://edge-abc123.run.app
APX_INTERNAL_API_KEY=internal-key-for-portal

# GCP Project (same as APX backend)
GCP_PROJECT_ID=apx-dev-abc123

# BigQuery
BIGQUERY_DATASET=apx_requests
BIGQUERY_TABLE=requests

# Firestore (same as APX backend)
FIREBASE_PROJECT_ID=apx-dev-abc123
FIREBASE_CLIENT_EMAIL=portal@apx-dev.iam.gserviceaccount.com
FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"

# Auth
NEXTAUTH_URL=https://portal.apx.dev
NEXTAUTH_SECRET=generated-secret-here
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...

# Stripe (for billing, M3)
STRIPE_SECRET_KEY=sk_test_...
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...
```

---

## Data Consistency Guarantees

### 1. API Key Creation

**Guarantee:** Key is **immediately** usable after creation

**Flow:**
1. Portal creates key in Firestore (`status: active`)
2. Firestore write succeeds (strong consistency)
3. APX Router reads from same Firestore instance
4. No replication lag (single region deployment)

**Test:**
```bash
# Create key via portal UI
# Copy key ID: apx_abc123...

# Immediately use in request (<1 second later)
curl https://router.run.app/v1/example \
  -H "x-apx-api-key: apx_abc123..."
# ✅ Returns 200 OK (not 401)
```

### 2. Key Revocation

**Guarantee:** Revoked key is **immediately** rejected

**Flow:**
1. Portal updates key: `status: revoked`
2. APX Router checks Firestore on every request
3. Next request with revoked key → 401 Unauthorized

**Test:**
```bash
# Revoke key in portal
# Wait 0 seconds

# Try request
curl https://router.run.app/v1/example \
  -H "x-apx-api-key: apx_abc123..."
# ✅ Returns 401 Unauthorized
```

### 3. Usage Data

**Guarantee:** Eventually consistent (30-60 second lag)

**Flow:**
1. User makes API request
2. APX Edge logs to BigQuery (streaming insert)
3. BigQuery makes data available (30-60s lag)
4. Portal queries BigQuery → shows recent data

**Acceptable:** Dashboard shows stats with ~1 minute lag

### 4. Request Tracing

**Guarantee:** Request ID is globally unique and traceable

**Flow:**
1. Portal generates UUID (request_id)
2. Passes to APX Router via `x-apx-request-id` header
3. Router propagates to Edge, workers, logs
4. BigQuery logs include request_id
5. Portal queries BigQuery by request_id → full trace

**Test:**
```bash
# Make request with ID
request_id=$(uuidgen)
curl https://router.run.app/v1/example \
  -H "x-apx-request-id: $request_id"

# Wait 1 minute

# Query BigQuery
bq query "SELECT * FROM apx_requests.requests WHERE request_id = '$request_id'"
# ✅ Returns full trace
```

---

## Error Handling

### Backend Unavailable

```typescript
// lib/apx-client.ts
export async function getRouterHealth(): Promise<Health> {
  try {
    const res = await fetch(`${APX_ROUTER_URL}/health`, {
      headers: { 'x-apx-internal-key': APX_INTERNAL_API_KEY },
      cache: 'no-store',
      signal: AbortSignal.timeout(5000), // 5s timeout
    })

    if (!res.ok) {
      throw new Error(`Router returned ${res.status}`)
    }

    return HealthSchema.parse(await res.json())
  } catch (error) {
    console.error('Router health check failed:', error)

    // Return degraded status (don't crash portal)
    return {
      status: 'down',
      version: 'unknown',
      timestamp: new Date().toISOString(),
      components: {
        firestore: 'down',
        pubsub: 'down',
        bigquery: 'down',
      },
    }
  }
}
```

### BigQuery Timeouts

```typescript
// lib/bigquery/usage.ts
export async function getUsageStats(userId: string) {
  try {
    const [rows] = await bigquery.query({
      query: USAGE_QUERY,
      params: { userId },
      timeoutMs: 10000, // 10s max
    })

    return rows
  } catch (error) {
    if (error.code === 'DEADLINE_EXCEEDED') {
      // Return cached data or empty state
      console.warn('BigQuery timeout, using cached data')
      return getCachedUsageStats(userId)
    }

    throw error
  }
}
```

### Firestore Quota Exceeded

```typescript
// lib/firestore/api-keys.ts
export async function createAPIKey(userId: string, data: CreateKeyData) {
  try {
    const keyId = generateKeyId()
    await db.collection('api_keys').doc(keyId).set({
      id: keyId,
      user_id: userId,
      ...data,
    })

    return keyId
  } catch (error) {
    if (error.code === 'RESOURCE_EXHAUSTED') {
      throw new Error('Rate limit exceeded. Please try again in 1 minute.')
    }

    throw error
  }
}
```

---

## Performance Optimizations

### 1. BigQuery Caching

```typescript
// Cache stats for 30 seconds
export const revalidate = 30

export async function GET() {
  // Next.js automatically caches this for 30s
  const stats = await getDashboardStats(userId)
  return NextResponse.json(stats)
}
```

### 2. Firestore Indexes

```yaml
# firestore.indexes.json
indexes:
  - collectionGroup: api_keys
    queryScope: COLLECTION
    fields:
      - fieldPath: user_id
        order: ASCENDING
      - fieldPath: status
        order: ASCENDING
      - fieldPath: created_at
        order: DESCENDING
```

### 3. Connection Pooling

```typescript
// Reuse BigQuery client (singleton)
let bigqueryClient: BigQuery | null = null

export function getBigQuery(): BigQuery {
  if (!bigqueryClient) {
    bigqueryClient = new BigQuery({
      projectId: process.env.GCP_PROJECT_ID,
      // ... credentials
    })
  }

  return bigqueryClient
}
```

---

## Security

### 1. API Key Scoping

**Portal enforces:**
- Users can only access their own keys
- Keys are scoped to products/orgs
- Revoked keys immediately rejected

**Implementation:**
```typescript
// app/api/keys/route.ts
const session = await getServerSession(authOptions)

// ALWAYS filter by session.user.id
const keys = await db
  .collection('api_keys')
  .where('user_id', '==', session.user.id) // ✅ User isolation
  .get()
```

### 2. BigQuery Row-Level Security

**Query always filters by user:**
```sql
-- Dashboard stats
SELECT COUNT(*) as requests
FROM `apx_requests.requests`
WHERE api_key IN (
  SELECT key_id
  FROM `apx_requests.api_keys`
  WHERE user_id = @userId  -- ✅ Only user's data
)
```

### 3. Firestore Security Rules

```javascript
// firestore.rules
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    // API keys: users can only access their own
    match /api_keys/{keyId} {
      allow read, write: if request.auth != null
                          && resource.data.user_id == request.auth.uid;
    }

    // Orgs: members can read, owners can write
    match /orgs/{orgId} {
      allow read: if request.auth.uid in resource.data.members;
      allow write: if request.auth.uid == resource.data.owner;
    }
  }
}
```

---

## Monitoring Integration Points

### Portal → APX Backend

```typescript
// Track portal-initiated requests
fetch(`${APX_ROUTER_URL}/v1/example`, {
  headers: {
    'x-apx-api-key': apiKey,
    'x-apx-request-id': requestId,
    'x-apx-source': 'portal-console', // ✅ Identify source
  },
})
```

### BigQuery Logs

```sql
-- Portal request volume
SELECT
  EXTRACT(HOUR FROM timestamp) as hour,
  COUNT(*) as portal_requests
FROM `apx_requests.requests`
WHERE headers.x_apx_source = 'portal-console'
GROUP BY hour
```

### Alerts

```yaml
# Alert if portal → router error rate >5%
alert: portal_error_rate_high
expr: |
  (
    sum(rate(apx_requests_total{source="portal-console", status=~"5.."}[5m]))
    /
    sum(rate(apx_requests_total{source="portal-console"}[5m]))
  ) > 0.05
for: 5m
```

---

## Summary

**The APX Portal is deeply integrated with the backend:**

✅ **API Keys:** Portal creates → Firestore → Router validates (instant)
✅ **Usage Data:** Edge logs → BigQuery → Portal displays (<1 min lag)
✅ **Request Tracing:** Request ID → full trace in BigQuery
✅ **Real-time Updates:** Pub/Sub → SSE → live dashboard
✅ **Same Database:** Portal and Router share Firestore (no sync)
✅ **Same GCP Project:** Unified billing, IAM, monitoring

**This tight integration ensures:**
- Zero data sync issues
- Instant key activation/revocation
- Complete observability
- Unified user experience

---

**Version:** 1.0
**Last Updated:** 2025-11-11
**Maintained by:** Portal Team
