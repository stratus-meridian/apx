import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const requestDuration = new Trend('request_duration');
const rateLimitHits = new Counter('rate_limit_hits');

// Configuration
const BASE_URL = 'https://api.apx.build';

// Scenario selection via environment variable
const SCENARIO = __ENV.SCENARIO || 'baseline';

// Scenario configurations
export const options = {
  scenarios: {
    baseline: {
      executor: 'constant-vus',
      vus: 10,
      duration: '2m',
      exec: 'testAPI',
    },
    rampup: {
      executor: 'ramping-vus',
      startVUs: 10,
      stages: [
        { duration: '1m', target: 50 },  // Ramp up to 50 VUs
        { duration: '3m', target: 50 },  // Stay at 50 VUs
        { duration: '1m', target: 0 },   // Ramp down to 0
      ],
      exec: 'testAPI',
    },
    spike: {
      executor: 'ramping-vus',
      startVUs: 10,
      stages: [
        { duration: '30s', target: 10 },   // Baseline
        { duration: '10s', target: 100 },  // Spike to 100
        { duration: '30s', target: 100 },  // Stay at spike
        { duration: '10s', target: 10 },   // Return to baseline
        { duration: '30s', target: 10 },   // Stay at baseline
      ],
      exec: 'testAPI',
    },
    sustained: {
      executor: 'constant-vus',
      vus: 100,
      duration: '10m',
      exec: 'testAPI',
    },
    ratelimit: {
      executor: 'constant-arrival-rate',
      rate: 200,           // 200 requests per timeUnit
      timeUnit: '1m',      // per minute
      duration: '2m',
      preAllocatedVUs: 10,
      maxVUs: 50,
      exec: 'testRateLimit',
    },
    multitenant: {
      executor: 'per-vu-iterations',
      vus: 100,  // 5 tenants × 20 VUs each
      iterations: 30,  // Each VU makes 30 requests
      maxDuration: '5m',
      exec: 'testMultiTenant',
    },
  },

  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],  // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.01'],  // Error rate < 1%
    errors: ['rate<0.01'],
  },
};

// Only run the selected scenario
if (SCENARIO !== 'all') {
  const selectedScenario = options.scenarios[SCENARIO];
  if (selectedScenario) {
    options.scenarios = { [SCENARIO]: selectedScenario };
  } else {
    throw new Error(`Unknown scenario: ${SCENARIO}`);
  }
}

// Test functions
export function testAPI() {
  const requestId = `load-${__VU}-${__ITER}-${Date.now()}`;
  const tenantId = `tenant-${(__VU % 5) + 1}`; // Distribute across 5 tenants

  const payload = JSON.stringify({
    test: 'load-test',
    vu: __VU,
    iter: __ITER,
    timestamp: Date.now(),
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Request-ID': requestId,
      'X-Tenant-ID': tenantId,
    },
    timeout: '30s',
  };

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/api/test`, payload, params);
  const duration = Date.now() - startTime;

  // Record metrics
  requestDuration.add(duration);

  // Check response
  const success = check(res, {
    'status is 200 or 202': (r) => r.status === 200 || r.status === 202,
    'response has request_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.request_id !== undefined;
      } catch {
        return false;
      }
    },
    'response time < 2s': (r) => r.timings.duration < 2000,
  });

  if (!success) {
    errorRate.add(1);
    console.log(`Error: VU ${__VU}, Status ${res.status}, Duration ${duration}ms`);
  } else {
    errorRate.add(0);
  }

  // Track rate limiting
  if (res.status === 429) {
    rateLimitHits.add(1);
  }

  // Small sleep to prevent overwhelming the system
  sleep(0.1);
}

export function testRateLimit() {
  const requestId = `ratelimit-${__VU}-${__ITER}-${Date.now()}`;

  const payload = JSON.stringify({
    test: 'rate-limit-test',
    vu: __VU,
    iter: __ITER,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Request-ID': requestId,
      'X-Tenant-ID': 'ratelimit-test',
    },
    timeout: '10s',
  };

  const res = http.post(`${BASE_URL}/api/test`, payload, params);

  // For rate limit testing, we expect 429s after threshold
  const success = check(res, {
    'status is 200, 202, or 429': (r) => [200, 202, 429].includes(r.status),
  });

  if (res.status === 429) {
    rateLimitHits.add(1);
  }

  if (!success) {
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }

  // No sleep - we want to hit rate limits
}

export function testMultiTenant() {
  const tenantId = `tenant-${(__VU % 5) + 1}`; // 5 tenants
  const requestId = `multitenant-${tenantId}-${__VU}-${__ITER}-${Date.now()}`;

  const payload = JSON.stringify({
    test: 'multi-tenant-test',
    tenant: tenantId,
    vu: __VU,
    iter: __ITER,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Request-ID': requestId,
      'X-Tenant-ID': tenantId,
    },
    timeout: '30s',
  };

  const res = http.post(`${BASE_URL}/api/test`, payload, params);

  const success = check(res, {
    'status is 200 or 202': (r) => r.status === 200 || r.status === 202,
    'response has tenant_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.tenant_id === tenantId;
      } catch {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }

  sleep(0.2); // Slight delay between requests
}

// Health check during setup
export function setup() {
  console.log(`\n=== Load Test Setup ===`);
  console.log(`Target: ${BASE_URL}`);
  console.log(`Scenario: ${SCENARIO}`);

  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health check passed': (r) => r.status === 200,
  });

  if (healthRes.status !== 200) {
    throw new Error('Health check failed - aborting test');
  }

  console.log('Health check passed ✓');
  console.log('======================\n');
}

// Summary report
export function handleSummary(data) {
  const scenario = SCENARIO || 'unknown';
  const timestamp = new Date().toISOString();

  // Calculate key metrics
  const requests = data.metrics.http_reqs?.values.count || 0;
  const failures = data.metrics.http_req_failed?.values.passes || 0;
  const errorRateVal = requests > 0 ? (failures / requests * 100).toFixed(2) : 0;
  const p95 = data.metrics.http_req_duration?.values['p(95)']?.toFixed(2) || 'N/A';
  const p99 = data.metrics.http_req_duration?.values['p(99)']?.toFixed(2) || 'N/A';
  const rateLimits = data.metrics.rate_limit_hits?.values.count || 0;

  console.log(`\n=== Load Test Summary: ${scenario} ===`);
  console.log(`Timestamp: ${timestamp}`);
  console.log(`Total Requests: ${requests}`);
  console.log(`Failed Requests: ${failures}`);
  console.log(`Error Rate: ${errorRateVal}%`);
  console.log(`p95 Latency: ${p95}ms`);
  console.log(`p99 Latency: ${p99}ms`);
  console.log(`Rate Limit Hits: ${rateLimits}`);
  console.log('=====================================\n');

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    [`tests/integration/results/load-test-${scenario}-${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}

// Helper for text summary (k6 built-in)
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';
