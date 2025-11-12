# Portal AI Agent Instructions

**Version:** 1.0
**Last Updated:** 2025-11-11
**For:** AI agents implementing the APX Developer Portal

---

## Welcome, Agent!

You are about to build the **APX Developer Portal** - a world-class developer experience for API management. This document will guide you through:

1. How to claim and execute tasks
2. Quality standards for frontend and backend work
3. How to integrate with APX backend services
4. Testing and verification procedures
5. Communication protocols

---

## Quick Start (5 Minutes)

### 1. Read These Documents (in order)

1. **This document** (PORTAL_AI_AGENT_INSTRUCTIONS.md) - How to work
2. **[PORTAL_AGENT_EXECUTION_PLAN.md](PORTAL_AGENT_EXECUTION_PLAN.md)** - What to build
3. **[PORTAL_TASK_TRACKER.yaml](../../PORTAL_TASK_TRACKER.yaml)** - Current status

### 2. Set Up Your Environment

```bash
# Clone repo
cd /Users/agentsy/APILEE

# Check backend is running
curl https://router-abc123.run.app/health
# Should return: {"status":"healthy",...}

# Navigate to portal
cd portal  # (will exist after PM0-T1-001)

# Install dependencies
npm install

# Start dev server
npm run dev
# Visit: http://localhost:3000
```

### 3. Claim Your First Task

```bash
# Open PORTAL_TASK_TRACKER.yaml
# Find first available task for your agent type
# Example: PM0-T1-001 (frontend agent)

# Update status:
PM0-T1-001:
  status: IN_PROGRESS
  assigned_to: "agent-frontend-1"  # Your agent ID
  started_at: "2025-11-12T10:00:00Z"

# Commit and push:
git add PORTAL_TASK_TRACKER.yaml
git commit -m "[PM0-T1-001] Claiming Next.js initialization"
git push
```

---

## Task Execution Workflow

### Step 1: Select Task

**Rules:**
- Only select tasks where ALL dependencies are `COMPLETE`
- Match your agent type (frontend, backend, integration, docs, testing)
- Prioritize by: P0 > P1 > P2 > P3
- Only ONE task `IN_PROGRESS` per agent at a time

**Query:**
```yaml
# Find your next task:
SELECT task
FROM PORTAL_TASK_TRACKER.yaml
WHERE status = 'NOT_STARTED'
  AND agent_type = 'frontend'  # Or your type
  AND ALL dependencies.status = 'COMPLETE'
ORDER BY priority ASC, estimated_hours ASC
LIMIT 1
```

### Step 2: Claim Task

**Update PORTAL_TASK_TRACKER.yaml:**

```yaml
PM1-T1-001:
  status: IN_PROGRESS
  assigned_to: "agent-frontend-1"
  started_at: "2025-11-12T10:00:00Z"
  notes:
    - "2025-11-12T10:00:00Z: Started dashboard implementation"
```

**Commit:**
```bash
git add PORTAL_TASK_TRACKER.yaml
git commit -m "[PM1-T1-001] Claiming Dashboard task"
git push
```

### Step 3: Execute Task

**Read task definition** in PORTAL_AGENT_EXECUTION_PLAN.md

**Follow steps exactly:**

1. **Backend Verification** (if applicable)
   ```bash
   # Verify APX backend is reachable
   curl https://router-abc123.run.app/health
   # Must return 200 OK
   ```

2. **Execute each step**
   ```bash
   # Example: Install dependencies
   npm install @google-cloud/bigquery

   # Example: Create file
   touch lib/bigquery.ts
   # ... (implement as specified in task)
   ```

3. **Test as you go**
   ```bash
   # Type check
   npm run type-check

   # Run tests
   npm run test

   # Dev server
   npm run dev
   # Manual verification in browser
   ```

4. **Document progress**
   ```yaml
   # Update PORTAL_TASK_TRACKER.yaml notes:
   notes:
     - "2025-11-12T10:00:00Z: Started"
     - "2025-11-12T11:30:00Z: BigQuery client implemented"
     - "2025-11-12T13:00:00Z: Dashboard UI complete"
     - "2025-11-12T14:00:00Z: Tests passing"
   ```

### Step 4: Verification (Critical!)

**Check EVERY acceptance criterion:**

