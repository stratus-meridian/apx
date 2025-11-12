# APX Validation & Hardening Plan

**Purpose:** Transform scaffolded foundation into production-ready, enterprise-grade platform
**Timeline:** 7-day validation sprint + ongoing hardening
**Status:** Ready for execution

---

## Overview

The foundation is solid, but **untested in production conditions**. This plan ensures APX will survive:

- ✅ **Real load** (1k → 20k rps spikes)
- ✅ **Enterprise scrutiny** (SOC2, HIPAA, FedRAMP audits)
- ✅ **Open-source usage** (community contributions, forks)
- ✅ **Malicious actors** (injection attacks, abuse)
- ✅ **Operational chaos** (region outages, API failures)

---

## Phase 1: Immediate Verification (72 Hours)

**Goal:** Prove core contracts work end-to-end

---

### Task V-001: Smoke Check - Request Pathing

```yaml
Task ID: V-001
Priority: P0
Estimated: 2 hours
Agent: backend-agent-1

Objective:
  Verify full request path: edge → router → Pub/Sub → worker → stream

Steps:
  1. Deploy minimal stack to dev environment
     ```bash
     cd /Users/agentsy/APILEE
     make deploy-dev
     ```

  2. Send test request:
     ```bash
     export EDGE_URL=$(gcloud run services describe apx-edge --region=us-central1 --format='value(status.url)')

     curl -X POST $EDGE_URL/v1/test \
       -H "Content-Type: application/json" \
       -H "X-API-Key: test-key-123" \
       -d '{"test": "data", "tenant": "smoke-test"}' \
       -v
     ```

  3. Verify response:
     ```json
     {
       "status": "accepted",
       "request_id": "req-abc123",
       "status_url": "/status/req-abc123",
       "message": "Request queued for processing"
     }
     ```

  4. Trace request through stack:
     ```bash
     # Check Pub/Sub message published
     gcloud pubsub topics list-subscriptions apx-requests-us-dev

     # Check worker received message (logs)
     gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=apx-worker" \
       --limit 10 --format json | jq '.[] | select(.jsonPayload.request_id == "req-abc123")'

     # Check response available
     curl $EDGE_URL/status/req-abc123
     ```

Acceptance Criteria:
  - [ ] Request returns 202 Accepted
  - [ ] Request ID generated and returned
  - [ ] Message published to Pub/Sub
  - [ ] Worker receives and processes message
  - [ ] Status endpoint returns result
  - [ ] End-to-end latency < 5 seconds (dev environment)

Artifacts:
  - tests/smoke/request_pathing_test.sh
```

---

### Task V-002: Header Propagation Verification

```yaml
Task ID: V-002
Priority: P0
Estimated: 1.5 hours
Agent: backend-agent-1
Dependencies: [V-001]

Objective:
  Verify headers set and propagated end-to-end

Steps:
  1. Send request with tracking:
     ```bash
     curl -X POST $EDGE_URL/v1/test \
       -H "X-Request-ID: req-manual-123" \
       -H "X-Tenant-ID: tenant-smoke" \
       -H "X-API-Key: test-key" \
       -d '{"test": "headers"}' \
       -v
     ```

  2. Verify headers in response:
     ```bash
     # Should echo back:
     # X-Request-ID: req-manual-123
     # X-Tenant-ID: tenant-smoke
     # X-APX-Policy-Version: 1.0.0
     # X-APX-Region: us-central1
     ```

  3. Verify headers in Cloud Trace:
     ```bash
     gcloud trace list --filter="displayName:'/v1/test'" --limit 1
     # Check span attributes contain:
     # - request_id
     # - tenant_id
     # - policy_version
     ```

  4. Verify headers in logs:
     ```bash
     gcloud logging read "jsonPayload.request_id='req-manual-123'" --limit 10
     # Every hop should have request_id, tenant_id
     ```

Acceptance Criteria:
  - [ ] Request ID preserved if provided, generated if not
  - [ ] Tenant ID extracted and propagated
  - [ ] Policy version tagged on every request
  - [ ] Region tagged based on deployment
  - [ ] All headers appear in traces
  - [ ] All headers appear in logs (where not redacted)

Artifacts:
  - tests/smoke/header_propagation_test.sh
```

