# Milestone 2: Analytics & Observability - Execution Plan

**Status:** IN PROGRESS
**Duration:** 2-3 days (with parallel agents)
**Goal:** Advanced analytics, request explorer, policy viewer, SLO dashboard, real-time monitoring
**Started:** 2025-11-12

---

## Overview

Milestone 2 enhances the portal with production-grade analytics and observability features that help users understand their API usage, debug issues, and monitor service health in real-time.

### Key Features

1. **Enhanced Usage Analytics** - Deep dive into request patterns
2. **Request Explorer** - Search and filter individual requests
3. **Latency Analysis** - P50/P95/P99 percentile tracking
4. **Error Monitoring** - Error rates and types
5. **SLO Dashboard** - Service level objective tracking
6. **Real-Time Tail** - Live request streaming
7. **Policy Viewer** - Visualize effective policies
8. **Quota Meters** - Visual progress bars for limits
9. **Cost Analytics** - Usage-based cost tracking
10. **Advanced Exports** - CSV, JSON, API access

---

## Task Breakdown

### Phase PM2-T1: Enhanced Analytics (5 tasks)

#### PM2-T1-001: Advanced Usage Charts

**Goal:** Enhanced charts with latency percentiles, error rates, and cost analysis

**Tasks:**
- Add P50/P95/P99 latency charts
- Add error rate visualization
- Add request method breakdown (GET/POST/PUT/DELETE)
- Add status code distribution
- Add cost tracking chart (if billing enabled)

**Files to Create:**
- `/components/analytics/latency-chart.tsx` - Percentile line chart
- `/components/analytics/error-rate-chart.tsx` - Error tracking
- `/components/analytics/method-breakdown.tsx` - Request method pie chart
- `/components/analytics/status-distribution.tsx` - Status code bar chart
- `/components/analytics/cost-chart.tsx` - Usage cost over time

**API Endpoints:**
- `/api/analytics/latency` - Get latency percentiles
- `/api/analytics/errors` - Get error analysis
- `/api/analytics/breakdown` - Get request breakdowns

**Dependencies:** BigQuery with enhanced schema

---

#### PM2-T1-002: Enhanced Metrics Dashboard

**Goal:** Comprehensive metrics with comparisons and trends

**Tasks:**
- Add comparison view (current vs previous period)
- Add trend indicators (↑ ↓)
- Add metric sparklines
- Add peak time analysis
- Add geographic distribution (if available)

**Files to Modify:**
- `/components/usage/metrics-grid.tsx` - Add comparisons
- `/app/dashboard/usage/page.tsx` - Add comparison controls

**Files to Create:**
- `/components/analytics/metric-card.tsx` - Enhanced metric display
- `/components/analytics/trend-indicator.tsx` - Trend arrows
- `/components/analytics/sparkline.tsx` - Mini charts

---

#### PM2-T1-003: Time Range Presets & Custom Ranges

**Goal:** Flexible time range selection with calendar

**Tasks:**
- Add calendar date picker
- Add custom range selection
- Add more presets (Today, Yesterday, Last 7 days, Last 30 days, Last 90 days, This month, Last month)
- Add timezone selector
- Add "Compare to previous period" toggle

**Files to Create:**
- `/components/analytics/date-range-picker.tsx` - Calendar component
- `/components/analytics/time-zone-selector.tsx` - Timezone picker

**Files to Modify:**
- `/app/dashboard/usage/page.tsx` - Integrate new controls

**UI Library:** Add shadcn/ui calendar and popover components

---

#### PM2-T1-004: Export Enhancements

**Goal:** Multiple export formats and scheduled exports

**Tasks:**
- Add JSON export
- Add Excel export
- Add PDF report generation
- Add email report scheduling (optional)
- Add export templates

**Files to Create:**
- `/lib/exports/json-exporter.ts` - JSON export logic
- `/lib/exports/excel-exporter.ts` - Excel generation
- `/lib/exports/pdf-generator.ts` - PDF reports
- `/components/analytics/export-dialog.tsx` - Export UI

**Dependencies:** Install xlsx library for Excel, jsPDF for PDF

---

#### PM2-T1-005: Usage Analytics Page Redesign

**Goal:** Unified analytics experience with all new features

**Tasks:**
- Integrate all new charts
- Add tabbed interface (Overview, Latency, Errors, Costs)
- Add filters sidebar (API key, method, status code, endpoint)
- Add save/load filter presets
- Add shareable dashboard links