```yaml
# Example from PM1-T1-001:
acceptance_criteria:
  - text: "Dashboard loads stats from BigQuery"
    checked: true  # ✅ Verified: Stats displayed in UI
  - text: "Page loads in <2s"
    checked: true  # ✅ Verified: Lighthouse score 94
  - text: "Responsive on mobile/tablet/desktop"
    checked: true  # ✅ Verified: Tested 375px, 768px, 1440px
  - text: "Data scoped to user's API keys"
    checked: true  # ✅ Verified: BigQuery query filters by user_id
```

**Run quality gates:**

```bash
# 1. Type safety
npm run type-check
# Must: Zero TypeScript errors

# 2. Tests
npm run test
npm run test:e2e  # If applicable
# Must: All tests pass

# 3. Accessibility
npm run test:a11y
# Must: Zero Axe violations

# 4. Performance (for pages)
npm run build
npm run lighthouse
# Must: Performance >90, Accessibility >95

# 5. Linting
npm run lint
# Must: Zero lint errors
```

**Backend integration verification:**

```bash
# Verify data flow (example):
# 1. Create API key in portal UI
# 2. Check Firestore (Firebase console)
# 3. Use key to call APX Router
curl https://router-abc123.run.app/v1/example \
  -H "x-apx-api-key: apx_abc123..."
# Must: Return 200 OK (key works immediately)
```

### Step 5: Completion

**Update PORTAL_TASK_TRACKER.yaml:**

```yaml
PM1-T1-001:
  status: COMPLETE
  completed_at: "2025-11-12T15:00:00Z"
  actual_hours: 5
  artifacts:
    - "portal/lib/bigquery.ts"
    - "portal/app/api/dashboard/stats/route.ts"
    - "portal/app/dashboard/page.tsx"
    - "portal/components/dashboard/stats-cards.tsx"
  acceptance_criteria:
    - text: "Dashboard loads stats from BigQuery"
      checked: true
    - text: "Page loads in <2s"
      checked: true
    # ... (all checked: true)
  notes:
    - "2025-11-12T10:00:00Z: Started"
    - "2025-11-12T15:00:00Z: Complete, all criteria met"
```

**Commit artifacts:**

```bash
git add portal/ PORTAL_TASK_TRACKER.yaml
git commit -m "[PM1-T1-001] Dashboard with live APX stats - COMPLETE

- Implemented BigQuery client for usage data
- Created dashboard with stats cards (requests, latency, errors)
- Integrated with APX Router health check
- Responsive design, dark mode support
- E2E tests passing
- Lighthouse score: 94

All acceptance criteria met."

git push
```

---

## Code Quality Standards

### TypeScript

**Strict mode required:**

```typescript
// tsconfig.json (already configured)
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "strictFunctionTypes": true
  }
}
```

**Zod for validation:**

```typescript
import { z } from 'zod'

// Define schemas for all external data
const UserSchema = z.object({
  id: z.string(),
  email: z.string().email(),
  name: z.string(),
})

// Validate at boundaries (API routes, external APIs)
const data = UserSchema.parse(unknownData)
```

**Type inference preferred:**

```typescript
// Good:
const stats = await getDashboardStats(userId)
// stats is inferred as DashboardStats

// Avoid:
const stats: DashboardStats = await getDashboardStats(userId)
```

### React Components

**Server Components by default:**

```typescript
// app/dashboard/page.tsx
// Server Component (no 'use client')
export default async function DashboardPage() {
  const stats = await getDashboardStats() // Direct DB call
  return <StatsCards stats={stats} />
}
```

**Client Components when needed:**

```typescript
// components/dashboard/stats-cards.tsx
'use client'

import { useState, useEffect } from 'react'

export function StatsCards() {
  const [stats, setStats] = useState(null)
  // Interactivity, hooks, browser APIs
  useEffect(() => { /* ... */ }, [])
  return <div>...</div>
}
```

**Component patterns:**

```typescript
// Use shadcn/ui components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

// Prefer composition
export function Dashboard() {
  return (
    <div className="container">
      <StatsCards />
      <RequestsChart />
      <RecentRequests />
    </div>
  )
}
```

### Tailwind CSS

**Use design system classes:**

```typescript
// Good:
<div className="container py-8 space-y-4">
  <h1 className="text-4xl font-bold">Dashboard</h1>
  <p className="text-muted-foreground">Overview of your API usage</p>
</div>

// Avoid:
<div style={{ padding: '2rem', display: 'flex', gap: '1rem' }}>
  <h1 style={{ fontSize: '2.25rem', fontWeight: 'bold' }}>Dashboard</h1>
</div>
```

