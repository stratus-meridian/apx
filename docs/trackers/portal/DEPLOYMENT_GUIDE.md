# APX Portal - Production Deployment Guide

**Version:** M2 + M3 (Analytics & Pro Features)
**Status:** Production Ready ✅
**Last Updated:** 2025-11-12

---

## 🎯 Overview

This guide covers deploying the APX Portal with:
- **M0:** Foundation (Auth, Navigation, Testing)
- **M1:** Core Portal (Dashboard, Products, API Keys, Organizations, Usage)
- **M2:** Analytics & Observability (7 pages, 11 endpoints)
- **M3:** Pro Features (5 pages, 31 endpoints)

**Total:** 18 dashboard pages, 50+ API endpoints, enterprise-grade features

---

## ✅ Pre-Deployment Checklist

### **1. Code Preparation**

- [x] M2 complete with all fixes applied
- [x] M3 complete with testing validated
- [x] Build succeeds (`npm run build`)
- [x] TypeScript: 0 production errors
- [ ] **Re-enable authentication** (see step 2)
- [ ] Environment variables configured
- [ ] Database connections tested (if using real backend)

### **2. Re-Enable Authentication** ⚠️ CRITICAL

**Current Status:** Authentication is DISABLED for testing

**Files to Update:**

**A. Middleware (Required)**
```bash
File: /Users/agentsy/APILEE/.private/portal/middleware.ts
```

**CHANGE FROM:**
```typescript
// ⚠️ AUTHENTICATION TEMPORARILY DISABLED FOR M2 TESTING
// ⚠️ RE-ENABLE BEFORE PRODUCTION DEPLOYMENT!
// export { default } from 'next-auth/middleware'

export const config = {
  matcher: [
    // TEMPORARILY DISABLED FOR TESTING
    // '/dashboard/:path*',
    // '/api/:path((?!auth).*)',
  ],
}
```

**CHANGE TO:**
```typescript
export { default } from 'next-auth/middleware'

// Protect all routes that match these patterns
export const config = {
  matcher: [
    // Protect dashboard routes
    '/dashboard/:path*',
    // Protect API routes (except auth routes)
    '/api/:path((?!auth).*)',
  ],
}
```

**B. API Keys Page (Required)**
```bash
File: /Users/agentsy/APILEE/.private/portal/app/dashboard/api-keys/page.tsx
```

**CHANGE FROM:**
```typescript
// ⚠️ TEMPORARILY DISABLED FOR M2 TESTING
// if (!session?.user?.id) {
//   redirect('/auth/signin')
// }

// Use test user ID when no session (for testing only)
const userId = session?.user?.id || 'test-user-123'
```

**CHANGE TO:**
```typescript
if (!session?.user?.id) {
  redirect('/auth/signin')
}

const userId = session.user.id
```

### **3. Environment Variables**

Create `.env.local` file with:

```bash
# NextAuth Configuration
NEXTAUTH_URL=https://your-domain.com
NEXTAUTH_SECRET=<generate-with-openssl-rand-base64-32>

# Google OAuth (Required for authentication)
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret

# Database (Optional - falls back to mock data)
FIRESTORE_PROJECT_ID=your_project_id
FIRESTORE_PRIVATE_KEY=your_private_key
FIRESTORE_CLIENT_EMAIL=your_client_email

# BigQuery (Optional - for analytics)
BIGQUERY_PROJECT_ID=your_project_id
BIGQUERY_DATASET=apx_analytics

# Stripe (Optional - for billing)
STRIPE_SECRET_KEY=sk_live_...
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_ID_PRO_MONTHLY=price_...
STRIPE_PRICE_ID_PRO_YEARLY=price_...
STRIPE_PRICE_ID_ENTERPRISE_MONTHLY=price_...
STRIPE_PRICE_ID_ENTERPRISE_YEARLY=price_...

# Redis (Optional - for rate limiting)
REDIS_URL=redis://your-redis-url

# Email (Optional - for team invitations)
SENDGRID_API_KEY=SG...
FROM_EMAIL=noreply@your-domain.com

# OpenTelemetry (Optional - for monitoring)
OTEL_EXPORTER_OTLP_ENDPOINT=your-otel-endpoint
```

**Required for Production:**
- NEXTAUTH_URL
- NEXTAUTH_SECRET
- GOOGLE_CLIENT_ID
- GOOGLE_CLIENT_SECRET

**Optional but Recommended:**
- FIRESTORE_* (for persistent data)
- STRIPE_* (for billing features)
- SENDGRID_API_KEY (for email invitations)

---

## 🚀 Deployment Options

### **Option 1: Vercel (Recommended)**

