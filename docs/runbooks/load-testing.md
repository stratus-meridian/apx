# Load Testing Runbook

**Audience:** SRE, DevOps, QA Engineers
**Last Updated:** 2025-11-11
**Status:** Production Ready

---

## Overview

This runbook provides step-by-step instructions for running load tests against the APX platform, analyzing results, and establishing performance baselines.

## Prerequisites

### Tools Required
- **k6** (load testing tool)
- **jq** (JSON parsing)
- **bash** (script execution)
- **docker** and **docker-compose** (for local testing)
- **gcloud** CLI (for GCP testing)

### Installation

#### macOS
```bash
brew install k6 jq
```

#### Linux
```bash
# k6
curl -L https://github.com/grafana/k6/releases/download/v0.48.0/k6-v0.48.0-linux-amd64.tar.gz | tar xvz
sudo mv k6-v0.48.0-linux-amd64/k6 /usr/local/bin/

# jq
sudo apt-get install jq  # Debian/Ubuntu
sudo yum install jq      # RHEL/CentOS
```

#### Verify Installation
```bash
k6 version
jq --version
```

---

## Load Testing Scenarios

### 1. Baseline Test (Development)

**Purpose:** Establish performance baseline in local environment

**Duration:** 9 minutes
**Target Load:** 1000 VUs (Virtual Users)
**Expected RPS:** ~1000 requests/second

#### Steps

1. **Start Infrastructure**
   ```bash
   cd /path/to/APILEE
   docker-compose up -d
   ```

2. **Verify Services**
   ```bash
   # Check router health
   curl http://localhost:8081/health

   # Check Redis
   docker exec -it apilee_redis_1 redis-cli ping
   ```

3. **Run Baseline Test**
   ```bash
   # Create results directory
   mkdir -p results

   # Run test with JSON output
   TIMESTAMP=$(date +%Y-%m-%d-%H%M%S)
   k6 run --out json=results/baseline-${TIMESTAMP}.json \
          tools/load-testing/baseline.js
   ```

4. **Analyze Results**
   ```bash
   # Run analysis script
   bash tools/load-testing/analyze_results.sh results/baseline-${TIMESTAMP}.json

   # View generated report
   cat results/baseline-${TIMESTAMP}_report.md
   ```

#### Expected Results
- **Throughput:** ≥ 900 RPS
- **p95 Latency:** < 100ms
- **p99 Latency:** < 200ms
- **Error Rate:** < 1%

---

### 2. Production Test (GCP)

**Purpose:** Validate auto-scaling and production performance

**Duration:** 30 minutes
**Target Load:** 1000-5000 VUs
**Infrastructure:** Cloud Run or GKE

#### Prerequisites
```bash
# Authenticate with GCP
gcloud auth login

# Set project
export GCP_PROJECT=apx-production
gcloud config set project $GCP_PROJECT

# Get service URL
export ROUTER_URL=$(gcloud run services describe apx-router \
                    --region=us-central1 \
                    --format='value(status.url)')
```

#### Steps

1. **Create Production Test Script**

   Copy `tools/load-testing/baseline.js` to `tools/load-testing/production.js` and update:
   ```javascript
   // Change endpoint URL
   let response = http.post(`${__ENV.ROUTER_URL}/api/test`, payload, params);

   // Add longer test duration
   export let options = {
     stages: [
       { duration: '5m', target: 1000 },   // Ramp-up
       { duration: '20m', target: 1000 },  // Sustain
       { duration: '5m', target: 0 },      // Ramp-down
     ],
     // ... rest of config
   };
   ```

2. **Run Production Test**
   ```bash
   TIMESTAMP=$(date +%Y-%m-%d-%H%M%S)
   k6 run --out json=results/production-${TIMESTAMP}.json \
          -e ROUTER_URL=$ROUTER_URL \
          tools/load-testing/production.js
   ```

3. **Monitor During Test**

   In separate terminals:
   ```bash
   # Watch Cloud Run metrics
   gcloud monitoring dashboards list

   # Monitor logs
   gcloud logging read "resource.type=cloud_run_revision" \
                       --limit=50 --format=json

   # Check auto-scaling
   watch -n 5 'gcloud run services describe apx-router \
                --region=us-central1 \
                --format="value(status.traffic)"'
   ```

4. **Analyze Costs**
   ```bash
   # Check BigQuery costs
   bq query --use_legacy_sql=false '
     SELECT
       DATE(timestamp) as date,
       COUNT(*) as requests,
       SUM(total_bytes) / 1024 / 1024 as mb_ingested
     FROM `apx-production.logs.requests`
     WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
     GROUP BY date
   '
   ```