**Responsive patterns:**

```typescript
<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
  {/* Stacks on mobile, 2 cols on tablet, 4 cols on desktop */}
</div>
```

**Dark mode:**

```typescript
// Use dark: variant
<div className="bg-white dark:bg-gray-900">
  <p className="text-gray-900 dark:text-gray-100">Text</p>
</div>
```

### API Routes (Next.js)

**Authentication required:**

```typescript
import { getServerSession } from 'next-auth'
import { authOptions } from '@/app/api/auth/[...nextauth]/route'

export async function GET() {
  const session = await getServerSession(authOptions)

  if (!session?.user?.id) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  // ... proceed with authenticated user
}
```

**Error handling:**

```typescript
export async function POST(req: NextRequest) {
  try {
    const body = await req.json()
    const data = CreateKeySchema.parse(body) // Zod validation

    const key = await createAPIKey(session.user.id, data)

    return NextResponse.json(key, { status: 201 })
  } catch (error) {
    console.error('Create key error:', error)

    if (error instanceof z.ZodError) {
      return NextResponse.json(
        { error: 'Invalid input', details: error.errors },
        { status: 400 }
      )
    }

    return NextResponse.json(
      { error: 'Failed to create key' },
      { status: 500 }
    )
  }
}
```

**Rate limiting (future):**

```typescript
// TODO: Add rate limiting middleware
// Example: max 100 req/min per user
```

### Backend Integration Patterns

**BigQuery queries:**

```typescript
import { BigQuery } from '@google-cloud/bigquery'

const bigquery = new BigQuery({
  projectId: process.env.GCP_PROJECT_ID,
  credentials: {
    client_email: process.env.FIREBASE_CLIENT_EMAIL,
    private_key: process.env.FIREBASE_PRIVATE_KEY?.replace(/\\n/g, '\n'),
  },
})

// Always parameterize queries
const query = `
  SELECT COUNT(*) as total
  FROM \`${PROJECT_ID}.${DATASET}.${TABLE}\`
  WHERE user_id = @userId
    AND timestamp >= @start_date
`

const [rows] = await bigquery.query({
  query,
  params: {
    userId: session.user.id,
    start_date: '2025-11-01',
  },
})
```

**Firestore operations:**

```typescript
import { db } from '@/lib/firestore/client'

// Create
await db.collection('api_keys').doc(keyId).set({
  id: keyId,
  user_id: userId,
  created_at: new Date().toISOString(),
})

// Read
const doc = await db.collection('api_keys').doc(keyId).get()
if (doc.exists) {
  const key = doc.data()
}

// Query
const snapshot = await db
  .collection('api_keys')
  .where('user_id', '==', userId)
  .where('status', '==', 'active')
  .get()

const keys = snapshot.docs.map(doc => doc.data())

// Update
await db.collection('api_keys').doc(keyId).update({
  status: 'revoked',
})

// Delete (prefer soft delete)
await db.collection('api_keys').doc(keyId).update({
  status: 'deleted',
  deleted_at: new Date().toISOString(),
})
```

**APX Router calls:**

```typescript
const APX_ROUTER_URL = process.env.NEXT_PUBLIC_APX_ROUTER_URL

// Always include trace headers
const res = await fetch(`${APX_ROUTER_URL}${endpoint}`, {
  method,
  headers: {
    'Content-Type': 'application/json',
    'x-apx-api-key': apiKey,
    'x-apx-request-id': crypto.randomUUID(),
  },
  body: JSON.stringify(data),
})

// Check status
if (!res.ok) {
  throw new Error(`APX Router error: ${res.status}`)
}

const result = await res.json()
```

---

## Testing Standards

### Unit Tests (Jest + Testing Library)

**Component tests:**

```typescript
// __tests__/components/dashboard/stats-cards.test.tsx
import { render, screen } from '@testing-library/react'
import { StatsCards } from '@/components/dashboard/stats-cards'

describe('StatsCards', () => {
  it('displays request count', async () => {
    render(<StatsCards />)

    // Wait for data to load
    const count = await screen.findByText(/1,234/)
    expect(count).toBeInTheDocument()
  })

  it('shows error state when API fails', async () => {
    // Mock fetch to fail
    global.fetch = jest.fn(() => Promise.reject('API error'))

    render(<StatsCards />)

    const error = await screen.findByText(/Failed to load/)
    expect(error).toBeInTheDocument()
  })
})
```

**API route tests:**

```typescript
// __tests__/api/keys.test.ts
import { POST } from '@/app/api/keys/route'
import { NextRequest } from 'next/server'