---

### Task V-003: Canary Rollout Test

```yaml
Task ID: V-003
Priority: P0
Estimated: 3 hours
Agent: backend-agent-1 + infrastructure-agent-1
Dependencies: [V-001]

Objective:
  Verify canary rollout with N/N-1 policy support

Steps:
  1. Deploy initial policy (v1.0.0):
     ```bash
     cd configs/samples
     # Edit payments-api.yaml, set version: 1.0.0
     ../tools/cli/apx compile payments-api.yaml
     ../tools/cli/apx apply --env dev
     ```

  2. Send 100 requests, verify all use v1.0.0:
     ```bash
     for i in {1..100}; do
       curl -s $EDGE_URL/v1/payments \
         -H "X-API-Key: test" \
         -d '{"amount": 100}' | jq -r '.policy_version'
     done | sort | uniq -c
     # Expected: 100 v1.0.0
     ```

  3. Deploy canary policy (v1.1.0) at 5%:
     ```bash
     # Edit payments-api.yaml, set version: 1.1.0
     # Add canary config to Route
     ../tools/cli/apx compile payments-api.yaml
     ../tools/cli/apx rollout --canary 5% pb-pay-v1@1.1.0
     ```

  4. Send 1000 requests, verify 5% use v1.1.0:
     ```bash
     for i in {1..1000}; do
       curl -s $EDGE_URL/v1/payments \
         -H "X-API-Key: test" \
         -d '{"amount": 100}' | jq -r '.policy_version'
     done | sort | uniq -c
     # Expected: ~50 v1.1.0, ~950 v1.0.0
     ```

  5. Increase canary to 25%:
     ```bash
     ../tools/cli/apx rollout --increase 25%
     ```

  6. Verify in-flight requests complete with their admitted version:
     ```bash
     # Send long-running request with v1.0.0
     curl -X POST $EDGE_URL/v1/payments/long \
       -H "X-API-Key: test" \
       -d '{"amount": 100, "delay": 30}' &

     # Immediately increase canary to 100%
     ../tools/cli/apx rollout --increase 100%

     # Wait for request to complete
     wait
     # Verify response shows policy_version: 1.0.0 (not 1.1.0)
     ```

  7. Introduce breaking policy (v1.2.0) and verify rollback:
     ```bash
     # Edit payments-api.yaml with intentional error (invalid Rego)
     # Set version: 1.2.0
     ../tools/cli/apx compile payments-api.yaml
     ../tools/cli/apx rollout --canary 5% pb-pay-v1@1.2.0

     # Send requests
     # Monitor error rate
     # Should auto-rollback within 2 minutes
     ```

Acceptance Criteria:
  - [ ] Canary traffic split accurate within ±2%
  - [ ] In-flight requests use admitted policy version
  - [ ] Workers support N and N-1 versions simultaneously
  - [ ] Breaking policy triggers auto-rollback
  - [ ] Rollback completes in < 2 minutes
  - [ ] Zero dropped requests during rollout

Artifacts:
  - tests/integration/canary_rollout_test.sh
  - tools/cli/apx rollout implementation
  - tools/cli/apx rollback implementation
```

---

### Task V-004: Async Contract Verification

