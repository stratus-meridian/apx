# APX Portal Agent Execution Plan

**Blueprint for AI-Assisted Portal Implementation**

**Version:** 1.0
**Last Updated:** 2025-11-11
**Status:** Ready for Agent Execution

---

## Purpose

This document is designed for **AI agents** to implement the APX Developer Portal systematically with **tight integration** to the APX backend (router, edge, workers, BigQuery, Firestore).

Each task is:

1. **Self-contained** - Can be executed independently
2. **Testable** - Has clear acceptance criteria
3. **Backend-integrated** - Connects to real APX services
4. **Tracked** - Has progress markers and completion signals

---

## How to Use This Plan

### For Human Coordinators

1. Assign tasks to agents (frontend, backend, integration specialists)
2. Agents mark progress by updating `Status` in PORTAL_TASK_TRACKER.yaml
3. Track completion via acceptance criteria checkboxes
4. Review generated artifacts before marking complete

### For AI Agents

Each task follows this format:

```yaml
Task ID: PM1-T1-001
Name: Dashboard with Live APX Stats
Agent Type: frontend | backend | integration | docs | testing
Priority: P0 (Critical) | P1 (High) | P2 (Medium) | P3 (Low)
Dependencies: [PM0-T1-001, PM0-T2-001]
Estimated Time: 4 hours
Status: NOT_STARTED | IN_PROGRESS | BLOCKED | REVIEW | COMPLETE

Context:
  - What this task achieves
  - Why it matters
  - Backend integration points (APX router, BigQuery, etc.)

Prerequisites:
  - Files/resources that must exist first
  - APX backend services running
  - Environment variables set

Steps:
  1. Concrete action with command/code
  2. Backend API integration details
  3. Verification step

Acceptance Criteria:
  - [ ] Testable outcome 1
  - [ ] Backend integration verified
  - [ ] Performance/accessibility/UX goals met

Artifacts:
  - file_path: Description of what's created

Backend Integration:
  - APX services used (router, edge, BigQuery)
  - API endpoints called
  - Data flow diagram

Rollback:
  - How to undo this task if needed
```

---

## Milestone 0: Foundation (Weeks 1-2)

**Status:** NOT_STARTED
**Duration:** 2 weeks
**Goal:** Next.js portal shell, auth, backend connectivity

---

### Phase PM0-T1: Frontend Foundation

---

#### Task PM0-T1-001: Next.js 14 Portal Initialization

```yaml
Task ID: PM0-T1-001
Name: Initialize Next.js 14 Portal with App Router
Agent Type: frontend
Priority: P0
Dependencies: []
Estimated Time: 2 hours
Status: NOT_STARTED

Context:
  Sets up the foundational Next.js 14 application with TypeScript,
  Tailwind CSS, and the App Router pattern. This is the base for
  all portal UI work.

Prerequisites:
  - Node.js 18+ installed
  - npm or pnpm available
  - Repository cloned

Steps:
  1. Create Next.js app:
     ```bash
     cd /Users/agentsy/APILEE
     npx create-next-app@14 portal \
       --typescript \
       --tailwind \
       --app \
       --no-src-dir \
       --import-alias "@/*"
     ```

  2. Install core dependencies:
     ```bash
     cd portal
     npm install \
       @radix-ui/react-icons \
       class-variance-authority \
       clsx \
       tailwind-merge \
       zod \
       react-hook-form \
       @hookform/resolvers
     ```

  3. Install dev dependencies:
     ```bash
     npm install -D \
       @types/node \
       @types/react \
       @types/react-dom \
       prettier \
       prettier-plugin-tailwindcss \
       eslint-config-prettier
     ```

  4. Configure TypeScript (tsconfig.json):
     ```json
     {
       "compilerOptions": {
         "target": "ES2020",
         "lib": ["dom", "dom.iterable", "esnext"],
         "allowJs": true,
         "skipLibCheck": true,
         "strict": true,
         "forceConsistentCasingInFileNames": true,
         "noEmit": true,
         "esModuleInterop": true,
         "module": "esnext",
         "moduleResolution": "bundler",
         "resolveJsonModule": true,
         "isolatedModules": true,
         "jsx": "preserve",
         "incremental": true,
         "plugins": [{ "name": "next" }],
         "paths": {
           "@/*": ["./*"]
         }
       },
       "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
       "exclude": ["node_modules"]
     }
     ```

  5. Create environment template:
     ```bash
     cat > .env.example <<EOF
     # APX Backend Integration
     NEXT_PUBLIC_APX_ROUTER_URL=https://router-abc123.run.app
     NEXT_PUBLIC_APX_EDGE_URL=https://edge-abc123.run.app
     APX_INTERNAL_API_KEY=your-internal-api-key-here

     # Firebase/Auth0
     NEXTAUTH_URL=http://localhost:3000
     NEXTAUTH_SECRET=generate-with-openssl-rand-base64-32
     FIREBASE_PROJECT_ID=your-firebase-project
     FIREBASE_CLIENT_EMAIL=your-service-account@project.iam.gserviceaccount.com
     FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"

     # BigQuery (for usage analytics)
     GCP_PROJECT_ID=apx-dev-abc123
     BIGQUERY_DATASET=apx_requests
     BIGQUERY_TABLE=requests

     # Stripe (for billing, optional in M1)
     STRIPE_SECRET_KEY=sk_test_...
     NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...
     EOF

     cp .env.example .env.local
     ```

  6. Verify setup:
     ```bash
     npm run dev
     # Visit http://localhost:3000
     # Should see Next.js welcome page
     ```

Acceptance Criteria:
  - [ ] Next.js dev server starts without errors
  - [ ] TypeScript strict mode enabled, zero errors
  - [ ] Tailwind CSS working (test with className)
  - [ ] .env.local contains APX backend URLs
  - [ ] Can build for production: npm run build

Artifacts:
  - portal/package.json: Dependencies configured
  - portal/tsconfig.json: Strict TypeScript config
  - portal/.env.example: Environment template with APX URLs
  - portal/.env.local: Local environment (gitignored)

Backend Integration:
  - Configures APX_ROUTER_URL for API calls
  - Configures APX_EDGE_URL for health checks
  - Sets up BigQuery connection for analytics

Rollback:
  - rm -rf portal/
```