describe('POST /api/keys', () => {
  it('creates API key with valid data', async () => {
    const req = new NextRequest('http://localhost/api/keys', {
      method: 'POST',
      body: JSON.stringify({
        name: 'Test Key',
        scopes: ['product:payments'],
      }),
    })

    const res = await POST(req)
    const data = await res.json()

    expect(res.status).toBe(201)
    expect(data.id).toMatch(/^apx_/)
    expect(data.name).toBe('Test Key')
  })

  it('rejects invalid scopes', async () => {
    const req = new NextRequest('http://localhost/api/keys', {
      method: 'POST',
      body: JSON.stringify({
        name: 'Test Key',
        scopes: [], // Invalid: empty
      }),
    })

    const res = await POST(req)

    expect(res.status).toBe(400)
  })
})
```

### E2E Tests (Playwright)

**Critical user flows:**

```typescript
// tests/e2e/api-keys.spec.ts
import { test, expect } from '@playwright/test'

test.describe('API Keys Management', () => {
  test.beforeEach(async ({ page }) => {
    // Sign in
    await page.goto('/auth/signin')
    await page.click('button:has-text("Sign in with Google")')
    // ... complete OAuth flow
    await expect(page).toHaveURL('/dashboard')
  })

  test('create API key flow', async ({ page }) => {
    // Navigate to keys
    await page.click('a:has-text("API Keys")')
    await expect(page).toHaveURL('/keys')

    // Click create
    await page.click('button:has-text("Create Key")')

    // Fill form
    await page.fill('input[name="name"]', 'Test Key')
    await page.click('input[value="product:payments"]')

    // Submit
    await page.click('button:has-text("Create")')

    // Verify success
    await expect(page.locator('text=Test Key')).toBeVisible()
    await expect(page.locator('text=apx_')).toBeVisible()
  })

  test('revoke API key', async ({ page }) => {
    await page.goto('/keys')

    // Click revoke on first key
    await page.click('button:has-text("Revoke")').first()

    // Confirm dialog
    await page.click('button:has-text("Confirm")')

    // Verify revoked
    await expect(page.locator('text=Revoked')).toBeVisible()
  })
})
```

**Test APX integration:**

```typescript
// tests/e2e/api-console.spec.ts
test('API console makes real request to APX Router', async ({ page }) => {
  await page.goto('/products/example-product/console')

  // Enter API key
  await page.fill('input[placeholder="apx_..."]', process.env.TEST_API_KEY!)

  // Select endpoint
  await page.selectOption('select[name="method"]', 'GET')
  await page.fill('input[name="endpoint"]', '/health')

  // Send request
  await page.click('button:has-text("Send Request")')

  // Verify response
  await expect(page.locator('text=200')).toBeVisible()
  await expect(page.locator('text=healthy')).toBeVisible()

  // Check trace
  await page.click('text=Trace')
  const requestId = await page.locator('code').first().textContent()
  expect(requestId).toMatch(/^[0-9a-f-]{36}$/) // UUID format
})
```

### Accessibility Tests (Axe)

**Automated scans:**

```typescript
// tests/a11y/pages.spec.ts
import { test, expect } from '@playwright/test'
import { injectAxe, checkA11y } from 'axe-playwright'

test.describe('Accessibility', () => {
  test('dashboard has no violations', async ({ page }) => {
    await page.goto('/dashboard')
    await injectAxe(page)
    await checkA11y(page)
  })

  test('API console keyboard navigable', async ({ page }) => {
    await page.goto('/products/example/console')

    // Tab through form
    await page.keyboard.press('Tab') // Method select
    await page.keyboard.press('Tab') // Endpoint input
    await page.keyboard.press('Tab') // API key input
    await page.keyboard.press('Tab') // Send button

    // Enter should submit
    await page.keyboard.press('Enter')
    // ... verify request sent
  })
})
```

### Performance Tests (Lighthouse)

**CI integration:**

```yaml
# .github/workflows/lighthouse.yml
name: Lighthouse CI

on: [pull_request]

jobs:
  lighthouse:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: npm run build
      - uses: treosh/lighthouse-ci-action@v9
        with:
          urls: |
            http://localhost:3000/
            http://localhost:3000/dashboard
            http://localhost:3000/products
          budgetPath: ./lighthouserc.json