```yaml
Task ID: V-004
Priority: P0
Estimated: 2 hours
Agent: backend-agent-2
Dependencies: [V-001]

Objective:
  Verify async pattern: 202 → status polling → stream resume

Steps:
  1. Send long-running request:
     ```bash
     curl -X POST $EDGE_URL/v1/test/long \
       -H "X-API-Key: test" \
       -d '{"delay_seconds": 10}' \
       -v
     ```

  2. Verify 202 response:
     ```json
     {
       "status": "accepted",
       "request_id": "req-long-123",
       "status_url": "/status/req-long-123",
       "message": "Request queued for processing"
     }
     ```

  3. Poll status endpoint:
     ```bash
     for i in {1..20}; do
       curl -s $EDGE_URL/status/req-long-123 | jq
       sleep 1
     done
     ```

  4. Verify status transitions:
     ```json
     {"status": "queued"}     # Initial
     {"status": "processing"} # Worker picked up
     {"status": "complete", "result": {...}} # Done
     ```

  5. Test SSE streaming:
     ```bash
     curl -N $EDGE_URL/v1/test/stream \
       -H "X-API-Key: test" \
       -H "Accept: text/event-stream" \
       -d '{"chunks": 10}'

     # Should receive:
     # data: {"chunk": 1, "total": 10}
     # data: {"chunk": 2, "total": 10}
     # ...
     # data: {"chunk": 10, "total": 10}
     ```

  6. Test stream resume:
     ```bash
     # Start stream
     curl -N $EDGE_URL/v1/test/stream \
       -H "Accept: text/event-stream" &
     PID=$!

     # Kill after receiving 5 chunks
     sleep 2
     kill $PID

     # Resume with token
     curl -N $EDGE_URL/v1/test/stream \
       -H "X-Resume-Token: <token-from-last-chunk>" \
       -H "Accept: text/event-stream"

     # Should start from chunk 6
     ```

Acceptance Criteria:
  - [ ] 202 response includes status_url
  - [ ] Status endpoint shows correct state transitions
  - [ ] SSE streaming works for long responses
  - [ ] Stream resume token works correctly
  - [ ] Clients can poll status every 1s without rate limiting
  - [ ] Status available for 24 hours after completion

Artifacts:
  - tests/integration/async_pattern_test.sh
  - workers/streaming_aggregator implementation
```

---

### Task V-005: Cost Controls Verification

```yaml
Task ID: V-005
Priority: P0
Estimated: 2 hours
Agent: observability-agent-1
Dependencies: [V-001]

Objective:
  Verify log sampling, BQ partitioning, budget alerts work

Steps:
  1. Generate 10k requests:
     ```bash
     for i in {1..10000}; do
       curl -s -o /dev/null $EDGE_URL/v1/test \
         -H "X-API-Key: test" \
         -d '{"test": "load"}'
     done
     ```

  2. Verify log sampling at edge:
     ```bash
     # Query Cloud Logging
     gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=apx-edge" \
       --limit 10000 --format json > edge_logs.json

     # Count success logs (status < 400)
     cat edge_logs.json | jq '[.[] | select(.jsonPayload.status < 400)] | length'
     # Should be ~100 (1% of 10k)

     # Count error logs (status >= 400)
     cat edge_logs.json | jq '[.[] | select(.jsonPayload.status >= 400)] | length'
     # Should be 100% of errors
     ```

  3. Verify BigQuery table partitioned:
     ```bash
     bq show --schema --format=prettyjson apx-dev:analytics.requests
     # Should have:
     # - timePartitioning: DAY
     # - clustering: tenant_tier, route_pattern
     ```

  4. Verify aggregates exist:
     ```bash
     bq query --use_legacy_sql=false '
     SELECT
       DATE(timestamp) as date,
       tenant_tier,
       route_pattern,
       COUNT(*) as request_count,
       AVG(duration_ms) as avg_duration
     FROM `apx-dev.analytics.requests_hourly_agg`
     WHERE DATE(timestamp) = CURRENT_DATE()
     GROUP BY date, tenant_tier, route_pattern
     '
     # Should return aggregated data
     ```

  5. Test budget alert:
     ```bash
     # Manually trigger high-volume logging
     # (skip sampling temporarily)

     # Verify alert fires when cost > threshold
     gcloud logging read "severity=WARNING AND jsonPayload.alert='observability_budget'" \
       --limit 1
     ```

  6. Verify auto-sampling adjustment:
     ```bash
     # When budget alarm triggers:
     # - Sample rate should drop to 0.1%
     # - Alert should show in Slack/email
     # - Dashboard should show reduced ingestion
     ```

Acceptance Criteria:
  - [ ] Log sampling at 1% for success requests
  - [ ] 100% of error logs captured
  - [ ] BigQuery partitioned by day, clustered correctly
  - [ ] Hourly aggregates materialized
  - [ ] Raw logs expire after 30 days
  - [ ] Budget alert fires at 70% threshold
  - [ ] Auto-sampling adjustment works

Artifacts:
  - tests/integration/cost_controls_test.sh
  - observability/budgets/alert_rules.yaml
```