**Files to Modify:**
- `/app/dashboard/usage/page.tsx` - Major redesign

**Files to Create:**
- `/components/analytics/filters-sidebar.tsx` - Filter controls
- `/components/analytics/filter-presets.tsx` - Save/load filters
- `/app/dashboard/analytics/page.tsx` - New dedicated analytics page

---

### Phase PM2-T2: Request Explorer (4 tasks)

#### PM2-T2-001: Request Search & Filter

**Goal:** Search individual requests by multiple criteria

**Tasks:**
- Create request explorer page
- Add search by request ID
- Add filter by date range
- Add filter by API key
- Add filter by HTTP method
- Add filter by status code
- Add filter by endpoint path
- Add filter by tenant ID (if multi-tenant)

**Files to Create:**
- `/app/dashboard/requests/page.tsx` - Request explorer page
- `/components/requests/request-filters.tsx` - Filter UI
- `/components/requests/request-search.tsx` - Search bar
- `/lib/bigquery/requests.ts` - Request query functions
- `/app/api/requests/route.ts` - Request search API

**BigQuery Schema Required:**
```sql
CREATE TABLE api_requests (
  request_id STRING,
  timestamp TIMESTAMP,
  user_id STRING,
  api_key_id STRING,
  method STRING,
  path STRING,
  status_code INT64,
  latency_ms FLOAT64,
  request_size INT64,
  response_size INT64,
  tenant_id STRING,
  error_message STRING,
  headers STRING,
  query_params STRING
)
```

---

#### PM2-T2-002: Request Detail View

**Goal:** Full request/response inspection

**Tasks:**
- Create request detail page
- Display request metadata
- Display request headers
- Display request body
- Display response headers
- Display response body
- Display timing breakdown
- Add "Replay request" button

**Files to Create:**
- `/app/dashboard/requests/[requestId]/page.tsx` - Detail page
- `/components/requests/request-details.tsx` - Request metadata
- `/components/requests/request-body-viewer.tsx` - Body display
- `/components/requests/timing-waterfall.tsx` - Timing visualization

---

#### PM2-T2-003: Request List with Pagination

**Goal:** Efficient browsing of large request sets

**Tasks:**
- Add infinite scroll or pagination
- Add sorting (by timestamp, latency, status)
- Add bulk actions (export selected)
- Add request comparison view
- Add quick filters

**Files to Create:**
- `/components/requests/request-table.tsx` - Table with pagination
- `/components/requests/request-row.tsx` - Individual row
- `/components/requests/bulk-actions.tsx` - Bulk operations

**Files to Modify:**
- `/app/api/requests/route.ts` - Add pagination support

---

#### PM2-T2-004: Request Analytics

**Goal:** Analytics specific to request explorer

**Tasks:**
- Add request volume chart on explorer
- Add latency histogram
- Add error rate chart
- Add geographic distribution map (optional)
- Add common errors list

**Files to Create:**
- `/components/requests/request-histogram.tsx` - Latency histogram
- `/components/requests/error-summary.tsx` - Error aggregation

---

### Phase PM2-T3: SLO & Health Monitoring (3 tasks)

#### PM2-T3-001: SLO Dashboard

**Goal:** Track service level objectives

**Tasks:**
- Create SLO dashboard page
- Add SLO definitions (uptime, latency, error rate)
- Add SLO status indicators (green/yellow/red)
- Add SLO trend charts
- Add SLO burn rate calculation
- Add alerting thresholds

**Files to Create:**
- `/app/dashboard/slo/page.tsx` - SLO dashboard
- `/components/slo/slo-card.tsx` - Individual SLO display
- `/components/slo/slo-chart.tsx` - SLO trend
- `/components/slo/burn-rate.tsx` - Burn rate indicator
- `/lib/slo/calculator.ts` - SLO calculations
- `/app/api/slo/route.ts` - SLO metrics API

**SLO Examples:**
- Uptime: 99.9% (3 nines)
- P95 Latency: < 500ms
- Error Rate: < 1%
- Availability: 99.99% (4 nines)

---

#### PM2-T3-002: Enhanced System Health

**Goal:** Comprehensive health monitoring

**Tasks:**
- Enhance SystemStatus component
- Add historical uptime chart
- Add component health trends
- Add incident timeline
- Add maintenance windows
- Add health check history

**Files to Modify:**
- `/components/system-status.tsx` - Major enhancement