```

**Budget configuration:**

```json
// lighthouserc.json
{
  "ci": {
    "assert": {
      "preset": "lighthouse:recommended",
      "assertions": {
        "categories:performance": ["error", { "minScore": 0.9 }],
        "categories:accessibility": ["error", { "minScore": 0.95 }],
        "categories:best-practices": ["error", { "minScore": 0.9 }],
        "categories:seo": ["error", { "minScore": 0.9 }]
      }
    }
  }
}
```

---

## Handling Blockers

### Types of Blockers

1. **Dependency Not Met**
   ```yaml
   # Task PM1-T1-003 needs PM1-T2-001
   # PM1-T2-001 status: IN_PROGRESS

   # Action: Wait or pick another task
   # Do NOT implement workarounds
   ```

2. **Backend Service Down**
   ```bash
   # APX Router unreachable
   curl https://router-abc123.run.app/health
   # Error: connection refused

   # Action: Mark task as BLOCKED
   blockers:
     - task: PM1-T1-001
       type: BACKEND_DOWN
       description: "APX Router unreachable"
       assigned_to: human-coordinator

   # Notify coordinator immediately
   ```

3. **Design Decision Needed**
   ```yaml
   # Example: API key rotation UX unclear

   # Action: Document options, ask for decision
   blockers:
     - task: PM1-T2-003
       type: DESIGN_DECISION
       description: "Key rotation UX: in-place edit vs modal vs wizard?"
       options:
         - "In-place edit (inline form)"
         - "Modal with 2-step wizard"
         - "Full-page wizard flow"
       assigned_to: human-product-owner
   ```

4. **Technical Issue**
   ```yaml
   # Example: BigQuery query timeout

   # Action: Document issue, try alternatives
   notes:
     - "BigQuery query timing out for >100k rows"
     - "Attempted: pagination, query optimization"
     - "Need: streaming results or pre-aggregation"

   # If unresolved after 2 hours, escalate
   ```

### Escalation Process

```yaml
# After 2 hours stuck:
1. Update PORTAL_TASK_TRACKER.yaml:
   status: BLOCKED
   blocker: { type: ..., description: ... }

2. Commit and push

3. Notify human coordinator:
   - Slack: #apx-portal-dev
   - GitHub issue: "Blocker: [PM1-T1-001] ..."

4. Pick another task while waiting
```

---

## Communication Protocol

### Agent ↔ Agent

**Medium:** PORTAL_TASK_TRACKER.yaml notes

```yaml
# Example: Frontend agent needs backend API
PM1-T1-003:
  notes:
    - "2025-11-12T14:00:00Z: Ready to integrate, waiting for PM1-T2-003"

PM1-T2-003:
  notes:
    - "2025-11-12T15:30:00Z: Usage API complete, /api/usage/[keyId] live"
```

### Agent ↔ Backend Team

**Medium:** GitHub issues, Slack #apx-integration

**When:** New endpoints needed, schema changes

```markdown
## New Endpoint Request

**Task:** PM2-T1-002 (Request Explorer)
**Agent:** agent-frontend-1

**Need:** BigQuery API to search requests by filters

**Endpoint:** GET /api/requests/search
**Query params:**
- request_id (optional)
- tenant (optional)
- date_from (required)
- date_to (required)
- limit (default: 100, max: 1000)

**Response:**
```json
{
  "requests": [
    {
      "request_id": "...",
      "timestamp": "...",
      "method": "GET",
      "path": "/v1/example",
      "status": 200,
      "latency_ms": 45
    }
  ],
  "total": 1234,
  "has_more": true
}
```

**ETA needed:** 2025-11-15
```

### Agent ↔ Human Coordinator

**Medium:** GitHub issues, Slack

**When:** Blockers, approvals, questions

**Template:**

```markdown
## Task Update: [PM1-T1-001]

**Status:** BLOCKED
**Agent:** agent-frontend-1
**Started:** 2025-11-12T10:00:00Z
**Blocked at:** 2025-11-12T14:00:00Z

**Issue:**
BigQuery quota exceeded (10k queries/day limit hit)

**Attempted:**
- Query optimization (reduced from 50s to 5s)
- Caching (30min TTL)
- Still hitting limit due to dashboard auto-refresh

**Need:**
- Increase BigQuery quota to 100k queries/day, OR
- Approve pre-aggregation approach (Cloud Run job every hour)