---

#### Task PM0-T1-002: shadcn/ui Component Library Setup

```yaml
Task ID: PM0-T1-002
Name: Install and configure shadcn/ui component library
Agent Type: frontend
Priority: P0
Dependencies: [PM0-T1-001]
Estimated Time: 1.5 hours
Status: NOT_STARTED

Context:
  shadcn/ui provides accessible, customizable React components
  built on Radix UI primitives. This ensures consistency and
  speeds up UI development.

Prerequisites:
  - PM0-T1-001 complete (Next.js initialized)

Steps:
  1. Initialize shadcn/ui:
     ```bash
     cd portal
     npx shadcn-ui@latest init
     # Select: New York style, Neutral gray, CSS variables
     ```

  2. Install core components:
     ```bash
     npx shadcn-ui@latest add button
     npx shadcn-ui@latest add card
     npx shadcn-ui@latest add input
     npx shadcn-ui@latest add label
     npx shadcn-ui@latest add select
     npx shadcn-ui@latest add dialog
     npx shadcn-ui@latest add dropdown-menu
     npx shadcn-ui@latest add table
     npx shadcn-ui@latest add tabs
     npx shadcn-ui@latest add toast
     npx shadcn-ui@latest add switch
     npx shadcn-ui@latest add badge
     npx shadcn-ui@latest add alert
     npx shadcn-ui@latest add skeleton
     ```

  3. Configure dark mode (app/providers.tsx):
     ```typescript
     'use client'

     import { ThemeProvider } from 'next-themes'
     import { ReactNode } from 'react'

     export function Providers({ children }: { children: ReactNode }) {
       return (
         <ThemeProvider
           attribute="class"
           defaultTheme="system"
           enableSystem
           disableTransitionOnChange
         >
           {children}
         </ThemeProvider>
       )
     }
     ```

  4. Wrap app in providers (app/layout.tsx):
     ```typescript
     import { Providers } from './providers'
     import './globals.css'

     export default function RootLayout({
       children,
     }: {
       children: React.ReactNode
     }) {
       return (
         <html lang="en" suppressHydrationWarning>
           <body>
             <Providers>{children}</Providers>
           </body>
         </html>
       )
     }
     ```

  5. Create theme toggle component (components/theme-toggle.tsx):
     ```typescript
     'use client'

     import { Moon, Sun } from 'lucide-react'
     import { useTheme } from 'next-themes'
     import { Button } from '@/components/ui/button'

     export function ThemeToggle() {
       const { theme, setTheme } = useTheme()

       return (
         <Button
           variant="ghost"
           size="icon"
           onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
           aria-label="Toggle theme"
         >
           <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
           <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
         </Button>
       )
     }
     ```

  6. Test component:
     ```typescript
     // app/page.tsx
     import { Button } from '@/components/ui/button'
     import { ThemeToggle } from '@/components/theme-toggle'

     export default function Home() {
       return (
         <div className="flex min-h-screen items-center justify-center gap-4">
           <Button>Test Button</Button>
           <ThemeToggle />
         </div>
       )
     }
     ```

Acceptance Criteria:
  - [ ] All shadcn/ui components installed in components/ui/
  - [ ] Dark mode toggle works and persists
  - [ ] Button, Card, Input render correctly in both themes
  - [ ] TypeScript has zero errors
  - [ ] Tailwind classes apply correctly

Artifacts:
  - portal/components/ui/*: shadcn/ui components
  - portal/components/theme-toggle.tsx: Theme switcher
  - portal/app/providers.tsx: Theme provider
  - portal/lib/utils.ts: cn() helper for className merging

Backend Integration:
  - N/A (pure frontend)

Rollback:
  - git checkout -- portal/components portal/app/providers.tsx
```

---

### Phase PM0-T2: Backend Integration Foundation

---

#### Task PM0-T2-001: APX Router Health Check Integration