**Files to Create:**
- `/components/health/uptime-chart.tsx` - Historical uptime
- `/components/health/incident-timeline.tsx` - Incidents
- `/components/health/health-history.tsx` - Check history
- `/app/dashboard/health/page.tsx` - Dedicated health page

---

#### PM2-T3-003: Alerts & Notifications

**Goal:** Proactive issue detection

**Tasks:**
- Add alert configuration UI
- Add alert types (email, webhook, in-app)
- Add alert rules (thresholds, conditions)
- Add alert history
- Add notification preferences

**Files to Create:**
- `/app/dashboard/alerts/page.tsx` - Alerts management
- `/components/alerts/alert-rules.tsx` - Rule configuration
- `/components/alerts/alert-history.tsx` - Alert log
- `/lib/alerts/rules-engine.ts` - Alert logic
- `/app/api/alerts/route.ts` - Alerts API

---

### Phase PM2-T4: Real-Time Features (2 tasks)

#### PM2-T4-001: Real-Time Request Tail

**Goal:** Live streaming of API requests

**Tasks:**
- Implement Server-Sent Events (SSE) or WebSocket
- Create request tail page
- Add real-time request stream
- Add filters for live stream
- Add pause/resume controls
- Add auto-scroll
- Add request highlighting

**Files to Create:**
- `/app/dashboard/tail/page.tsx` - Request tail page
- `/components/tail/request-stream.tsx` - Live stream display
- `/components/tail/stream-controls.tsx` - Play/pause controls
- `/app/api/tail/route.ts` - SSE endpoint
- `/lib/streams/request-stream.ts` - Stream handling

**Backend Integration:** Pub/Sub subscription for real-time events

---

#### PM2-T4-002: Live Dashboard Updates

**Goal:** Real-time metrics updates

**Tasks:**
- Add WebSocket connection for live metrics
- Update dashboard stats in real-time
- Add "Live" indicator
- Add auto-refresh toggle
- Add update frequency selector

**Files to Modify:**
- `/app/dashboard/page.tsx` - Add WebSocket
- `/components/dashboard/stats-cards.tsx` - Real-time updates

**Files to Create:**
- `/lib/websocket/metrics-client.ts` - WebSocket client
- `/app/api/ws/metrics/route.ts` - WebSocket server

---

### Phase PM2-T5: Policy & Configuration Viewer (1 task)

#### PM2-T5-001: Policy Viewer

**Goal:** Visualize effective policies

**Tasks:**
- Create policy viewer page
- Display active PolicyBundle
- Show rate limits visually
- Show quota usage
- Show policy inheritance
- Add policy validation
- Add policy simulation (what-if analysis)

**Files to Create:**
- `/app/dashboard/policies/page.tsx` - Policy viewer
- `/components/policies/policy-card.tsx` - Policy display
- `/components/policies/rate-limit-visual.tsx` - Visual limits
- `/components/policies/quota-meter.tsx` - Quota progress
- `/components/policies/policy-tree.tsx` - Inheritance tree
- `/lib/policies/simulator.ts` - Policy simulation
- `/app/api/policies/route.ts` - Policies API

**Backend Integration:** Fetch PolicyBundles from Firestore

---

## Dependencies & Prerequisites

### New Dependencies to Install

```bash
# Analytics & Charts
npm install xlsx jspdf recharts-to-png

# Date/Time
npm install date-fns-tz

# Real-time
npm install ws @types/ws eventsource

# Export
npm install xlsx jspdf html2canvas

# UI Components (if not already installed)
npx shadcn-ui@latest add calendar popover
```

### BigQuery Schema Updates

Add these fields to `api_requests` table:
- `tenant_id` (STRING) - Multi-tenant support
- `error_message` (STRING) - Error details
- `headers` (STRING) - Request headers (JSON)
- `query_params` (STRING) - Query parameters (JSON)
- `cost_cents` (INT64) - Request cost

### Environment Variables

Add to `.env.example`:
```bash
# Real-time features
ENABLE_REALTIME=true
PUBSUB_SUBSCRIPTION=apx-requests-realtime

# Alerts
SENDGRID_API_KEY=your-sendgrid-key
ALERT_WEBHOOK_URL=https://hooks.slack.com/...

# SLO Configuration
SLO_UPTIME_TARGET=99.9
SLO_LATENCY_P95_TARGET=500
SLO_ERROR_RATE_TARGET=1.0
```

---

## Testing Requirements

### Each Phase Must Have:

