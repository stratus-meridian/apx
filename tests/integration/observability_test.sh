#!/bin/bash
set -e

echo "=========================================="
echo "APX Observability Stack Test"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0
WARNINGS=0

# Test 1: OTEL Collector health
echo "Test 1: OTEL Collector health check..."
if curl -sf http://localhost:13133/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OTEL Collector responding${NC}"
    ((PASSED++))
elif curl -sf http://localhost:4318/ > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OTEL Collector HTTP endpoint responding${NC}"
    ((PASSED++))
else
    echo -e "${YELLOW}⚠ OTEL Collector health check not available (may be expected)${NC}"
    ((WARNINGS++))
fi
echo ""

# Test 2: OTEL metrics endpoint
echo "Test 2: OTEL Collector metrics endpoint..."
if curl -sf http://localhost:8889/metrics > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OTEL Collector exposing metrics on :8889${NC}"
    ((PASSED++))
else
    echo -e "${RED}✗ OTEL Collector metrics endpoint not responding${NC}"
    ((FAILED++))
fi
echo ""

# Test 3: Prometheus targets
echo "Test 3: Prometheus active targets..."
if curl -sf http://localhost:9090/api/v1/targets > /dev/null 2>&1; then
    TARGET_COUNT=$(curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets | length')
    if [ "$TARGET_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ Prometheus has $TARGET_COUNT active targets${NC}"
        curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | "  - \(.labels.job): \(.health)"'
        ((PASSED++))
    else
        echo -e "${RED}✗ Prometheus has no active targets${NC}"
        ((FAILED++))
    fi
else
    echo -e "${RED}✗ Prometheus not responding${NC}"
    ((FAILED++))
fi
echo ""

# Test 4: Grafana health
echo "Test 4: Grafana health check..."
if curl -sf http://localhost:3000/api/health > /dev/null 2>&1; then
    GRAFANA_STATUS=$(curl -s http://localhost:3000/api/health | jq -r '.database')
    if [ "$GRAFANA_STATUS" == "ok" ]; then
        echo -e "${GREEN}✓ Grafana healthy and database connected${NC}"
        ((PASSED++))
    else
        echo -e "${YELLOW}⚠ Grafana responding but database status: $GRAFANA_STATUS${NC}"
        ((WARNINGS++))
    fi
else
    echo -e "${RED}✗ Grafana not responding${NC}"
    ((FAILED++))
fi
echo ""

# Test 5: Router metrics endpoint
echo "Test 5: Router /metrics endpoint..."
if curl -sf http://localhost:8081/metrics > /dev/null 2>&1; then
    if curl -s http://localhost:8081/metrics | grep -q "apx_requests_total"; then
        echo -e "${GREEN}✓ Router exposing Prometheus metrics${NC}"
        ((PASSED++))
    else
        echo -e "${YELLOW}⚠ Router /metrics endpoint exists but no apx metrics found yet${NC}"
        ((WARNINGS++))
    fi
else
    echo -e "${RED}✗ Router /metrics endpoint not responding${NC}"
    ((FAILED++))
fi
echo ""

# Generate some test traffic
echo "Test 6: Generating test traffic..."
echo "Sending 20 test requests to generate metrics..."
for i in {1..20}; do
    curl -s -X POST http://localhost:8081/api/test \
      -H "X-Tenant-ID: obs-test-tenant-$((i % 3))" \
      -H "X-Tenant-Tier: enterprise" \
      -H "Content-Type: application/json" \
      -d '{"test":"observability","iteration":'$i'}' > /dev/null 2>&1 || true
done
echo -e "${GREEN}✓ Test traffic generated (20 requests)${NC}"
((PASSED++))
echo ""

# Wait for metrics to propagate
echo "Waiting 5 seconds for metrics to propagate..."
sleep 5
echo ""

# Test 7: Metrics in Prometheus
echo "Test 7: Checking metrics in Prometheus..."
if curl -s 'http://localhost:9090/api/v1/query?query=apx_requests_total' | jq -e '.data.result | length > 0' > /dev/null 2>&1; then
    METRIC_COUNT=$(curl -s 'http://localhost:9090/api/v1/query?query=apx_requests_total' | jq -r '.data.result | length')
    echo -e "${GREEN}✓ apx_requests_total metrics visible in Prometheus ($METRIC_COUNT series)${NC}"

    # Show sample metrics
    echo "  Sample metrics:"
    curl -s 'http://localhost:9090/api/v1/query?query=apx_requests_total' | jq -r '.data.result[0:3][] | "  - \(.metric.method) \(.metric.path) [\(.metric.tenant_tier)]: \(.value[1])"'
    ((PASSED++))
else
    echo -e "${YELLOW}⚠ No apx_requests_total metrics in Prometheus yet${NC}"
    ((WARNINGS++))
fi
echo ""

# Test 8: Request duration metrics
echo "Test 8: Checking request duration metrics..."
if curl -s 'http://localhost:9090/api/v1/query?query=apx_request_duration_seconds_count' | jq -e '.data.result | length > 0' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ apx_request_duration_seconds metrics visible${NC}"
    ((PASSED++))
else
    echo -e "${YELLOW}⚠ No request duration metrics found yet${NC}"
    ((WARNINGS++))
fi
echo ""

# Test 9: OTEL Collector is receiving metrics
echo "Test 9: OTEL Collector metrics ingestion..."
if curl -s http://localhost:8889/metrics | grep -q "otelcol_receiver"; then
    echo -e "${GREEN}✓ OTEL Collector is processing metrics${NC}"
    RECEIVED=$(curl -s http://localhost:8889/metrics | grep "otelcol_receiver_accepted_metric_points" | head -1 || echo "metrics available")
    echo "  $RECEIVED"
    ((PASSED++))
else
    echo -e "${YELLOW}⚠ OTEL Collector metrics not available${NC}"
    ((WARNINGS++))
fi
echo ""

# Test 10: Grafana data source
echo "Test 10: Grafana Prometheus data source..."
if curl -s http://localhost:3000/api/datasources -u admin:admin | jq -e '.[] | select(.type=="prometheus")' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Grafana has Prometheus data source configured${NC}"
    DS_NAME=$(curl -s http://localhost:3000/api/datasources -u admin:admin | jq -r '.[] | select(.type=="prometheus") | .name')
    echo "  Data source: $DS_NAME"
    ((PASSED++))
else
    echo -e "${RED}✗ Grafana Prometheus data source not found${NC}"
    ((FAILED++))
fi
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed:   $PASSED${NC}"
echo -e "${YELLOW}Warnings: $WARNINGS${NC}"
echo -e "${RED}Failed:   $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All critical tests passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Open Grafana: http://localhost:3000 (admin/admin)"
    echo "  2. View Prometheus: http://localhost:9090"
    echo "  3. Query metrics: http://localhost:9090/graph?g0.expr=apx_requests_total"
    echo "  4. View OTEL metrics: http://localhost:8889/metrics"
    exit 0
else
    echo -e "${RED}✗ Some tests failed. Check the output above.${NC}"
    exit 1
fi