```yaml
Task ID: PM0-T2-001
Name: Connect portal to APX Router health endpoint
Agent Type: integration
Priority: P0
Dependencies: [PM0-T1-001]
Estimated Time: 2 hours
Status: NOT_STARTED

Context:
  This establishes the first connection between the portal and
  the APX backend. It verifies that the router is reachable and
  provides system status information for the dashboard.

Prerequisites:
  - PM0-T1-001 complete (Next.js initialized)
  - APX Router deployed and accessible
  - APX_ROUTER_URL in .env.local

Steps:
  1. Create API client (lib/apx-client.ts):
     ```typescript
     // lib/apx-client.ts
     import { z } from 'zod'

     const APX_ROUTER_URL = process.env.NEXT_PUBLIC_APX_ROUTER_URL
     const APX_INTERNAL_API_KEY = process.env.APX_INTERNAL_API_KEY

     if (!APX_ROUTER_URL) {
       throw new Error('NEXT_PUBLIC_APX_ROUTER_URL not set')
     }

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

     export type Health = z.infer<typeof HealthSchema>

     export async function getRouterHealth(): Promise<Health> {
       const res = await fetch(`${APX_ROUTER_URL}/health`, {
         headers: {
           'x-apx-internal-key': APX_INTERNAL_API_KEY || '',
         },
         cache: 'no-store', // Always fresh
       })

       if (!res.ok) {
         throw new Error(`Router health check failed: ${res.status}`)
       }

       const data = await res.json()
       return HealthSchema.parse(data)
     }
     ```

  2. Create server action (app/actions/health.ts):
     ```typescript
     'use server'

     import { getRouterHealth } from '@/lib/apx-client'

     export async function checkSystemHealth() {
       try {
         const health = await getRouterHealth()
         return { success: true, data: health }
       } catch (error) {
         console.error('Health check failed:', error)
         return {
           success: false,
           error: error instanceof Error ? error.message : 'Unknown error',
         }
       }
     }
     ```

  3. Create status component (components/system-status.tsx):
     ```typescript
     'use client'

     import { useEffect, useState } from 'react'
     import { checkSystemHealth } from '@/app/actions/health'
     import { Badge } from '@/components/ui/badge'
     import { Card } from '@/components/ui/card'
     import type { Health } from '@/lib/apx-client'

     export function SystemStatus() {
       const [health, setHealth] = useState<Health | null>(null)
       const [error, setError] = useState<string | null>(null)

       useEffect(() => {
         const check = async () => {
           const result = await checkSystemHealth()
           if (result.success) {
             setHealth(result.data)
             setError(null)
           } else {
             setError(result.error)
           }
         }

         check()
         const interval = setInterval(check, 30000) // Every 30s

         return () => clearInterval(interval)
       }, [])

       if (error) {
         return <Badge variant="destructive">System Offline</Badge>
       }

       if (!health) {
         return <Badge variant="secondary">Checking...</Badge>
       }

       const statusColor = {
         healthy: 'default',
         degraded: 'warning',
         down: 'destructive',
       } as const

       return (
         <div className="space-y-2">
           <Badge variant={statusColor[health.status]}>
             {health.status.toUpperCase()}
           </Badge>
           <div className="text-sm text-muted-foreground">
             Router v{health.version}
           </div>
         </div>
       )
     }
     ```

  4. Test integration:
     ```typescript
     // app/page.tsx
     import { SystemStatus } from '@/components/system-status'

     export default function Home() {
       return (
         <div className="container py-8">
           <h1 className="text-4xl font-bold mb-4">APX Portal</h1>
           <SystemStatus />
         </div>
       )
     }
     ```

  5. Verify in browser:
     ```bash
     npm run dev
     # Visit http://localhost:3000
     # Should see "HEALTHY" badge if router is up
     # Check Network tab for /health request
     ```

Acceptance Criteria:
  - [ ] Health check successfully fetches from APX Router
  - [ ] Status badge updates every 30 seconds
  - [ ] Error states handled gracefully (offline router)
  - [ ] TypeScript validates Health schema with Zod
  - [ ] No CORS errors (router has correct headers)

Artifacts:
  - portal/lib/apx-client.ts: APX backend client
  - portal/app/actions/health.ts: Server action
  - portal/components/system-status.tsx: Status UI

Backend Integration:
  - APX Router: GET /health
  - Headers: x-apx-internal-key for auth
  - Response: { status, version, timestamp, components }

Rollback:
  - git checkout -- portal/lib portal/app/actions portal/components/system-status.tsx
```

---

#### Task PM0-T2-002: Firebase/Auth0 Authentication Setup

```yaml
Task ID: PM0-T2-002
Name: Implement user authentication with Firebase/Auth0
Agent Type: backend
Priority: P0
Dependencies: [PM0-T1-001]
Estimated Time: 4 hours
Status: NOT_STARTED

Context:
  Users need to authenticate to access the portal. This task
  sets up NextAuth.js with Firebase (or Auth0) and creates
  protected routes. User sessions are stored in Firestore
  (same database as APX backend).

Prerequisites:
  - PM0-T1-001 complete (Next.js initialized)
  - Firebase project created (same as APX backend)
  - Firebase service account key available

Steps:
  1. Install NextAuth.js:
     ```bash
     cd portal
     npm install next-auth firebase-admin
     ```

  2. Create NextAuth config (app/api/auth/[...nextauth]/route.ts):
     ```typescript
     import NextAuth, { AuthOptions } from 'next-auth'
     import GoogleProvider from 'next-auth/providers/google'
     import { FirestoreAdapter } from '@auth/firebase-adapter'
     import { cert } from 'firebase-admin/app'

     export const authOptions: AuthOptions = {
       adapter: FirestoreAdapter({
         credential: cert({
           projectId: process.env.FIREBASE_PROJECT_ID,
           clientEmail: process.env.FIREBASE_CLIENT_EMAIL,
           privateKey: process.env.FIREBASE_PRIVATE_KEY?.replace(/\\n/g, '\n'),
         }),
       }),
       providers: [
         GoogleProvider({
           clientId: process.env.GOOGLE_CLIENT_ID!,
           clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
         }),
       ],
       callbacks: {
         async session({ session, user }) {
           // Add user ID to session
           if (session.user) {
             session.user.id = user.id
           }
           return session
         },
       },
       pages: {
         signIn: '/auth/signin',
       },
     }

     const handler = NextAuth(authOptions)
     export { handler as GET, handler as POST }
     ```

  3. Create sign-in page (app/auth/signin/page.tsx):
     ```typescript
     'use client'

     import { signIn } from 'next-auth/react'
     import { Button } from '@/components/ui/button'
     import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

     export default function SignIn() {
       return (
         <div className="flex min-h-screen items-center justify-center">
           <Card className="w-[400px]">
             <CardHeader>
               <CardTitle>Sign in to APX</CardTitle>
               <CardDescription>
                 Access your API keys, usage analytics, and more
               </CardDescription>
             </CardHeader>
             <CardContent>
               <Button
                 onClick={() => signIn('google', { callbackUrl: '/dashboard' })}
                 className="w-full"
               >
                 Sign in with Google
               </Button>
             </CardContent>
           </Card>
         </div>
       )
     }
     ```

  4. Create auth provider (app/providers.tsx - update):
     ```typescript
     'use client'

     import { SessionProvider } from 'next-auth/react'
     import { ThemeProvider } from 'next-themes'
     import { ReactNode } from 'react'

     export function Providers({ children }: { children: ReactNode }) {
       return (
         <SessionProvider>
           <ThemeProvider
             attribute="class"
             defaultTheme="system"
             enableSystem
             disableTransitionOnChange
           >
             {children}
           </ThemeProvider>
         </SessionProvider>
       )
     }
     ```

  5. Create protected route middleware (middleware.ts):
     ```typescript
     import { withAuth } from 'next-auth/middleware'

     export default withAuth({
       callbacks: {
         authorized({ req, token }) {
           // Protect all routes under /dashboard, /api (except /api/auth)
           if (req.nextUrl.pathname.startsWith('/dashboard')) {
             return !!token
           }
           if (
             req.nextUrl.pathname.startsWith('/api') &&
             !req.nextUrl.pathname.startsWith('/api/auth')
           ) {
             return !!token
           }
           return true
         },
       },
     })

     export const config = {
       matcher: ['/dashboard/:path*', '/api/:path*'],
     }
     ```

  6. Test auth flow:
     ```bash
     # Add to .env.local:
     GOOGLE_CLIENT_ID=your-google-client-id
     GOOGLE_CLIENT_SECRET=your-google-client-secret

     npm run dev
     # Visit http://localhost:3000/dashboard
     # Should redirect to /auth/signin
     # Sign in with Google
     # Should redirect back to /dashboard
     ```

Acceptance Criteria:
  - [ ] Sign-in flow works end-to-end
  - [ ] User sessions stored in Firestore (same DB as APX backend)
  - [ ] Protected routes redirect to /auth/signin
  - [ ] Session persists after page refresh
  - [ ] Sign-out works correctly

Artifacts:
  - portal/app/api/auth/[...nextauth]/route.ts: NextAuth config
  - portal/app/auth/signin/page.tsx: Sign-in page
  - portal/middleware.ts: Route protection
  - portal/app/providers.tsx: Session provider

Backend Integration:
  - Firestore: Users, sessions, accounts (collections)
  - Shares same Firebase project as APX router
  - User IDs from portal match API key ownership in Firestore

Rollback:
  - git checkout -- portal/app/api/auth portal/app/auth portal/middleware.ts
  - npm uninstall next-auth firebase-admin
```

