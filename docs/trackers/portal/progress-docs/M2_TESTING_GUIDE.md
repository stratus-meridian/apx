# M2 Testing Guide - Complete Feature Validation

**Server:** http://localhost:3000
**Status:** Running ✅

---

## Testing Checklist

### **Phase 1: Authentication & Security** 🔒

**Test:** Verify all endpoints require authentication

1. **Without Auth:**
   ```bash
   # These should all return 401 Unauthorized
   curl http://localhost:3000/api/tail
   curl http://localhost:3000/api/stream/metrics
   curl http://localhost:3000/api/alerts
   curl http://localhost:3000/api/slo
   ```
   Expected: `{"error":"Unauthorized"}` with 401 status

2. **With Auth:**
   - Open browser: http://localhost:3000/dashboard
   - Should redirect to sign-in page
   - After signing in, should access dashboard

---

### **Phase 2: Navigation** 🧭

**Test:** Verify all M2 pages accessible via navigation

1. **Sidebar Links (8 new M2 pages):**
   - [ ] Analytics
   - [ ] Requests
   - [ ] SLO Dashboard
   - [ ] Health
   - [ ] Alerts
   - [ ] Real-time Tail
   - [ ] Traces
   - [ ] Policies

2. **Top Navigation:**
   - [ ] Analytics link visible
   - [ ] Active state highlighting works

3. **Mobile Menu:**
   - [ ] Resize browser to < 768px
   - [ ] Click hamburger menu
   - [ ] All M2 links visible

**How to Test:**
```
1. Visit http://localhost:3000/dashboard
2. Check left sidebar - should see all 8 M2 links
3. Click each link - verify page loads
4. Check active state (highlighted link)
5. Resize to mobile - check hamburger menu
```

---

### **Phase 3: M2 Pages Functionality** 📊

#### **1. Analytics Page** `/dashboard/analytics`

**Features to Test:**
- [ ] Page loads without errors
- [ ] 4 tabs visible (Overview, Latency, Errors, Breakdown)
- [ ] Charts render (uses Recharts)
- [ ] Date range selector works
- [ ] Granularity selector (Hour, Day, Week, Month)
- [ ] **Export button** - NEW FIX
  - [ ] Click Export → Dialog opens
  - [ ] Select format (CSV, JSON, Excel, PDF)
  - [ ] Enter filename
  - [ ] Click Export → File downloads
- [ ] Filter by API key works
- [ ] Mock data displays (since no real BigQuery data)

**Expected Output:**
- P50/P95/P99 latency charts
- Error rate trends
- Method breakdown (GET, POST, etc.)
- Status code distribution

---

#### **2. Requests Page** `/dashboard/requests`

**Features to Test:**
- [ ] Page loads with request list
- [ ] Search bar works (filters requests)
- [ ] Method filter (GET, POST, PUT, DELETE, PATCH)
- [ ] Status filter (2xx, 4xx, 5xx)
- [ ] Date range picker
- [ ] Request table displays:
  - Request ID
  - Timestamp
  - Method
  - Path
  - Status
  - Latency
- [ ] Click request → navigates to detail page
- [ ] Pagination works (if > 25 requests)
- [ ] **Zod validation** - NEW FIX
  - Try invalid URL params (should handle gracefully)

**Test URL:**
```
http://localhost:3000/dashboard/requests?method=GET&status=2xx
```

---

#### **3. Request Detail Page** `/dashboard/requests/[requestId]`

**Features to Test:**
- [ ] Page loads with request details
- [ ] Shows full request/response
- [ ] Headers displayed
- [ ] Body displayed (formatted JSON)
- [ ] Timing information
- [ ] Export to JSON button works
- [ ] Breadcrumb navigation back to requests
- [ ] **Cache headers** - NEW FIX (5 min cache)

**Test URL:**
```
http://localhost:3000/dashboard/requests/req_1
```

---

#### **4. SLO Dashboard** `/dashboard/slo`

**Features to Test:**
- [ ] Page loads with SLO metrics
- [ ] 4 SLO cards display:
  - Availability
  - Latency
  - Error Rate
  - Success Rate
