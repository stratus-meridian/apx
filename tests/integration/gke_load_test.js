import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

// Custom metrics
const errorRate = new Rate('errors');
const requestDuration = new Trend('request_duration');
const rateLimitHits = new Counter('rate_limit_hits');
const successfulRequests = new Counter('successful_requests');

// Configuration - GKE via port-forward
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
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
  },

  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],  // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.05'],  // Error rate < 5% (more lenient for direct testing)
    errors: ['rate<0.05'],
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

// Test function - GKE API testing
export function testAPI() {
  const requestId = `gke-load-${__VU}-${__ITER}-${Date.now()}`;
  const tenantId = `tenant-gke-${(__VU % 5) + 1}`; // Distribute across 5 tenants

  const payload = JSON.stringify({
    test: 'gke-load-test',
    vu: __VU,
    iter: __ITER,
    timestamp: Date.now(),
    environment: 'gke',
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
    if (res.status !== 429) { // Don't log rate limits as errors
      console.log(`Error: VU ${__VU}, Status ${res.status}, Duration ${duration}ms`);
    }
  } else {
    errorRate.add(0);
    successfulRequests.add(1);
  }

  // Track rate limiting
  if (res.status === 429) {
    rateLimitHits.add(1);
  }

  // Small sleep to prevent overwhelming the system
  sleep(0.1);
}

// Health check during setup
export function setup() {
  console.log(`\n=== GKE Load Test Setup ===`);
  console.log(`Target: ${BASE_URL}`);
  console.log(`Scenario: ${SCENARIO}`);
  console.log(`Testing GKE deployment via port-forward`);

  const healthRes = http.get(`${BASE_URL}/healthz`);
  const healthCheck = check(healthRes, {
    'health check passed': (r) => r.status === 200,
  });

  if (!healthCheck) {
    console.log(`Health check failed with status ${healthRes.status}`);
    console.log(`Response: ${healthRes.body}`);
    throw new Error('Health check failed - aborting test');
  }

  console.log('Health check passed ✓');
  console.log('===========================\n');

  return { startTime: new Date().toISOString() };
}

// Summary report
export function handleSummary(data) {
  const scenario = SCENARIO || 'unknown';
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');

  // Calculate key metrics
  const requests = data.metrics.http_reqs?.values.count || 0;
  const failures = data.metrics.http_req_failed?.values.passes || 0;
  const successes = requests - failures;
  const errorRateVal = requests > 0 ? (failures / requests * 100).toFixed(2) : 0;
  const successRate = requests > 0 ? (successes / requests * 100).toFixed(2) : 0;

  const p50 = data.metrics.http_req_duration?.values['p(50)']?.toFixed(2) || 'N/A';
  const p95 = data.metrics.http_req_duration?.values['p(95)']?.toFixed(2) || 'N/A';
  const p99 = data.metrics.http_req_duration?.values['p(99)']?.toFixed(2) || 'N/A';
  const median = data.metrics.http_req_duration?.values.med?.toFixed(2) || 'N/A';
  const avg = data.metrics.http_req_duration?.values.avg?.toFixed(2) || 'N/A';
  const min = data.metrics.http_req_duration?.values.min?.toFixed(2) || 'N/A';
  const max = data.metrics.http_req_duration?.values.max?.toFixed(2) || 'N/A';

  const rateLimits = data.metrics.rate_limit_hits?.values.count || 0;
  const duration = data.state?.testRunDurationMs || 0;
  const durationSec = (duration / 1000).toFixed(2);
  const rps = duration > 0 ? (requests / (duration / 1000)).toFixed(2) : 0;

  console.log(`\n=== GKE Load Test Summary: ${scenario} ===`);
  console.log(`Timestamp: ${timestamp}`);
  console.log(`Duration: ${durationSec}s`);
  console.log(`\nRequests:`);
  console.log(`  Total: ${requests}`);
  console.log(`  Successful: ${successes} (${successRate}%)`);
  console.log(`  Failed: ${failures} (${errorRateVal}%)`);
  console.log(`  Rate Limited: ${rateLimits}`);
  console.log(`  RPS: ${rps} req/s`);
  console.log(`\nLatency:`);
  console.log(`  Min: ${min}ms`);
  console.log(`  Avg: ${avg}ms`);
  console.log(`  Median (p50): ${median}ms`);
  console.log(`  p95: ${p95}ms ${p95 !== 'N/A' && parseFloat(p95) < 500 ? '✓' : '✗'} (target: <500ms)`);
  console.log(`  p99: ${p99}ms ${p99 !== 'N/A' && parseFloat(p99) < 1000 ? '✓' : '✗'} (target: <1000ms)`);
  console.log(`  Max: ${max}ms`);
  console.log('==========================================\n');

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    [`tests/integration/results/gke-load-test-${scenario}-${timestamp}.json`]: JSON.stringify(data, null, 2),
  };
}