---

## Milestone 1: Core Portal (Weeks 3-6)

**Status:** NOT_STARTED
**Duration:** 4 weeks
**Goal:** Dashboard, API keys, product catalog, Try It console, usage charts

---

### Phase PM1-T1: Core UI Pages

---

#### Task PM1-T1-001: Dashboard with Live APX Stats

```yaml
Task ID: PM1-T1-001
Name: Build dashboard with real-time stats from BigQuery
Agent Type: frontend + integration
Priority: P0
Dependencies: [PM0-T1-002, PM0-T2-001, PM0-T2-002]
Estimated Time: 6 hours
Status: NOT_STARTED

Context:
  The dashboard is the landing page after login. It shows:
  - Total requests (24h, 7d, 30d)
  - p95 latency from BigQuery
  - Error rate
  - Quota usage (from Firestore)
  - Recent requests (live tail)

  Data flows from APX Edge → BigQuery → Portal API → Dashboard UI

Prerequisites:
  - PM0-T1-002 complete (shadcn/ui installed)
  - PM0-T2-001 complete (Router health check working)
  - PM0-T2-002 complete (Auth working)
  - APX backend writing to BigQuery (table: apx_requests.requests)

Steps:
  1. Create BigQuery client (lib/bigquery.ts):
     ```typescript
     import { BigQuery } from '@google-cloud/bigquery'

     const bigquery = new BigQuery({
       projectId: process.env.GCP_PROJECT_ID,
       credentials: {
         client_email: process.env.FIREBASE_CLIENT_EMAIL,
         private_key: process.env.FIREBASE_PRIVATE_KEY?.replace(/\\n/g, '\n'),
       },
     })

     const DATASET = process.env.BIGQUERY_DATASET || 'apx_requests'
     const TABLE = process.env.BIGQUERY_TABLE || 'requests'

     export interface DashboardStats {
       requests_24h: number
       requests_7d: number
       requests_30d: number
       latency_p95_ms: number
       error_rate_percent: number
       quota_used_percent: number
     }

     export async function getDashboardStats(
       userId: string
     ): Promise<DashboardStats> {
       // Query BigQuery for user's API keys
       const query = `
         WITH user_keys AS (
           SELECT key_id
           FROM \`${process.env.GCP_PROJECT_ID}.${DATASET}.api_keys\`
           WHERE user_id = @userId
         )
         SELECT
           COUNTIF(timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)) AS requests_24h,
           COUNTIF(timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)) AS requests_7d,
           COUNTIF(timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)) AS requests_30d,
           APPROX_QUANTILES(latency_ms, 100)[OFFSET(95)] AS latency_p95_ms,
           COUNTIF(status_code >= 400) * 100.0 / COUNT(*) AS error_rate_percent
         FROM \`${process.env.GCP_PROJECT_ID}.${DATASET}.${TABLE}\`
         WHERE api_key IN (SELECT key_id FROM user_keys)
           AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
       `

       const [rows] = await bigquery.query({
         query,
         params: { userId },
       })

       return rows[0] as DashboardStats
     }
     ```

  2. Create API route (app/api/dashboard/stats/route.ts):
     ```typescript
     import { NextResponse } from 'next/server'
     import { getServerSession } from 'next-auth'
     import { authOptions } from '@/app/api/auth/[...nextauth]/route'
     import { getDashboardStats } from '@/lib/bigquery'

     export async function GET() {
       const session = await getServerSession(authOptions)

       if (!session?.user?.id) {
         return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
       }

       try {
         const stats = await getDashboardStats(session.user.id)
         return NextResponse.json(stats)
       } catch (error) {
         console.error('Dashboard stats error:', error)
         return NextResponse.json(
           { error: 'Failed to fetch stats' },
           { status: 500 }
         )
       }
     }
     ```

  3. Create dashboard page (app/dashboard/page.tsx):
     ```typescript
     import { StatsCards } from '@/components/dashboard/stats-cards'
     import { RequestsChart } from '@/components/dashboard/requests-chart'
     import { RecentRequests } from '@/components/dashboard/recent-requests'

     export default async function DashboardPage() {
       return (
         <div className="container py-8 space-y-8">
           <div>
             <h1 className="text-4xl font-bold">Dashboard</h1>
             <p className="text-muted-foreground">
               Overview of your API usage and performance
             </p>
           </div>

           <StatsCards />
           <RequestsChart />
           <RecentRequests />
         </div>
       )
     }
     ```

  4. Create stats cards component (components/dashboard/stats-cards.tsx):
     ```typescript
     'use client'

     import { useEffect, useState } from 'react'
     import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
     import { Activity, TrendingUp, AlertCircle, Zap } from 'lucide-react'

     interface Stats {
       requests_24h: number
       requests_7d: number
       latency_p95_ms: number
       error_rate_percent: number
     }

     export function StatsCards() {
       const [stats, setStats] = useState<Stats | null>(null)

       useEffect(() => {
         fetch('/api/dashboard/stats')
           .then((res) => res.json())
           .then(setStats)
       }, [])

       if (!stats) {
         return <div>Loading...</div>
       }

       return (
         <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
           <Card>
             <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
               <CardTitle className="text-sm font-medium">Requests (24h)</CardTitle>
               <Activity className="h-4 w-4 text-muted-foreground" />
             </CardHeader>
             <CardContent>
               <div className="text-2xl font-bold">
                 {stats.requests_24h.toLocaleString()}
               </div>
             </CardContent>
           </Card>

           <Card>
             <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
               <CardTitle className="text-sm font-medium">Requests (7d)</CardTitle>
               <TrendingUp className="h-4 w-4 text-muted-foreground" />
             </CardHeader>
             <CardContent>
               <div className="text-2xl font-bold">
                 {stats.requests_7d.toLocaleString()}
               </div>
             </CardContent>
           </Card>

           <Card>
             <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
               <CardTitle className="text-sm font-medium">p95 Latency</CardTitle>
               <Zap className="h-4 w-4 text-muted-foreground" />
             </CardHeader>
             <CardContent>
               <div className="text-2xl font-bold">{stats.latency_p95_ms}ms</div>
             </CardContent>
           </Card>

           <Card>
             <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
               <CardTitle className="text-sm font-medium">Error Rate</CardTitle>
               <AlertCircle className="h-4 w-4 text-muted-foreground" />
             </CardHeader>
             <CardContent>
               <div className="text-2xl font-bold">
                 {stats.error_rate_percent.toFixed(2)}%
               </div>
             </CardContent>
           </Card>
         </div>
       )
     }
     ```

  5. Install BigQuery SDK:
     ```bash
     npm install @google-cloud/bigquery
     ```

  6. Test dashboard:
     ```bash
     npm run dev
     # Sign in
     # Visit http://localhost:3000/dashboard
     # Should see stats cards with real data from BigQuery
     # Check Network tab for /api/dashboard/stats request
     ```

Acceptance Criteria:
  - [ ] Dashboard loads stats from BigQuery
  - [ ] Stats cards show: requests (24h, 7d), p95 latency, error rate
  - [ ] Data is scoped to user's API keys only
  - [ ] Page loads in <2s (Lighthouse performance >90)
  - [ ] Responsive on mobile, tablet, desktop
  - [ ] Error states handled gracefully (BigQuery down)

Artifacts:
  - portal/lib/bigquery.ts: BigQuery client
  - portal/app/api/dashboard/stats/route.ts: Stats API
  - portal/app/dashboard/page.tsx: Dashboard page
  - portal/components/dashboard/stats-cards.tsx: Stats UI

Backend Integration:
  - BigQuery: apx_requests.requests table
  - Queries: 24h/7d/30d request counts, p95 latency, error rate
  - Firestore: api_keys collection (to get user's keys)
  - Data flow: Edge → BigQuery → Portal API → Dashboard

Rollback:
  - git checkout -- portal/lib/bigquery.ts portal/app/dashboard portal/components/dashboard
  - npm uninstall @google-cloud/bigquery
```