**Why Vercel:**
- ✅ Native Next.js support
- ✅ Automatic HTTPS
- ✅ Built-in CDN
- ✅ Serverless functions
- ✅ Easy environment variable management
- ✅ Preview deployments

**Steps:**

1. **Install Vercel CLI**
   ```bash
   npm install -g vercel
   ```

2. **Login to Vercel**
   ```bash
   vercel login
   ```

3. **Deploy**
   ```bash
   cd /Users/agentsy/APILEE/.private/portal
   vercel --prod
   ```

4. **Configure Environment Variables**
   - Go to Vercel Dashboard → Project Settings → Environment Variables
   - Add all required variables from `.env.local`
   - Redeploy after adding variables

5. **Set up Custom Domain** (Optional)
   - Vercel Dashboard → Domains
   - Add your domain (e.g., portal.apx.com)
   - Update DNS records as instructed

**Vercel Configuration:**
```json
// vercel.json (create if needed)
{
  "buildCommand": "npm run build",
  "devCommand": "npm run dev",
  "installCommand": "npm install",
  "framework": "nextjs",
  "regions": ["iad1"]
}
```

---

### **Option 2: Google Cloud Run**

**Why Cloud Run:**
- ✅ Integrates with existing GCP infrastructure
- ✅ Serverless containerized deployment
- ✅ Auto-scaling
- ✅ Pay per use

**Steps:**

1. **Create Dockerfile** (if not exists)
   ```dockerfile
   FROM node:20-alpine AS base

   # Install dependencies only when needed
   FROM base AS deps
   WORKDIR /app
   COPY package*.json ./
   RUN npm ci

   # Build the app
   FROM base AS builder
   WORKDIR /app
   COPY --from=deps /app/node_modules ./node_modules
   COPY . .
   RUN npm run build

   # Production image
   FROM base AS runner
   WORKDIR /app
   ENV NODE_ENV=production

   RUN addgroup --system --gid 1001 nodejs
   RUN adduser --system --uid 1001 nextjs

   COPY --from=builder /app/public ./public
   COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
   COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

   USER nextjs
   EXPOSE 3000
   ENV PORT=3000

   CMD ["node", "server.js"]
   ```

2. **Update next.config.js**
   ```javascript
   module.exports = {
     output: 'standalone',
     // ... other config
   }
   ```

3. **Build and Push Image**
   ```bash
   gcloud builds submit --tag gcr.io/apx-build-478003/portal
   ```

4. **Deploy to Cloud Run**
   ```bash
   gcloud run deploy portal \
     --image gcr.io/apx-build-478003/portal \
     --platform managed \
     --region us-central1 \
     --allow-unauthenticated \
     --set-env-vars="NEXTAUTH_URL=https://portal.apx.com"
   ```

5. **Set Environment Variables**
   ```bash
   gcloud run services update portal \
     --update-env-vars=NEXTAUTH_SECRET=xxx,GOOGLE_CLIENT_ID=xxx
   ```

---

### **Option 3: Docker + Cloud Run (via Terraform)**

**Using existing Terraform setup:**

1. **Add Portal to terraform/cloudrun.tf**
   ```hcl
   resource "google_cloud_run_service" "portal" {
     name     = "portal"
     location = var.region

     template {
       spec {
         containers {
           image = "gcr.io/${var.project_id}/portal:latest"

           env {
             name  = "NEXTAUTH_URL"
             value = "https://portal.apx.com"
           }

           env {
             name = "NEXTAUTH_SECRET"
             value_from {
               secret_key_ref {
                 name = google_secret_manager_secret.nextauth_secret.secret_id
                 key  = "latest"
               }
             }
           }
         }
       }
     }
   }
   ```

2. **Apply Terraform**
   ```bash
   cd /Users/agentsy/APILEE/infra/terraform
   terraform plan
   terraform apply
   ```

---

## 🔒 Security Checklist

### **Before Deployment:**

- [ ] Authentication re-enabled (middleware.ts)
- [ ] OAuth credentials configured (Google/GitHub)
- [ ] NEXTAUTH_SECRET is strong (32+ characters)
- [ ] API routes protected with authentication
- [ ] Environment variables set (not committed to git)
- [ ] HTTPS enforced (automatic with Vercel/Cloud Run)
- [ ] CORS configured if needed
- [ ] Rate limiting enabled (optional)

### **After Deployment:**

- [ ] Test authentication flow
- [ ] Verify all protected routes require login
- [ ] Check API endpoints return 401 without auth
- [ ] Test OAuth callback URLs
- [ ] Verify Stripe webhook signatures (if using billing)
- [ ] Monitor error logs
- [ ] Set up alerts for failures

---

## 🧪 Post-Deployment Testing

### **1. Authentication Test**
```bash
# Should redirect to sign-in
curl -I https://your-domain.com/dashboard

# Should return 401
curl https://your-domain.com/api/keys
```