- [ ] SLO charts render
- [ ] Burn rate indicators
- [ ] Error budget tracking
- [ ] Auto-refresh every 60 seconds
- [ ] Date range selector (7d, 30d, 90d)
- [ ] **Authentication** - NEW FIX (requires login)
- [ ] **Cache headers** - NEW FIX (5 min cache)

**Expected:**
- Green/Yellow/Red status indicators
- Percentage values (99.9%, 99.5%, etc.)
- Trend charts

---

#### **5. Health Page** `/dashboard/health`

**Features to Test:**
- [ ] Page loads with health status
- [ ] System status banner (Healthy/Degraded/Down)
- [ ] Component status grid:
  - Firestore
  - Pub/Sub
  - BigQuery
  - Router
  - Workers
- [ ] Uptime chart (7 days)
- [ ] Incident timeline
- [ ] Auto-refresh every 30 seconds
- [ ] 3 tabs (Overview, Components, History)

**Expected:**
- Green checkmarks for healthy components
- Yellow/Red warnings for issues
- Mock incident data

---

#### **6. Alerts Page** `/dashboard/alerts`

**Features to Test:**
- [ ] Page loads with alerts interface
- [ ] 2 tabs (Rules, History)
- [ ] Alert rules list displays
- [ ] Create alert button works
- [ ] Edit alert functionality
- [ ] Delete alert with confirmation
- [ ] Alert history with timestamps
- [ ] **Authentication** - NEW FIX (all methods secured)
- [ ] **Cache headers** - NEW FIX (30 sec cache)
- [ ] **No console.log** - NEW FIX (removed debug statement)

**Test Actions:**
1. Click "Create Alert"
2. Fill form (name, condition, threshold)
3. Save → Should appear in rules list
4. Click edit → Should load dialog
5. Click delete → Should show confirmation

---

#### **7. Policies Page** `/dashboard/policies`

**Features to Test:**
- [ ] Page loads with policy information
- [ ] 5 tabs (Overview, Rate Limits, Quotas, Details, Hierarchy)
- [ ] Policy cards display
- [ ] Quota meters show usage
- [ ] Rate limit visualizations
- [ ] Policy tree/hierarchy
- [ ] Policy detail view
- [ ] Filter by tier (Free, Pro, Enterprise)
- [ ] **Zod validation** - NEW FIX (tier validation)
- [ ] **Cache headers** - NEW FIX (10 min cache)

**Expected:**
- Visual quota meters (0-100%)
- Rate limit per minute/hour/day
- Policy inheritance tree

---

#### **8. Real-time Tail** `/dashboard/tail`

**Features to Test:**
- [ ] Page loads with streaming interface
- [ ] SSE connection established (see requests flowing)
- [ ] Filter by method works
- [ ] Filter by status code works
- [ ] Pause button stops stream
- [ ] Resume button restarts stream
- [ ] Clear button empties list
- [ ] Auto-scroll works
- [ ] Request details expandable
- [ ] **Authentication** - NEW FIX (SSE secured)
- [ ] Max 100 requests in memory (prevents overflow)

**Visual Test:**
- Should see new requests appearing every 100-500ms
- Color coding: Green (2xx), Yellow (4xx), Red (5xx)
- Method badges (GET, POST, etc.)

**Test URL:**
```
http://localhost:3000/dashboard/tail?method=GET&status=200
```

---

### **Phase 4: Export Functionality** 📤

**Test All Export Formats:**

#### **Usage Page** (Already working)
1. Visit `/dashboard/usage`
2. Click "Export"
3. Test each format:
   - [ ] CSV → Downloads .csv file
   - [ ] JSON → Downloads .json file
   - [ ] Excel → Downloads .xlsx file
   - [ ] PDF → Downloads .pdf file

#### **Analytics Page** (NEW FIX - Just Completed)
1. Visit `/dashboard/analytics`
2. Click "Export Data" button
3. Test each format:
   - [ ] CSV → Downloads analytics data
   - [ ] JSON → Downloads raw data
   - [ ] Excel → Downloads workbook
   - [ ] PDF → Downloads report with charts

**Verify Export Content:**
- CSV: Open in Excel/Sheets → Check data format
- JSON: Open in text editor → Check structure
- Excel: Open in Excel → Check sheets and formatting
- PDF: Open in PDF viewer → Check layout