---

#### Task PM1-T1-003: API Console "Try It" with Live Router Calls

```yaml
Task ID: PM1-T1-003
Name: Interactive API console that calls APX Router
Agent Type: frontend + integration
Priority: P0
Dependencies: [PM0-T1-002, PM0-T2-001, PM1-T2-001]
Estimated Time: 8 hours
Status: NOT_STARTED

Context:
  This is the "Try It" console where users can:
  - Select a product/route from their catalog
  - Auto-populate headers (x-apx-api-key, x-apx-request-id)
  - Edit request body (if POST/PUT)
  - Execute request against APX Router
  - See response with syntax highlighting
  - View request trace (request ID → BigQuery)
  - Copy as cURL, Node, Python

  This creates the "5 minute first call" experience.

Prerequisites:
  - PM0-T1-002 complete (shadcn/ui)
  - PM0-T2-001 complete (Router integration)
  - PM1-T2-001 complete (API keys CRUD)
  - User has at least one API key created

Steps:
  1. Create API console page (app/products/[productId]/console/page.tsx):
     ```typescript
     'use client'

     import { useState } from 'react'
     import { Button } from '@/components/ui/button'
     import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
     import { Input } from '@/components/ui/input'
     import { Label } from '@/components/ui/label'
     import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
     import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
     import { CodeBlock } from '@/components/code-block'

     export default function APIConsolePage({
       params,
     }: {
       params: { productId: string }
     }) {
       const [method, setMethod] = useState('GET')
       const [endpoint, setEndpoint] = useState('/v1/example')
       const [apiKey, setApiKey] = useState('')
       const [body, setBody] = useState('{}')
       const [response, setResponse] = useState<any>(null)
       const [loading, setLoading] = useState(false)

       const executeRequest = async () => {
         setLoading(true)
         try {
           const requestId = crypto.randomUUID()
           const res = await fetch('/api/proxy', {
             method: 'POST',
             headers: { 'Content-Type': 'application/json' },
             body: JSON.stringify({
               method,
               endpoint,
               apiKey,
               body: method !== 'GET' ? JSON.parse(body) : undefined,
               requestId,
             }),
           })

           const data = await res.json()
           setResponse({
             status: res.status,
             headers: data.headers,
             body: data.body,
             requestId,
             latency: data.latency_ms,
           })
         } catch (error) {
           setResponse({ error: String(error) })
         } finally {
           setLoading(false)
         }
       }

       return (
         <div className="container py-8 space-y-8">
           <div>
             <h1 className="text-4xl font-bold">API Console</h1>
             <p className="text-muted-foreground">Test your API in real-time</p>
           </div>

           <div className="grid lg:grid-cols-2 gap-8">
             {/* Request Panel */}
             <Card>
               <CardHeader>
                 <CardTitle>Request</CardTitle>
               </CardHeader>
               <CardContent className="space-y-4">
                 <div className="flex gap-2">
                   <Select value={method} onValueChange={setMethod}>
                     <SelectTrigger className="w-[100px]">
                       <SelectValue />
                     </SelectTrigger>
                     <SelectContent>
                       <SelectItem value="GET">GET</SelectItem>
                       <SelectItem value="POST">POST</SelectItem>
                       <SelectItem value="PUT">PUT</SelectItem>
                       <SelectItem value="DELETE">DELETE</SelectItem>
                     </SelectContent>
                   </Select>
                   <Input
                     value={endpoint}
                     onChange={(e) => setEndpoint(e.target.value)}
                     placeholder="/v1/example"
                   />
                 </div>

                 <div>
                   <Label>API Key</Label>
                   <Input
                     type="password"
                     value={apiKey}
                     onChange={(e) => setApiKey(e.target.value)}
                     placeholder="apx_..."
                   />
                 </div>

                 {method !== 'GET' && (
                   <div>
                     <Label>Body (JSON)</Label>
                     <textarea
                       value={body}
                       onChange={(e) => setBody(e.target.value)}
                       className="w-full min-h-[200px] p-2 border rounded font-mono text-sm"
                     />
                   </div>
                 )}

                 <Button onClick={executeRequest} disabled={loading} className="w-full">
                   {loading ? 'Sending...' : 'Send Request'}
                 </Button>
               </CardContent>
             </Card>

             {/* Response Panel */}
             <Card>
               <CardHeader>
                 <CardTitle>Response</CardTitle>
               </CardHeader>
               <CardContent>
                 {response ? (
                   <Tabs defaultValue="body">
                     <TabsList>
                       <TabsTrigger value="body">Body</TabsTrigger>
                       <TabsTrigger value="headers">Headers</TabsTrigger>
                       <TabsTrigger value="trace">Trace</TabsTrigger>
                     </TabsList>

                     <TabsContent value="body" className="space-y-2">
                       <div className="flex items-center gap-2 text-sm">
                         <span className="font-semibold">Status:</span>
                         <span
                           className={
                             response.status < 400
                               ? 'text-green-600'
                               : 'text-red-600'
                           }
                         >
                           {response.status}
                         </span>
                         <span className="text-muted-foreground">
                           ({response.latency}ms)
                         </span>
                       </div>
                       <CodeBlock
                         language="json"
                         code={JSON.stringify(response.body, null, 2)}
                       />
                     </TabsContent>

                     <TabsContent value="headers">
                       <CodeBlock
                         language="json"
                         code={JSON.stringify(response.headers, null, 2)}
                       />
                     </TabsContent>

                     <TabsContent value="trace" className="space-y-2">
                       <div>
                         <span className="text-sm font-semibold">Request ID:</span>
                         <code className="ml-2 text-xs">{response.requestId}</code>
                       </div>
                       <Button
                         variant="outline"
                         onClick={() =>
                           window.open(`/requests/${response.requestId}`, '_blank')
                         }
                       >
                         View Full Trace
                       </Button>
                     </TabsContent>
                   </Tabs>
                 ) : (
                   <p className="text-muted-foreground">No response yet</p>
                 )}
               </CardContent>
             </Card>
           </div>
         </div>
       )
     }
     ```

  2. Create proxy API route (app/api/proxy/route.ts):
     ```typescript
     import { NextRequest, NextResponse } from 'next/server'
     import { getServerSession } from 'next-auth'
     import { authOptions } from '@/app/api/auth/[...nextauth]/route'

     const APX_ROUTER_URL = process.env.NEXT_PUBLIC_APX_ROUTER_URL

     export async function POST(req: NextRequest) {
       const session = await getServerSession(authOptions)
       if (!session) {
         return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
       }

       const { method, endpoint, apiKey, body, requestId } = await req.json()

       const startTime = Date.now()

       try {
         const res = await fetch(`${APX_ROUTER_URL}${endpoint}`, {
           method,
           headers: {
             'Content-Type': 'application/json',
             'x-apx-api-key': apiKey,
             'x-apx-request-id': requestId,
           },
           body: body ? JSON.stringify(body) : undefined,
         })

         const responseData = await res.json()
         const latency_ms = Date.now() - startTime

         return NextResponse.json({
           status: res.status,
           headers: Object.fromEntries(res.headers.entries()),
           body: responseData,
           latency_ms,
         })
       } catch (error) {
         return NextResponse.json(
           { error: String(error) },
           { status: 500 }
         )
       }
     }
     ```

  3. Create code block component (components/code-block.tsx):
     ```typescript
     'use client'

     import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
     import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'

     export function CodeBlock({
       code,
       language = 'json',
     }: {
       code: string
       language?: string
     }) {
       return (
         <SyntaxHighlighter
           language={language}
           style={oneDark}
           customStyle={{
             borderRadius: '0.5rem',
             fontSize: '0.875rem',
           }}
         >
           {code}
         </SyntaxHighlighter>
       )
     }
     ```

  4. Install dependencies:
     ```bash
     npm install react-syntax-highlighter
     npm install -D @types/react-syntax-highlighter
     ```

  5. Test API console:
     ```bash
     npm run dev
     # Sign in
     # Visit /products/example-product/console
     # Enter API key
     # Select GET method, endpoint: /health
     # Click "Send Request"
     # Should see response from APX Router
     # Check "Trace" tab for request ID
     ```

Acceptance Criteria:
  - [ ] API console successfully calls APX Router
  - [ ] Request ID propagates (portal → router → trace)
  - [ ] Response shows status, headers, body, latency
  - [ ] Syntax highlighting works for JSON responses
  - [ ] Error states handled (invalid key, network error)
  - [ ] "View Full Trace" links to request detail page
  - [ ] Mobile responsive (stacked panels)

Artifacts:
  - portal/app/products/[productId]/console/page.tsx: Console UI
  - portal/app/api/proxy/route.ts: Proxy to APX Router
  - portal/components/code-block.tsx: Syntax highlighter

Backend Integration:
  - APX Router: Proxies all HTTP methods to router
  - Headers: x-apx-api-key (user's key), x-apx-request-id (trace)
  - Response: Full APX Router response (status, headers, body)
  - Trace: Request ID can be looked up in BigQuery later

Rollback:
  - git checkout -- portal/app/products portal/app/api/proxy portal/components/code-block.tsx
  - npm uninstall react-syntax-highlighter @types/react-syntax-highlighter
```

