# Pre-Deployment Checklist

**Portal Version:** M2 + M3
**Target:** Production
**Date:** 2025-11-12

---

## ⚠️ CRITICAL - Must Complete Before Deploy

### **1. Re-Enable Authentication** 🔒

**Status:** ❌ **NOT DONE - REQUIRED**

**Files to Update:**

#### A. Middleware (CRITICAL)
```bash
File: /Users/agentsy/APILEE/.private/portal/middleware.ts
Action: Uncomment authentication lines
```

**Current:** Authentication DISABLED ⚠️
**Required:** Authentication ENABLED ✅

#### B. API Keys Page
```bash
File: /Users/agentsy/APILEE/.private/portal/app/dashboard/api-keys/page.tsx
Action: Remove test user fallback, re-enable auth redirect
```

---

### **2. Environment Variables** 🔑

**Required (Must Have):**
- [ ] NEXTAUTH_URL (production domain)
- [ ] NEXTAUTH_SECRET (generate: `openssl rand -base64 32`)
- [ ] GOOGLE_CLIENT_ID
- [ ] GOOGLE_CLIENT_SECRET

**Optional (Recommended):**
- [ ] FIRESTORE_* (for persistent data)
- [ ] STRIPE_* (for billing features)
- [ ] SENDGRID_API_KEY (for emails)
- [ ] REDIS_URL (for caching/rate limiting)

---

### **3. OAuth Configuration** 🔐

**Google OAuth Console:**
- [ ] Add production callback URL: `https://your-domain.com/api/auth/callback/google`
- [ ] Add authorized JavaScript origins: `https://your-domain.com`
- [ ] Add authorized redirect URIs

---

### **4. Build Verification** ✅

Run these commands:

```bash
cd /Users/agentsy/APILEE/.private/portal

# 1. TypeScript check
npx tsc --noEmit

# 2. Production build
npm run build

# 3. Verify output
# Should see: ✓ Compiled successfully
```

**Expected:**
- ✅ 0 TypeScript errors (production)
- ✅ Build succeeds
- ✅ 49 pages generated
- ✅ 46 API routes

---

### **5. Security Review** 🔒

- [ ] Authentication re-enabled (middleware.ts)
- [ ] No hardcoded secrets in code
- [ ] .env.local NOT committed to git
- [ ] API routes have auth checks
- [ ] HTTPS enforced (auto with Vercel/Cloud Run)

---

## ✅ Ready to Deploy When

- [x] M2 complete (7 pages, 11 endpoints)
- [x] M3 complete (5 pages, 31 endpoints)
- [x] Build succeeds
- [x] Testing completed
- [ ] **Authentication re-enabled** ⚠️
- [ ] Environment variables configured
- [ ] OAuth credentials ready
- [ ] Domain/hosting prepared

---

## 🚀 Deployment Commands

### **Vercel (Recommended):**
```bash
# After completing above checklist
vercel --prod
```

### **Cloud Run:**
```bash
# Build image
gcloud builds submit --tag gcr.io/apx-build-478003/portal

# Deploy
gcloud run deploy portal \
  --image gcr.io/apx-build-478003/portal \
  --platform managed \
  --region us-central1
```

---

## 📋 Post-Deployment Verification

After deployment, test:

1. **Authentication:** Visit `/dashboard` → should redirect to sign-in
2. **OAuth:** Sign in with Google → should succeed
3. **Protected Routes:** Try `/api/keys` without auth → should return 401
4. **M2 Pages:** Visit analytics, SLO, health, etc. → all load
5. **M3 Pages:** Visit billing, webhooks, team, audit → all load

---

## ⚠️ Do NOT Deploy Until

1. ❌ Authentication is re-enabled
2. ❌ Environment variables are configured
3. ❌ OAuth is properly set up
4. ❌ Production domain is ready

**Current Status:** NOT READY - Auth still disabled for testing

---

**Next Step:** Complete items in "CRITICAL" section above