### **2. Page Load Test**
Visit each page and verify:
- [ ] https://your-domain.com/dashboard
- [ ] https://your-domain.com/dashboard/analytics
- [ ] https://your-domain.com/dashboard/requests
- [ ] https://your-domain.com/dashboard/slo
- [ ] https://your-domain.com/dashboard/health
- [ ] https://your-domain.com/dashboard/alerts
- [ ] https://your-domain.com/dashboard/policies
- [ ] https://your-domain.com/dashboard/tail
- [ ] https://your-domain.com/dashboard/billing
- [ ] https://your-domain.com/dashboard/webhooks
- [ ] https://your-domain.com/dashboard/team
- [ ] https://your-domain.com/dashboard/audit

### **3. Feature Test**
- [ ] Sign in with Google OAuth
- [ ] Create API key
- [ ] View analytics charts
- [ ] Create webhook
- [ ] Invite team member
- [ ] View audit logs

### **4. Performance Test**
```bash
# Lighthouse score
npx lighthouse https://your-domain.com/dashboard --view

# Load time check
curl -w "@-" -o /dev/null -s https://your-domain.com <<'EOF'
time_namelookup:  %{time_namelookup}\n
time_connect:  %{time_connect}\n
time_appconnect:  %{time_appconnect}\n
time_total:  %{time_total}\n
EOF
```

---

## 📊 Monitoring & Observability

### **Recommended Setup:**

1. **Error Tracking**
   - Sentry (recommended)
   - Or Vercel Analytics
   - Or Google Cloud Logging

2. **Performance Monitoring**
   - Vercel Analytics
   - Or Google Cloud Monitoring
   - Or New Relic

3. **Uptime Monitoring**
   - UptimeRobot
   - Or Pingdom
   - Or Google Cloud Uptime Checks

### **Key Metrics to Monitor:**

- Response times (< 200ms target)
- Error rates (< 1% target)
- Uptime (> 99.9% target)
- Authentication success rate
- API endpoint latency
- Build success rate

---

## 🔄 Rollback Plan

### **If Issues Occur:**

**Vercel:**
```bash
# List deployments
vercel ls

# Rollback to previous
vercel rollback [deployment-url]
```

**Cloud Run:**
```bash
# List revisions
gcloud run revisions list --service=portal

# Rollback to previous
gcloud run services update-traffic portal \
  --to-revisions=portal-001=100
```

---

## 📝 Post-Deployment Tasks

### **Week 1:**
- [ ] Monitor error logs daily
- [ ] Check authentication success rate
- [ ] Verify all features working
- [ ] Collect user feedback
- [ ] Address any critical issues

### **Week 2-4:**
- [ ] Optimize slow pages (if any)
- [ ] Add missing features (based on feedback)
- [ ] Implement rate limiting (if needed)
- [ ] Set up automated backups
- [ ] Create runbook for common issues

---

## 🚨 Common Issues & Solutions

### **Issue: OAuth Redirect Mismatch**
**Solution:** Update OAuth callback URLs in Google Console:
- Add: `https://your-domain.com/api/auth/callback/google`

### **Issue: 500 Error on API Routes**
**Solution:** Check environment variables are set correctly

### **Issue: Stripe Webhooks Failing**
**Solution:**
1. Update webhook URL in Stripe Dashboard
2. Verify STRIPE_WEBHOOK_SECRET is correct
3. Check signature verification in logs

### **Issue: Slow Page Load**
**Solution:**
1. Enable caching headers (already implemented)
2. Optimize bundle size (consider code splitting)
3. Use CDN for static assets

---

## 🎯 Success Criteria

**Deployment is successful when:**
- ✅ All pages load without errors
- ✅ Authentication works (Google OAuth)
- ✅ All M2 features accessible (analytics, SLO, health, etc.)
- ✅ All M3 features accessible (billing, webhooks, team, etc.)
- ✅ API endpoints return proper responses
- ✅ No console errors in production
- ✅ Response times < 500ms
- ✅ Uptime > 99%

---

## 📞 Support

**If deployment issues occur:**
1. Check deployment logs (Vercel/Cloud Run console)
2. Review error tracking dashboard (Sentry)
3. Check environment variables
4. Verify DNS configuration
5. Review this guide's troubleshooting section

---

## 🔗 Additional Resources

- [Next.js Deployment Docs](https://nextjs.org/docs/deployment)
- [Vercel Documentation](https://vercel.com/docs)
- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [NextAuth.js Documentation](https://next-auth.js.org)
- [Stripe Documentation](https://stripe.com/docs)

---

**Last Updated:** 2025-11-12
**Portal Version:** M2 + M3
**Status:** Ready for Production ✅