---

### Task V-006: Load Testing Baseline

```yaml
Task ID: V-006
Priority: P0
Estimated: 4 hours
Agent: observability-agent-1 + backend-agent-2
Dependencies: [V-001, V-002, V-003, V-004, V-005]

Objective:
  Establish performance baseline under load

Steps:
  1. Install k6:
     ```bash
     brew install k6  # macOS
     # or
     wget https://github.com/grafana/k6/releases/download/v0.47.0/k6-v0.47.0-linux-amd64.tar.gz
     tar -xzf k6-v0.47.0-linux-amd64.tar.gz
     ```

  2. Create load test script:
     ```bash
     cat > tools/load-testing/baseline.js <<'EOF'
     import http from 'k6/http';
     import { check, sleep } from 'k6';

     export const options = {
       stages: [
         { duration: '2m', target: 100 },   // Ramp up to 100 VUs
         { duration: '5m', target: 100 },   // Stay at 100
         { duration: '2m', target: 1000 },  // Ramp up to 1000
         { duration: '5m', target: 1000 },  // Stay at 1000
         { duration: '2m', target: 5000 },  // Spike to 5000
         { duration: '2m', target: 5000 },  // Stay at 5000
         { duration: '2m', target: 0 },     // Ramp down
       ],
       thresholds: {
         http_req_duration: ['p(95)<100', 'p(99)<200'],
         http_req_failed: ['rate<0.01'],
       },
     };

     export default function () {
       const url = __ENV.EDGE_URL + '/v1/test';
       const payload = JSON.stringify({ test: 'load' });
       const params = {
         headers: {
           'Content-Type': 'application/json',
           'X-API-Key': 'test-key',
         },
       };

       const res = http.post(url, payload, params);

       check(res, {
         'status is 202': (r) => r.status === 202,
         'has request_id': (r) => r.json('request_id') !== undefined,
       });

       sleep(0.1);
     }
     EOF
     ```

  3. Run baseline test:
     ```bash
     export EDGE_URL=$(gcloud run services describe apx-edge --region=us-central1 --format='value(status.url)')

     k6 run tools/load-testing/baseline.js \
       --out json=results/baseline.json \
       --out influxdb=http://localhost:8086/k6
     ```

  4. Analyze results:
     ```bash
     cat results/baseline.json | jq -r '
       select(.type=="Point" and .metric=="http_req_duration")
       | .data.value
     ' | awk '
       {sum+=$1; count+=1}
       END {print "Average:", sum/count, "ms"}
     '
     ```

  5. Verify SLOs:
     ```bash
     # p95 < 100ms
     # p99 < 200ms
     # Error rate < 1%
     # Auto-scaling worked (instances increased)

     gcloud run services describe apx-edge --region=us-central1 \
       --format='value(status.traffic[0].revisionName)' | \
       xargs gcloud run revisions describe --region=us-central1 \
       --format='value(status.containerStatuses[0].imageDigest)'
     ```

Acceptance Criteria:
  - [ ] Sustained 1k rps for 5 minutes
  - [ ] p95 latency < 100ms
  - [ ] p99 latency < 200ms
  - [ ] Error rate < 1%
  - [ ] Auto-scaling: 1 → 100 instances
  - [ ] No dropped requests during scale-up
  - [ ] BigQuery cost < $1 for test duration

Artifacts:
  - tools/load-testing/baseline.js
  - tools/load-testing/analyze_results.sh
  - results/baseline-YYYY-MM-DD.json
  - docs/runbooks/load-testing.md
```