1. **Unit Tests** - Component rendering and logic
2. **E2E Tests** - User flows for key features
3. **API Tests** - Endpoint validation
4. **Performance Tests** - Chart rendering with large datasets
5. **Accessibility Tests** - WCAG 2.1 AA compliance

### Specific Test Cases:

**Analytics:**
- Chart renders with 10k+ data points in <500ms
- Time range selection updates charts correctly
- Export generates valid files
- Comparison view calculates correctly

**Request Explorer:**
- Search returns results in <2s for 1M+ requests
- Pagination works smoothly
- Filters combine correctly
- Detail view loads in <1s

**SLO Dashboard:**
- SLO calculations are accurate
- Burn rate alerts trigger at correct thresholds
- Historical data displays correctly

**Real-Time:**
- Stream handles 100+ req/sec without lag
- Pause/resume works correctly
- Filters apply to live stream
- No memory leaks during extended streaming

---

## Success Criteria

### Milestone 2 Complete When:

- [ ] All 15 tasks marked as COMPLETE
- [ ] Enhanced usage charts rendering correctly
- [ ] Request explorer search working (sub-2s)
- [ ] SLO dashboard displaying metrics
- [ ] Real-time tail streaming requests
- [ ] Policy viewer showing active policies
- [ ] All tests passing (250+ existing + 100+ new)
- [ ] TypeScript compiles with zero errors
- [ ] Production build succeeds
- [ ] Lighthouse score remains >85
- [ ] No accessibility regressions
- [ ] Documentation updated

### Quality Gates:

- **Performance:** Charts render large datasets (<500ms)
- **Search:** Request explorer <2s for 1M requests
- **Real-time:** Stream 100+ req/sec without lag
- **Accuracy:** SLO calculations verified
- **Accessibility:** All new pages WCAG 2.1 AA compliant
- **Tests:** 100+ new tests added
- **Bundle Size:** No route >400KB

---

## Parallel Execution Strategy

### Agent Team Assignment:

**Team 1: Analytics Agent**
- PM2-T1-001: Advanced Usage Charts
- PM2-T1-002: Enhanced Metrics
- PM2-T1-005: Analytics Page Redesign

**Team 2: Explorer Agent**
- PM2-T2-001: Request Search & Filter
- PM2-T2-002: Request Detail View
- PM2-T2-003: Request List & Pagination
- PM2-T2-004: Request Analytics

**Team 3: Monitoring Agent**
- PM2-T3-001: SLO Dashboard
- PM2-T3-002: Enhanced System Health
- PM2-T3-003: Alerts & Notifications

**Team 4: Real-Time Agent**
- PM2-T4-001: Real-Time Request Tail
- PM2-T4-002: Live Dashboard Updates

**Team 5: Policy Agent**
- PM2-T5-001: Policy Viewer

**Team 6: Infrastructure Agent**
- PM2-T1-003: Time Range Presets
- PM2-T1-004: Export Enhancements
- Dependencies installation
- Testing infrastructure
- Documentation

### Timeline (with 6 parallel agents):

- **Day 1:** Setup + Core features (6-8 hours)
- **Day 2:** Integration + Polish (6-8 hours)
- **Day 3:** Testing + QA (4-6 hours)
- **Total:** 2-3 days

---

## Backend Integration Notes

### APX Services Required:

1. **BigQuery** - Enhanced schema for analytics
2. **Pub/Sub** - Real-time request streaming
3. **Firestore** - Policy storage and SLO config
4. **APX Router** - Health and metrics endpoints
5. **APX Edge** - Request logs and tracing

### API Endpoints to Build:

```
GET  /api/analytics/latency       - Latency percentiles
GET  /api/analytics/errors         - Error analysis
GET  /api/analytics/breakdown      - Request breakdowns
GET  /api/requests                 - Search requests
GET  /api/requests/[id]            - Request details
GET  /api/slo                      - SLO metrics
GET  /api/policies                 - Active policies
GET  /api/tail                     - SSE stream (real-time)
WS   /api/ws/metrics              - WebSocket (live metrics)
```

---

## Next Steps

1. **Review & Approve** this execution plan
2. **Launch 6 parallel agent teams**
3. **Monitor progress** via task tracker
4. **Daily sync** to resolve blockers
5. **QA review** before marking M2 complete

---

**Ready to launch Milestone 2?** 🚀

This plan provides comprehensive analytics and observability features while maintaining the high quality standards from M0 and M1.