---

### 3. Stress Test

**Purpose:** Find system breaking point

**Target:** Ramp until failure or 10k RPS

#### Steps

1. **Create Stress Test Script**
   ```javascript
   export let options = {
     stages: [
       { duration: '2m', target: 1000 },
       { duration: '2m', target: 2000 },
       { duration: '2m', target: 5000 },
       { duration: '2m', target: 10000 },
       { duration: '1m', target: 0 },
     ],
     thresholds: {
       // Allow higher error rates to find limits
       'error_rate': ['rate<0.05'],  // 5%
     },
   };
   ```

2. **Run with Monitoring**
   ```bash
   k6 run --out json=results/stress-${TIMESTAMP}.json \
          -e ROUTER_URL=$ROUTER_URL \
          tools/load-testing/stress.js
   ```

3. **Identify Breaking Point**
   - Watch for error rate increase
   - Monitor latency degradation
   - Check resource saturation (CPU, memory)
   - Note maximum sustainable RPS

---

### 4. Soak Test

**Purpose:** Validate long-term stability

**Duration:** 4-8 hours
**Target:** Sustained 500-1000 RPS

#### Steps

1. **Create Soak Test Script**
   ```javascript
   export let options = {
     stages: [
       { duration: '5m', target: 500 },      // Ramp-up
       { duration: '4h', target: 500 },      // Soak
       { duration: '5m', target: 0 },        // Ramp-down
     ],
   };
   ```

2. **Run Overnight**
   ```bash
   nohup k6 run --out json=results/soak-${TIMESTAMP}.json \
                -e ROUTER_URL=$ROUTER_URL \
                tools/load-testing/soak.js > results/soak-${TIMESTAMP}.log 2>&1 &
   ```

3. **Monitor for Issues**
   - Memory leaks (increasing memory over time)
   - Connection pool exhaustion
   - Latency drift (p95/p99 increasing)
   - Error rate creep

---

## Interpreting Results

### k6 Output Metrics

#### HTTP Metrics
- **http_reqs:** Total HTTP requests made
- **http_req_duration:** Total request time (send + wait + receive)
- **http_req_blocked:** Time blocked on connection acquisition
- **http_req_connecting:** Time establishing TCP connection
- **http_req_sending:** Time sending request data
- **http_req_waiting:** Time waiting for server response (TTFB)
- **http_req_receiving:** Time receiving response data
- **http_req_failed:** Number of failed requests

#### Custom Metrics (from baseline.js)
- **request_errors:** Count of failed requests
- **request_success:** Count of successful requests
- **error_rate:** Percentage of failed requests
- **status_poll_latency:** Latency of status endpoint polling

### Success Criteria

#### Green (Healthy)
- ✅ Error rate < 0.1%
- ✅ p95 latency < 50ms
- ✅ p99 latency < 100ms
- ✅ No dropped requests
- ✅ Steady resource usage

#### Yellow (Warning)
- ⚠️ Error rate 0.1% - 1%
- ⚠️ p95 latency 50-100ms
- ⚠️ p99 latency 100-200ms
- ⚠️ Occasional timeouts
- ⚠️ Slow ramp-up/down

#### Red (Critical)
- ❌ Error rate > 1%
- ❌ p95 latency > 100ms
- ❌ p99 latency > 200ms
- ❌ Dropped requests
- ❌ Resource exhaustion

---

## Troubleshooting Performance Issues

### High Latency

#### Symptoms
- p95 > 100ms
- p99 > 200ms
- Increasing latency over time

#### Diagnosis
```bash
# Check router CPU/memory
docker stats apilee_router_1

# Check Redis latency
redis-cli --latency

# Check connection pool
# Look for "connection pool exhausted" in logs
docker logs apilee_router_1 | grep -i "pool"

# Profile router with pprof
curl http://localhost:8081/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

#### Solutions
1. **Scale horizontally:** Add more router instances
2. **Optimize middleware:** Profile and remove expensive operations
3. **Add caching:** Cache policy bundles, tenant configs
4. **Tune connection pools:** Increase max connections
5. **Add CDN:** Cache static responses

### High Error Rate

#### Symptoms
- Error rate > 1%
- 500/502/503 status codes
- Connection refused errors

#### Diagnosis
```bash
# Check error types
cat results/baseline-*.json | jq -r 'select(.metric=="http_req_failed" and .data.value==1) | .data.tags.status' | sort | uniq -c