---

## Phase 2: Things to Double-Check (Days 3-5)

**Goal:** Harden security, reliability, and operational readiness

---

### Task V-007: Immutable Signed Artifacts

```yaml
Task ID: V-007
Priority: P0
Estimated: 3 hours
Agent: infrastructure-agent-1

Objective:
  Ensure policy artifacts are immutable and signed

Steps:
  1. Install cosign:
     ```bash
     brew install cosign
     # or
     wget https://github.com/sigstore/cosign/releases/download/v2.2.0/cosign-linux-amd64
     chmod +x cosign-linux-amd64
     mv cosign-linux-amd64 /usr/local/bin/cosign
     ```

  2. Generate signing key:
     ```bash
     cosign generate-key-pair
     # Stores private key in Secret Manager
     gcloud secrets create apx-artifact-signing-key \
       --data-file=cosign.key
     ```

  3. Update policy compiler to sign artifacts:
     ```bash
     cat > control/compiler/sign.go <<'EOF'
     package compiler

     import (
       "crypto"
       "github.com/sigstore/cosign/v2/pkg/cosign"
     )

     func SignArtifact(artifactPath string, privateKey crypto.PrivateKey) error {
       // Sign artifact with cosign
       sig, err := cosign.SignBlob(ctx, privateKey, artifactPath)
       if err != nil {
         return err
       }

       // Store signature alongside artifact
       sigPath := artifactPath + ".sig"
       return os.WriteFile(sigPath, sig, 0644)
     }
     EOF
     ```

  4. Update workers to verify signatures:
     ```bash
     cat > workers/cpu-pool/verify.go <<'EOF'
     package worker

     import (
       "github.com/sigstore/cosign/v2/pkg/cosign"
     )

     func VerifyArtifact(artifactPath string, publicKey crypto.PublicKey) error {
       sigPath := artifactPath + ".sig"
       sig, err := os.ReadFile(sigPath)
       if err != nil {
         return err
       }

       return cosign.VerifyBlobSignature(ctx, artifactPath, sig, publicKey)
     }

     func LoadPolicy(ref string) (*PolicyBundle, error) {
       artifact := downloadArtifact(ref)

       // CRITICAL: Verify signature before using
       if err := VerifyArtifact(artifact.Path, publicKey); err != nil {
         return nil, fmt.Errorf("signature verification failed: %w", err)
       }

       // Reject unknown or unsigned versions
       if artifact.Version != "1.0.0" && artifact.Version != "1.1.0" {
         return nil, fmt.Errorf("unknown policy version: %s", artifact.Version)
       }

       return parsePolicy(artifact)
     }
     EOF
     ```

  5. Test signature verification:
     ```bash
     # Compile and sign policy
     ./tools/cli/apx compile configs/samples/payments-api.yaml
     # Should produce: pb-pay-v1@1.0.0.wasm + pb-pay-v1@1.0.0.wasm.sig

     # Verify worker rejects unsigned artifact
     rm pb-pay-v1@1.0.0.wasm.sig
     # Worker should fail to load policy

     # Verify worker rejects tampered artifact
     echo "malicious" >> pb-pay-v1@1.0.0.wasm
     # Worker should fail signature verification
     ```

Acceptance Criteria:
  - [ ] All artifacts signed with cosign
  - [ ] Workers verify signatures before loading
  - [ ] Unsigned artifacts rejected
  - [ ] Tampered artifacts rejected
  - [ ] Unknown policy versions rejected
  - [ ] Signing key stored in Secret Manager

Artifacts:
  - control/compiler/sign.go
  - workers/cpu-pool/verify.go
  - .github/workflows/sign-artifacts.yaml
```

---

### Task V-008: Tenant Isolation Enforcement

