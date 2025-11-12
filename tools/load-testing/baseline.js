/**
 * APX Load Testing Baseline - k6 Script
 *
 * This script tests the APX platform under load to establish performance baselines.
 *
 * Acceptance Criteria:
 * - Sustained 1k RPS for 5 minutes
 * - p95 latency < 100ms
 * - p99 latency < 200ms
 * - Error rate < 1%
 * - No dropped requests during scale-up
 *
 * Usage:
 *   k6 run --out json=results/baseline-$(date +%Y-%m-%d-%H%M%S).json tools/load-testing/baseline.js
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const requestErrors = new Counter('request_errors');
const requestSuccess = new Counter('request_success');
const errorRate = new Rate('error_rate');
const statusPollLatency = new Trend('status_poll_latency');

// Test configuration
export const options = {
  stages: [
    // Ramp-up phase: 0 -> 100 VUs over 1 minute
    { duration: '1m', target: 100 },

    // Scale to target load: 100 -> 1000 VUs over 2 minutes
    { duration: '2m', target: 1000 },

    // Sustain 1k RPS for 5 minutes (main test phase)
    { duration: '5m', target: 1000 },

    // Ramp-down: 1000 -> 0 over 1 minute
    { duration: '1m', target: 0 },
  ],

  // Performance thresholds (acceptance criteria)
  thresholds: {
    // p95 latency must be < 100ms
    'http_req_duration{type:api_request}': ['p(95)<100', 'p(99)<200'],

    // Error rate must be < 1%
    'error_rate': ['rate<0.01'],

    // 99.9% of requests must succeed
    'http_req_failed{type:api_request}': ['rate<0.001'],

    // Status polling should be fast (< 50ms p95)
    'status_poll_latency': ['p(95)<50'],
  },

  // Test metadata
  tags: {
    test_name: 'baseline_load_test',
    environment: 'dev',
  },
};

// Generate unique tenant IDs for load distribution
function getTenantID(vu) {
  // Distribute load across 100 simulated tenants
  const tenantNumber = (vu % 100) + 1;
  return `tenant-${String(tenantNumber).padStart(3, '0')}`;
}

// Main test function - executed by each Virtual User
export default function() {
  const requestID = `load-${__ITER}-${__VU}-${Date.now()}`;
  const tenantID = getTenantID(__VU);

  // Prepare request payload
  const payload = JSON.stringify({
    message: 'load test request',
    timestamp: Date.now(),
    iteration: __ITER,
    vu: __VU,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantID,
      'X-Request-ID': requestID,
      'X-Tenant-Tier': 'standard', // Mix of tiers
    },
    tags: { type: 'api_request' },
  };

  // Main API request (202 Accepted pattern)
  const response = http.post('http://localhost:8081/api/test', payload, params);

  // Validate response
  const apiCheckResult = check(response, {
    'status is 202': (r) => r.status === 202,
    'has request_id': (r) => {
      try {
        return r.json('request_id') !== undefined;
      } catch (e) {
        return false;
      }
    },
    'has status_url': (r) => {
      try {
        return r.json('status_url') !== undefined;
      } catch (e) {
        return false;
      }
    },
    'has stream_url': (r) => {
      try {
        return r.json('stream_url') !== undefined;
      } catch (e) {
        return false;
      }
    },
    'response time < 200ms': (r) => r.timings.duration < 200,
  });

  // Track metrics
  if (apiCheckResult) {
    requestSuccess.add(1);
    errorRate.add(0);
  } else {
    requestErrors.add(1);
    errorRate.add(1);
  }

  // Simulate realistic client behavior:
  // 10% of requests poll status (simulating clients checking progress)
  if (Math.random() < 0.1 && response.status === 202) {
    try {
      const statusURL = response.json('status_url');
      if (statusURL) {
        const statusStart = Date.now();
        const statusResponse = http.get(statusURL, {
          tags: { type: 'status_poll' },
        });
        statusPollLatency.add(Date.now() - statusStart);

        check(statusResponse, {
          'status endpoint responds': (r) => r.status === 200 || r.status === 404,
        });
      }
    } catch (e) {
      // Ignore status polling errors
    }
  }

  // Think time: simulate realistic user behavior
  // Average 1 request per second per VU
  sleep(1);
}

// Setup function - runs once before test starts
export function setup() {
  console.log('Starting APX Load Test - Baseline');
  console.log('Target: 1000 VUs (simulating ~1k RPS)');
  console.log('Duration: 9 minutes total (5 minutes sustained)');

  // Verify router is healthy
  const healthCheck = http.get('http://localhost:8081/health');
  if (healthCheck.status !== 200) {
    throw new Error('Router health check failed - ensure services are running');
  }

  console.log('Router health check passed - starting test...');
  return { startTime: Date.now() };
}

// Teardown function - runs once after test completes
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`Test completed in ${duration.toFixed(2)} seconds`);
}