---

### Phase PM1-T2: Backend API Routes

---

#### Task PM1-T2-001: API Keys CRUD with Firestore

```yaml
Task ID: PM1-T2-001
Name: API keys create/read/update/delete with Firestore backend
Agent Type: backend
Priority: P0
Dependencies: [PM0-T2-002]
Estimated Time: 5 hours
Status: NOT_STARTED

Context:
  Users need to manage API keys from the portal. Keys are stored
  in Firestore (same database as APX router uses for validation).

  This creates tight integration: Portal creates key → Firestore →
  APX Router reads key → validates request.

Prerequisites:
  - PM0-T2-002 complete (Auth working)
  - Firestore initialized (same project as APX backend)
  - api_keys collection schema defined

Steps:
  1. Define Firestore schema (lib/firestore/schema.ts):
     ```typescript
     import { z } from 'zod'

     export const APIKeySchema = z.object({
       id: z.string(), // apx_...
       user_id: z.string(),
       name: z.string(),
       scopes: z.array(z.string()), // ['product:payments', 'product:users']
       created_at: z.string(),
       last_used_at: z.string().nullable(),
       status: z.enum(['active', 'revoked']),
       rate_limit: z.number().optional(), // requests per second
       ip_allowlist: z.array(z.string()).optional(), // ['1.2.3.4/32']
     })

     export type APIKey = z.infer<typeof APIKeySchema>
     ```

  2. Create Firestore client (lib/firestore/client.ts):
     ```typescript
     import { initializeApp, getApps, cert } from 'firebase-admin/app'
     import { getFirestore } from 'firebase-admin/firestore'

     if (!getApps().length) {
       initializeApp({
         credential: cert({
           projectId: process.env.FIREBASE_PROJECT_ID,
           clientEmail: process.env.FIREBASE_CLIENT_EMAIL,
           privateKey: process.env.FIREBASE_PRIVATE_KEY?.replace(/\\n/g, '\n'),
         }),
       })
     }

     export const db = getFirestore()
     ```

  3. Create API keys service (lib/firestore/api-keys.ts):
     ```typescript
     import { db } from './client'
     import { APIKey, APIKeySchema } from './schema'
     import { randomBytes } from 'crypto'

     const API_KEYS_COLLECTION = 'api_keys'

     export async function createAPIKey(
       userId: string,
       data: {
         name: string
         scopes: string[]
         rate_limit?: number
         ip_allowlist?: string[]
       }
     ): Promise<APIKey> {
       const keyId = `apx_${randomBytes(24).toString('hex')}`

       const apiKey: APIKey = {
         id: keyId,
         user_id: userId,
         name: data.name,
         scopes: data.scopes,
         created_at: new Date().toISOString(),
         last_used_at: null,
         status: 'active',
         rate_limit: data.rate_limit,
         ip_allowlist: data.ip_allowlist,
       }

       await db.collection(API_KEYS_COLLECTION).doc(keyId).set(apiKey)

       return apiKey
     }

     export async function listAPIKeys(userId: string): Promise<APIKey[]> {
       const snapshot = await db
         .collection(API_KEYS_COLLECTION)
         .where('user_id', '==', userId)
         .where('status', '==', 'active')
         .get()

       return snapshot.docs.map((doc) => APIKeySchema.parse(doc.data()))
     }

     export async function getAPIKey(keyId: string): Promise<APIKey | null> {
       const doc = await db.collection(API_KEYS_COLLECTION).doc(keyId).get()

       if (!doc.exists) {
         return null
       }

       return APIKeySchema.parse(doc.data())
     }

     export async function revokeAPIKey(keyId: string, userId: string): Promise<void> {
       const key = await getAPIKey(keyId)

       if (!key || key.user_id !== userId) {
         throw new Error('Unauthorized')
       }

       await db.collection(API_KEYS_COLLECTION).doc(keyId).update({
         status: 'revoked',
       })
     }
     ```

  4. Create API routes (app/api/keys/route.ts):
     ```typescript
     import { NextRequest, NextResponse } from 'next/server'
     import { getServerSession } from 'next-auth'
     import { authOptions } from '@/app/api/auth/[...nextauth]/route'
     import { createAPIKey, listAPIKeys } from '@/lib/firestore/api-keys'
     import { z } from 'zod'

     const CreateKeySchema = z.object({
       name: z.string().min(1).max(100),
       scopes: z.array(z.string()).min(1),
       rate_limit: z.number().optional(),
       ip_allowlist: z.array(z.string()).optional(),
     })

     // GET /api/keys - List all keys
     export async function GET() {
       const session = await getServerSession(authOptions)

       if (!session?.user?.id) {
         return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
       }

       try {
         const keys = await listAPIKeys(session.user.id)
         return NextResponse.json(keys)
       } catch (error) {
         console.error('List keys error:', error)
         return NextResponse.json(
           { error: 'Failed to list keys' },
           { status: 500 }
         )
       }
     }

     // POST /api/keys - Create new key
     export async function POST(req: NextRequest) {
       const session = await getServerSession(authOptions)

       if (!session?.user?.id) {
         return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
       }

       try {
         const body = await req.json()
         const data = CreateKeySchema.parse(body)

         const key = await createAPIKey(session.user.id, data)

         return NextResponse.json(key, { status: 201 })
       } catch (error) {
         console.error('Create key error:', error)
         return NextResponse.json(
           { error: 'Failed to create key' },
           { status: 500 }
         )
       }
     }
     ```

  5. Create revoke endpoint (app/api/keys/[keyId]/route.ts):
     ```typescript
     import { NextRequest, NextResponse } from 'next/server'
     import { getServerSession } from 'next-auth'
     import { authOptions } from '@/app/api/auth/[...nextauth]/route'
     import { revokeAPIKey } from '@/lib/firestore/api-keys'

     // DELETE /api/keys/:keyId - Revoke key
     export async function DELETE(
       req: NextRequest,
       { params }: { params: { keyId: string } }
     ) {
       const session = await getServerSession(authOptions)

       if (!session?.user?.id) {
         return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
       }

       try {
         await revokeAPIKey(params.keyId, session.user.id)
         return NextResponse.json({ success: true })
       } catch (error) {
         console.error('Revoke key error:', error)
         return NextResponse.json(
           { error: 'Failed to revoke key' },
           { status: 500 }
         )
       }
     }
     ```

  6. Test API endpoints:
     ```bash
     # Create key
     curl -X POST http://localhost:3000/api/keys \
       -H "Content-Type: application/json" \
       -H "Cookie: next-auth.session-token=..." \
       -d '{
         "name": "Test Key",
         "scopes": ["product:payments"],
         "rate_limit": 100
       }'

     # List keys
     curl http://localhost:3000/api/keys \
       -H "Cookie: next-auth.session-token=..."

     # Revoke key
     curl -X DELETE http://localhost:3000/api/keys/apx_abc123 \
       -H "Cookie: next-auth.session-token=..."
     ```

Acceptance Criteria:
  - [ ] Create API key saves to Firestore
  - [ ] List API keys shows only user's keys
  - [ ] Revoke API key marks status as 'revoked'
  - [ ] Key IDs are cryptographically random (apx_...)
  - [ ] Validation with Zod prevents invalid data
  - [ ] Unauthorized users cannot access others' keys

Artifacts:
  - portal/lib/firestore/schema.ts: Firestore schemas
  - portal/lib/firestore/client.ts: Firestore init
  - portal/lib/firestore/api-keys.ts: CRUD service
  - portal/app/api/keys/route.ts: GET/POST endpoints
  - portal/app/api/keys/[keyId]/route.ts: DELETE endpoint

Backend Integration:
  - Firestore: api_keys collection
  - Same database as APX Router uses for key validation
  - Data flow: Portal creates key → Firestore → Router validates
  - Keys are immediately usable by APX Router (no delay)

Rollback:
  - git checkout -- portal/lib/firestore portal/app/api/keys
```