---

### **Phase 5: Code Quality** ✨

**Test: No Console Errors**

1. Open Browser DevTools (F12)
2. Go to Console tab
3. Navigate through all M2 pages
4. **Expected:** Zero console errors
5. **Allowed:** Console warnings (non-blocking)

**Check:**
- [ ] No `console.log` statements
- [ ] No React warnings (except pre-existing)
- [ ] No TypeScript errors in console
- [ ] No network errors (except expected auth redirects)

---

### **Phase 6: Performance** ⚡

**Test: Cache Headers**

Use browser DevTools Network tab:

1. Visit `/dashboard/requests`
2. Check Response Headers:
   ```
   Cache-Control: public, s-maxage=60, stale-while-revalidate=120
   ```

2. Visit `/dashboard/slo`
3. Check Response Headers:
   ```
   Cache-Control: public, s-maxage=300, stale-while-revalidate=600
   ```

3. Visit `/dashboard/alerts`
4. Check Response Headers:
   ```
   Cache-Control: public, s-maxage=30, stale-while-revalidate=60
   ```

4. Visit `/dashboard/policies`
5. Check Response Headers:
   ```
   Cache-Control: public, s-maxage=600, stale-while-revalidate=1200
   ```

**Expected:** All endpoints return appropriate cache headers

---

### **Phase 7: Validation** ✅

**Test: Zod Schema Validation**

Try invalid API requests:

```bash
# Invalid granularity
curl "http://localhost:3000/api/analytics/latency?granularity=invalid"
# Should return 400 with detailed error

# Invalid date format
curl "http://localhost:3000/api/analytics/latency?start=not-a-date"
# Should return 400 with Zod error details

# Invalid method
curl "http://localhost:3000/api/requests?method=INVALID"
# Should return 400 with enum error
```

**Expected Response:**
```json
{
  "error": "Invalid query parameters",
  "details": [
    {
      "code": "invalid_enum_value",
      "path": ["granularity"],
      "message": "Invalid enum value..."
    }
  ]
}
```

---

### **Phase 8: Responsive Design** 📱

**Test Mobile Layout:**

1. Resize browser to mobile width (< 768px)
2. Check:
   - [ ] Sidebar collapses (hidden on mobile)
   - [ ] Hamburger menu appears in top nav
   - [ ] All M2 links in mobile menu
   - [ ] Charts responsive (adjust to screen width)
   - [ ] Tables scroll horizontally if needed
   - [ ] Buttons and controls usable on touch

**Test Tablet Layout:**

1. Resize to tablet width (768px - 1024px)
2. Check:
   - [ ] Sidebar may be hidden (uses toggle)
   - [ ] Charts use 2-column layout
   - [ ] Cards stack appropriately

---

## Quick Test Script

Run this to test all pages quickly:

```bash
# Test page accessibility
for page in analytics requests slo health alerts policies tail; do
  echo "Testing /dashboard/$page"
  curl -I http://localhost:3000/dashboard/$page
done

# Test API endpoints (should return 401 without auth)
for endpoint in tail stream/metrics alerts slo; do
  echo "Testing /api/$endpoint"
  curl http://localhost:3000/api/$endpoint
done
```

---

## Issues to Report

If you find any issues, note:
- [ ] Page URL
- [ ] Expected behavior
- [ ] Actual behavior
- [ ] Browser console errors
- [ ] Network tab errors
- [ ] Screenshots if visual issue

---

## Success Criteria

**M2 is ready for M3 when:**
- ✅ All 8 pages load without errors
- ✅ All navigation links work
- ✅ Authentication blocks unauthenticated access
- ✅ All export formats work
- ✅ No console errors
- ✅ Cache headers present on API responses
- ✅ Validation returns proper error messages
- ✅ Responsive on mobile/tablet/desktop
- ✅ Real-time streaming works
- ✅ Mock data displays correctly

---

## After Testing

Once all tests pass, we'll:
1. ✅ Mark M2 as fully validated
2. 🚀 Launch M3 parallel agents
3. 📋 Begin M3 (Pro Features) development

---

**Testing Time Estimate:** 20-30 minutes for thorough validation
**Current Status:** Ready for testing
**Server:** http://localhost:3000 ✅