# Check router logs
docker logs apilee_router_1 --tail=100 | grep ERROR

# Check infrastructure health
docker ps
docker-compose ps
```

#### Solutions
1. **502 Bad Gateway:** Router crashed or not responding
   - Check router logs for panics
   - Verify health endpoint
   - Check resource limits

2. **503 Service Unavailable:** Rate limiting or overload
   - Increase rate limits
   - Scale up resources
   - Enable auto-scaling

3. **Connection Refused:** Service not running
   - Restart services
   - Check network connectivity
   - Verify firewall rules

### Dropped Requests

#### Symptoms
- Requests timing out
- k6 reporting socket errors
- Connection pool exhaustion

#### Diagnosis
```bash
# Check k6 errors
cat results/baseline-*.json | grep error_code

# Check OS connection limits
ulimit -n

# Check Docker network
docker network inspect apilee_apx-network
```

#### Solutions
1. Increase OS file descriptor limits
2. Tune TCP settings (TIME_WAIT, backlog)
3. Increase router connection pool size
4. Add load balancer with connection limiting

---

## Best Practices

### Test Planning

1. **Start Small**
   - Run 1-minute smoke test first
   - Gradually increase load
   - Validate metrics before full test

2. **Monitor During Tests**
   - Watch real-time metrics
   - Set up alerts for critical thresholds
   - Have rollback plan ready

3. **Isolate Changes**
   - Test one change at a time
   - Compare against baseline
   - Document all configuration changes

### Load Test Hygiene

1. **Use Realistic Patterns**
   - Match production traffic mix
   - Include think time (sleep)
   - Vary request sizes

2. **Distribute Load**
   - Use multiple tenants
   - Vary request IDs
   - Randomize payloads

3. **Clean Up**
   - Archive test results
   - Document findings
   - Update baselines

### Production Testing

1. **Schedule Carefully**
   - Avoid peak hours
   - Communicate with team
   - Have SRE on standby

2. **Limit Blast Radius**
   - Use staging environment first
   - Limit to single region
   - Enable circuit breakers

3. **Monitor Everything**
   - Application metrics
   - Infrastructure metrics
   - Cost metrics
   - User experience

---

## Performance Baselines

### Current Baselines (as of 2025-11-11)

#### Single Router Instance (Local)
- **Throughput:** 1000+ RPS sustainable
- **Latency:** p95 < 7ms, p99 < 75ms
- **Concurrency:** 1000+ connections
- **Error Rate:** 0%

#### Production (GCP) - To Be Established
- **Throughput:** TBD
- **Latency:** TBD
- **Auto-scaling:** TBD
- **Cost:** TBD

### When to Update Baselines

- After major performance optimizations
- After infrastructure changes
- Quarterly (at minimum)
- After GCP migrations
- Before/after Go version upgrades

---

## Appendix

### A. k6 Script Reference

See `/tools/load-testing/baseline.js` for full example.

Key sections:
- `options.stages`: Define load profile
- `options.thresholds`: Set success criteria
- `default()`: Main test function
- `setup()`: Pre-test initialization
- `teardown()`: Post-test cleanup

### B. Cloud Run Auto-scaling Configuration

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: apx-router
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "100"
        autoscaling.knative.dev/target: "80"  # Target 80% CPU
    spec:
      containers:
      - image: gcr.io/apx-production/router:latest
        resources:
          limits:
            cpu: "2"
            memory: "1Gi"
```

### C. Useful Commands

```bash
# Quick smoke test (30 seconds)
k6 run --duration 30s --vus 10 tools/load-testing/baseline.js

# Test specific endpoint
k6 run --env TARGET_URL=http://localhost:8081/api/custom tools/load-testing/baseline.js

# Real-time metrics
k6 run --out influxdb=http://localhost:8086/k6 tools/load-testing/baseline.js

# Distributed load test (multiple machines)
k6 run --out cloud tools/load-testing/baseline.js
```

### D. Related Documentation

- [APX Architecture Overview](../architecture/README.md)
- [Monitoring & Observability](../observability/README.md)
- [Incident Response](./incident-response.md)
- [Cost Controls](../COST_CONTROLS.md)

---

## Support

**Questions?** Contact SRE team or see [INDEX.md](../../INDEX.md) for project navigation.

**Issues?** Create ticket with `performance` label and attach test results.

---

*Last updated: 2025-11-11 by AI Agent*