---

## Milestone 2: Analytics & Observability (Weeks 7-10)

**Status:** NOT_STARTED
**Duration:** 4 weeks
**Goal:** Usage charts, request explorer, policy viewer, SLO dashboard

*(Additional 15+ tasks with similar structure...)*

---

## Milestone 3: Pro Features (Weeks 11-14)

**Status:** NOT_STARTED
**Duration:** 4 weeks
**Goal:** Stripe billing, webhooks, RBAC, policy diffs

*(Additional 20+ tasks...)*

---

## Milestone 4: Copilot & Enterprise (Weeks 15-18)

**Status:** NOT_STARTED
**Duration:** 4 weeks
**Goal:** AI copilot, SAML SSO, custom domains

*(Additional 15+ tasks...)*

---

## Summary of Backend Integration Points

### APX Router
- **Health Check:** `GET /health` - System status
- **API Proxy:** All user requests proxied through `/api/proxy`
- **Request Tracing:** `x-apx-request-id` header propagation

### APX Edge
- **Request Logs:** Tail recent requests for dashboard
- **Latency Data:** p50/p95/p99 metrics

### BigQuery
- **Usage Analytics:** Requests, latency, errors by user/key
- **Request Explorer:** Search by ID, tenant, key, date range
- **Cost Analytics:** Per-user/per-product usage for billing