```yaml
Task ID: V-008
Priority: P0
Estimated: 4 hours
Agent: backend-agent-1

Objective:
  Verify tenant isolation at every layer

Steps:
  1. Redis keyspace isolation:
     ```bash
     # Update rate limiter to use tenant prefix
     cat > router/internal/ratelimit/redis.go <<'EOF'
     func RateLimitKey(tenantID, resource string) string {
       // CRITICAL: Tenant ID must be part of key
       return fmt.Sprintf("apx:rl:%s:%s", tenantID, resource)
     }

     func CheckRateLimit(ctx context.Context, tenantID string, limit int) error {
       key := RateLimitKey(tenantID, "requests")

       // CRITICAL: Use tenant-specific key
       count, err := redisClient.Incr(ctx, key).Result()
       if err != nil {
         return err
       }

       if count == 1 {
         redisClient.Expire(ctx, key, time.Minute)
       }

       if count > int64(limit) {
         return ErrRateLimitExceeded
       }

       return nil
     }
     EOF
     ```

  2. Pub/Sub message attributes:
     ```bash
     # Verify messages tagged with tenant_id
     gcloud pubsub subscriptions pull apx-workers-us-dev \
       --limit 10 --format json | \
       jq '.[] | .message.attributes.tenant_id'
     # All messages must have tenant_id
     ```

  3. Worker namespace isolation:
     ```bash
     # Update worker to enforce per-tenant limits
     cat > workers/cpu-pool/limits.go <<'EOF'
     var tenantConcurrency = make(map[string]*semaphore.Weighted)

     func ProcessMessage(msg *pubsub.Message) error {
       tenantID := msg.Attributes["tenant_id"]

       // Get tenant-specific semaphore
       sem, ok := tenantConcurrency[tenantID]
       if !ok {
         limit := getTenantConcurrencyLimit(tenantID)
         sem = semaphore.NewWeighted(int64(limit))
         tenantConcurrency[tenantID] = sem
       }

       // CRITICAL: Enforce per-tenant concurrency
       if !sem.TryAcquire(1) {
         return fmt.Errorf("tenant %s exceeded concurrency limit", tenantID)
       }
       defer sem.Release(1)

       return processRequest(msg)
     }
     EOF
     ```

  4. Log isolation test:
     ```bash
     # Send requests from two tenants
     curl -X POST $EDGE_URL/v1/test \
       -H "X-Tenant-ID: tenant-a" \
       -H "X-API-Key: key-a" \
       -d '{"secret": "tenant-a-data"}'

     curl -X POST $EDGE_URL/v1/test \
       -H "X-Tenant-ID: tenant-b" \
       -H "X-API-Key: key-b" \
       -d '{"secret": "tenant-b-data"}'

     # Query logs for tenant-a
     gcloud logging read "jsonPayload.tenant_id='tenant-a'" --limit 100

     # CRITICAL: Should NOT see tenant-b's secret data
     # Verify PII redaction working
     ```

  5. Negative test - cross-tenant access:
     ```bash
     # Try to access tenant-b's data with tenant-a's key
     curl -X GET $EDGE_URL/v1/status/tenant-b-request-id \
       -H "X-API-Key: tenant-a-key"
     # Should return 403 Forbidden
     ```

Acceptance Criteria:
  - [ ] Redis keys include tenant_id prefix
  - [ ] Pub/Sub messages tagged with tenant_id
  - [ ] Worker enforces per-tenant concurrency limits
  - [ ] Logs redact cross-tenant data
  - [ ] Cross-tenant access blocked at API level
  - [ ] Negative tests fail as expected

Artifacts:
  - router/internal/ratelimit/redis.go
  - workers/cpu-pool/limits.go
  - tests/security/tenant_isolation_test.sh
```

---

### Task V-009: Replay & Snapshot on Error