**Impact:**
- PM1-T1-001 blocked (dashboard)
- PM2-T1-001 blocked (charts)
- 2 days delay if not resolved
```

---

## Git Workflow

### Branch Strategy

```bash
# Main branch
main  # Production-ready code

# Feature branches (per task)
git checkout -b PM1-T1-001-dashboard
# ... work
git commit -m "[PM1-T1-001] Implement dashboard"
git push origin PM1-T1-001-dashboard

# Open PR
gh pr create \
  --title "[PM1-T1-001] Dashboard with live APX stats" \
  --body "Closes #123. Implements dashboard with BigQuery integration."
```

### Commit Messages

**Format:**

```
[TASK-ID] Short summary (50 chars)

- Detailed point 1
- Detailed point 2
- Integration: APX Router, BigQuery

Acceptance criteria met:
- [x] Dashboard loads stats
- [x] Page loads in <2s
- [x] Responsive design

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

---

## Success Checklist

Before marking task COMPLETE, verify:

- [ ] All acceptance criteria checked: true
- [ ] TypeScript: Zero errors
- [ ] Tests: All passing (unit + E2E if applicable)
- [ ] Accessibility: Zero Axe violations
- [ ] Performance: Lighthouse >90 (if UI page)
- [ ] Backend integration: Data flows correctly
- [ ] Mobile responsive: Tested 375px, 768px, 1440px
- [ ] Dark mode: Works in both themes
- [ ] Error states: Handled gracefully
- [ ] Loading states: Skeleton or spinner
- [ ] PORTAL_TASK_TRACKER.yaml: Updated with artifacts, notes
- [ ] Git: Committed and pushed

---

## Example Full Session

```bash
# 1. Start of day
cd /Users/agentsy/APILEE
git pull

# 2. Read task tracker
cat PORTAL_TASK_TRACKER.yaml
# Identify: PM1-T1-001 (Dashboard)

# 3. Claim task
vim PORTAL_TASK_TRACKER.yaml
# Update: status: IN_PROGRESS, assigned_to: agent-frontend-1
git add PORTAL_TASK_TRACKER.yaml
git commit -m "[PM1-T1-001] Claiming Dashboard task"
git push

# 4. Verify backend
curl https://router-abc123.run.app/health
# ✅ 200 OK {"status":"healthy"}

# 5. Create branch
git checkout -b PM1-T1-001-dashboard

# 6. Execute task (from PORTAL_AGENT_EXECUTION_PLAN.md)
cd portal
npm install @google-cloud/bigquery

# Create files
touch lib/bigquery.ts
# ... implement BigQuery client

touch app/api/dashboard/stats/route.ts
# ... implement API route

touch app/dashboard/page.tsx
# ... implement dashboard UI

# 7. Test
npm run type-check  # ✅ Zero errors
npm run test        # ✅ All passing
npm run dev
# Manual: Visit http://localhost:3000/dashboard
# ✅ Stats cards show data

npm run build       # ✅ Build succeeds
npm run lighthouse  # ✅ Score 94

# 8. Commit
git add portal/
git commit -m "[PM1-T1-001] Dashboard with live APX stats - COMPLETE

- Implemented BigQuery client
- Created stats cards (requests, latency, errors)
- Responsive design, dark mode
- E2E tests passing
- Lighthouse: 94

All acceptance criteria met."

git push origin PM1-T1-001-dashboard

# 9. Update tracker
vim PORTAL_TASK_TRACKER.yaml
# Update: status: COMPLETE, actual_hours: 5, artifacts: [...]
git add PORTAL_TASK_TRACKER.yaml
git commit -m "[PM1-T1-001] Task complete, updating tracker"
git push

# 10. Open PR (if required by team)
gh pr create \
  --title "[PM1-T1-001] Dashboard with live APX stats" \
  --body "Implements dashboard with real-time stats from BigQuery."

# 11. Merge (if approved)
gh pr merge --squash

# 12. Next task
# Return to step 2, select PM1-T1-002
```

---

## Final Notes

- **Quality over speed:** Take time to do it right
- **Verify backend integration:** Always test data flow end-to-end
- **Accessibility matters:** Test with keyboard, screen reader
- **Mobile first:** Test on small screens early
- **Document blockers:** Don't stay stuck for >2 hours
- **Ask questions:** Better to clarify than assume

**Let's build an amazing developer portal! 🚀**

---

**Version:** 1.0
**Last Updated:** 2025-11-11
**Maintained by:** Portal Team