### Firestore
- **API Keys:** CRUD, validation, scopes
- **Users/Orgs:** Auth sessions, team management
- **Policies:** PolicyBundle versions, rollout state

### Pub/Sub
- **Webhooks:** Delivery logs, retries, DLQ
- **Real-time Updates:** Live dashboard updates

### Control Plane (Future)
- **Policy Management:** Create/update PolicyBundles
- **Deployments:** Canary rollouts, rollback

---

## Agent Responsibilities Matrix

| Agent Type | Focus | Key Integrations |
|------------|-------|------------------|
| frontend | UI components, pages | Next.js, Tailwind, shadcn/ui |
| backend | API routes, auth, data | NextAuth, Firestore, BigQuery |
| integration | APX services | Router, Edge, Pub/Sub |
| docs | Content, guides | MDX, OpenAPI |
| testing | E2E, a11y, perf | Playwright, Axe, Lighthouse |

---

## Next Steps

1. **Human Coordinator:** Assign PM0-T1-001 to frontend agent
2. **Frontend Agent:** Initialize Next.js portal
3. **Backend Agent:** Set up Firebase/Auth0 (PM0-T2-002)
4. **Integration Agent:** Test APX Router health check (PM0-T2-001)

---

**Document Version:** 1.0
**Last Updated:** 2025-11-11
**Status:** Ready for Execution
**Maintainers:** Portal Team