```yaml
Task ID: V-009
Priority: P1
Estimated: 3 hours
Agent: backend-agent-2

Objective:
  Capture failed requests for debugging and replay

Steps:
  1. Implement error snapshot:
     ```bash
     cat > workers/cpu-pool/snapshot.go <<'EOF'
     func CaptureErrorSnapshot(ctx context.Context, req *Request, err error) error {
       snapshot := ErrorSnapshot{
         RequestID:      req.ID,
         TenantID:       req.TenantID,
         PolicyVersion:  req.PolicyVersion,
         Timestamp:      time.Now(),
         Error:          err.Error(),

         // PII-safe request body
         RequestBody:    redactPII(req.Body),

         // System context
         WorkerPool:     os.Getenv("WORKER_POOL"),
         WorkerInstance: os.Getenv("INSTANCE_ID"),
         QueueDepth:     getQueueDepth(),
         MemoryUsage:    getMemoryUsage(),
       }

       // Store to GCS
       bucket := "apx-error-snapshots"
       path := fmt.Sprintf("%s/%s/%s.json", req.TenantID, req.RequestID, time.Now().Format("20060102"))

       data, _ := json.Marshal(snapshot)
       return uploadToGCS(ctx, bucket, path, data)
     }

     func processRequest(req *Request) error {
       defer func() {
         if r := recover(); r != nil {
           err := fmt.Errorf("panic: %v", r)
           CaptureErrorSnapshot(context.Background(), req, err)
         }
       }()

       // ... normal processing

       if err != nil {
         CaptureErrorSnapshot(ctx, req, err)
         return err
       }

       return nil
     }
     EOF
     ```

  2. Implement replay tool:
     ```bash
     cat > tools/replay/main.go <<'EOF'
     package main

     import (
       "context"
       "encoding/json"
       "flag"
       "fmt"
       "io"
       "os"
     )

     func main() {
       snapshotPath := flag.String("snapshot", "", "Path to error snapshot")
       targetEnv := flag.String("env", "staging", "Environment to replay in")
       flag.Parse()

       // Load snapshot
       f, _ := os.Open(*snapshotPath)
       var snapshot ErrorSnapshot
       json.NewDecoder(f).Decode(&snapshot)

       fmt.Printf("Replaying request: %s\n", snapshot.RequestID)
       fmt.Printf("Original error: %s\n", snapshot.Error)
       fmt.Printf("Policy version: %s\n", snapshot.PolicyVersion)

       // Replay request in staging
       result := replayRequest(context.Background(), snapshot, *targetEnv)

       if result.Success {
         fmt.Println("✅ Replay succeeded")
       } else {
         fmt.Printf("❌ Replay failed: %s\n", result.Error)
       }
     }
     EOF
     ```

  3. Test snapshot capture:
     ```bash
     # Trigger error
     curl -X POST $EDGE_URL/v1/test \
       -H "X-API-Key: test" \
       -d '{"trigger_error": true}'

     # Verify snapshot created
     gsutil ls gs://apx-error-snapshots/tenant-test/
     # Should show snapshot file

     # Download and inspect
     gsutil cp gs://apx-error-snapshots/tenant-test/req-xyz-123/20251111.json .
     cat 20251111.json | jq
     ```

  4. Test replay:
     ```bash
     ./tools/replay/replay \
       --snapshot=20251111.json \
       --env=staging

     # Should re-execute request in staging and show results
     ```

Acceptance Criteria:
  - [ ] Error snapshots captured for all failures
  - [ ] Snapshots include PII-safe request body
  - [ ] Snapshots include system context (queue depth, memory)
  - [ ] Snapshots stored in GCS with lifecycle policy (90 days)
  - [ ] Replay tool can re-execute failed requests
  - [ ] Replay works in staging environment

Artifacts:
  - workers/cpu-pool/snapshot.go
  - tools/replay/main.go
  - docs/runbooks/replay-failed-requests.md
```

---

## Phase 3: Component Hardening (Days 6-7)

I'll create the remaining tasks (V-010 through V-020) covering:
- Edge security headers
- Router caching
- Worker idempotency
- Control plane CMEK
- CI/CD guardrails
- Observability budgets
- Security & compliance
- API contracts
- Load & chaos tests
- Docs & DX improvements

Would you like me to continue with the complete validation plan, or would you prefer to start executing the tasks we've defined so far?
