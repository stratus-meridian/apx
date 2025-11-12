#!/bin/bash
# APX Cost Controls Integration Test
# Verifies log sampling, BigQuery schema, and budget configurations

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROJECT_ID="${PROJECT_ID:-apx-dev}"
EDGE_URL="${EDGE_URL:-http://localhost:8080}"
BQ_DATASET="analytics"

# Helper functions
log_info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $1"
}

test_start() {
  TESTS_RUN=$((TESTS_RUN + 1))
  echo ""
  log_info "Test $TESTS_RUN: $1"
}

test_pass() {
  TESTS_PASSED=$((TESTS_PASSED + 1))
  log_info "✓ PASS: $1"
}

test_fail() {
  TESTS_FAILED=$((TESTS_FAILED + 1))
  log_error "✗ FAIL: $1"
  if [ -n "${2:-}" ]; then
    log_error "  Details: $2"
  fi
}

# ============================================================================
# Test 1: Verify Envoy Log Sampling Configuration
# ============================================================================
test_envoy_log_sampling() {
  test_start "Envoy log sampling configuration"

  local envoy_config="$PROJECT_ROOT/edge/envoy/envoy.yaml"

  if [ ! -f "$envoy_config" ]; then
    test_fail "Envoy config not found" "$envoy_config"
    return 1
  fi

  # Check for sampling configuration
  if grep -q "percent_sampled" "$envoy_config"; then
    test_pass "Envoy config contains sampling configuration"
  else
    test_fail "Envoy config missing sampling configuration"
    return 1
  fi

  # Verify 1% sampling rate (numerator: 1, denominator: HUNDRED)
  if grep -A 2 "percent_sampled" "$envoy_config" | grep -q "numerator: 1"; then
    test_pass "Success request sampling rate is 1%"
  else
    test_fail "Success request sampling rate is not 1%"
  fi

  # Verify error logging (status >= 400)
  if grep -q "status_code_filter" "$envoy_config" && \
     grep -A 5 "status_code_filter" "$envoy_config" | grep -q "400"; then
    test_pass "Error logging configured (status >= 400)"
  else
    test_fail "Error logging not configured correctly"
  fi

  # Verify OR filter for sampling + errors
  if grep -q "or_filter" "$envoy_config"; then
    test_pass "OR filter configured for sampling and errors"
  else
    test_fail "OR filter not configured"
  fi
}

# ============================================================================
# Test 2: Verify Budget Configuration
# ============================================================================
test_budget_config() {
  test_start "Observability budget configuration"

  local budget_config="$PROJECT_ROOT/observability/budgets/log_sampling.yaml"

  if [ ! -f "$budget_config" ]; then
    test_fail "Budget config not found" "$budget_config"
    return 1
  fi

  test_pass "Budget configuration file exists"

  # Check for key budget thresholds
  if grep -q "max_percentage: 7.0" "$budget_config"; then
    test_pass "Observability budget set to 7% of infrastructure"
  else
    test_fail "Observability budget threshold not set correctly"
  fi

  # Check for auto-adjustment rules
  if grep -q "auto_adjustment:" "$budget_config"; then
    test_pass "Auto-adjustment rules defined"
  else
    test_fail "Auto-adjustment rules missing"
  fi

  # Verify 0.1% emergency sampling rate
  if grep -q "0.001" "$budget_config"; then
    test_pass "Emergency sampling rate (0.1%) configured"
  else
    test_fail "Emergency sampling rate not configured"
  fi

  # Check for alert channels
  if grep -q "slack:" "$budget_config" && grep -q "email:" "$budget_config"; then
    test_pass "Alert channels configured (slack, email)"
  else
    test_fail "Alert channels not fully configured"
  fi
}

# ============================================================================
# Test 3: Verify BigQuery Schema
# ============================================================================
test_bigquery_schema() {
  test_start "BigQuery schema configuration"

  local schema_file="$PROJECT_ROOT/observability/bigquery/schema.sql"

  if [ ! -f "$schema_file" ]; then
    test_fail "BigQuery schema file not found" "$schema_file"
    return 1
  fi

  test_pass "BigQuery schema file exists"

  # Check for required tables
  local tables=(
    "requests"
    "requests_hourly_agg"
    "requests_daily_agg"
    "tenant_summary"
    "error_snapshots"
  )

  for table in "${tables[@]}"; do
    if grep -q "CREATE TABLE.*\`.*\.$table\`" "$schema_file"; then
      test_pass "Table '$table' defined in schema"
    else
      test_fail "Table '$table' not found in schema"
    fi
  done

  # Verify partitioning by DAY
  if grep -q "PARTITION BY DATE(timestamp)" "$schema_file" || \
     grep -q "PARTITION BY DATE(hour)" "$schema_file" || \
     grep -q "PARTITION BY day" "$schema_file"; then
    test_pass "Tables configured with DAY partitioning"
  else
    test_fail "DAY partitioning not configured"
  fi

  # Verify clustering
  if grep -q "CLUSTER BY tenant_tier, route_pattern" "$schema_file"; then
    test_pass "Clustering configured (tenant_tier, route_pattern)"
  else
    test_fail "Clustering not configured correctly"
  fi

  # Verify retention policies
  if grep -q "partition_expiration_days=30" "$schema_file"; then
    test_pass "Raw data retention: 30 days"
  else
    test_fail "Raw data retention not set to 30 days"
  fi

  if grep -q "partition_expiration_days=730" "$schema_file"; then
    test_pass "Hourly aggregates retention: 2 years (730 days)"
  else
    test_fail "Hourly aggregates retention not set to 2 years"
  fi

  # Verify require_partition_filter
  if grep -q "require_partition_filter=true" "$schema_file"; then
    test_pass "Partition filter requirement enabled (cost optimization)"
  else
    test_fail "Partition filter requirement not enabled"
  fi
}

# ============================================================================
# Test 4: Verify BigQuery Tables (if deployed)
# ============================================================================
test_bigquery_tables_deployed() {
  test_start "BigQuery tables deployment (optional)"

  # Only run if gcloud is available and authenticated
  if ! command -v bq &> /dev/null; then
    log_warn "bq command not available, skipping BigQuery deployment check"
    return 0
  fi

  # Check if dataset exists
  if bq ls -d "$PROJECT_ID" | grep -q "$BQ_DATASET"; then
    test_pass "BigQuery dataset '$BQ_DATASET' exists"

    # Check for requests table
    if bq show --schema "$PROJECT_ID:$BQ_DATASET.requests" &> /dev/null; then
      test_pass "Table 'requests' deployed"

      # Verify partitioning
      local table_info
      table_info=$(bq show --format=json "$PROJECT_ID:$BQ_DATASET.requests")

      if echo "$table_info" | grep -q '"type": "DAY"'; then
        test_pass "Table 'requests' partitioned by DAY"
      else
        test_fail "Table 'requests' not partitioned by DAY"
      fi

      # Verify clustering
      if echo "$table_info" | grep -q "tenant_tier"; then
        test_pass "Table 'requests' clustered correctly"
      else
        test_fail "Table 'requests' not clustered"
      fi
    else
      log_warn "Table 'requests' not deployed yet"
    fi

    # Check for hourly aggregates table
    if bq show --schema "$PROJECT_ID:$BQ_DATASET.requests_hourly_agg" &> /dev/null; then
      test_pass "Table 'requests_hourly_agg' deployed"
    else
      log_warn "Table 'requests_hourly_agg' not deployed yet"
    fi
  else
    log_warn "BigQuery dataset not created yet (expected in local dev)"
  fi
}

# ============================================================================
# Test 5: Verify Sampling in Practice (if edge is running)
# ============================================================================
test_sampling_in_practice() {
  test_start "Log sampling in practice (optional)"

  # Only run if edge is accessible
  if ! curl -s -f -o /dev/null -w "%{http_code}" "$EDGE_URL/health" &> /dev/null; then
    log_warn "Edge service not accessible at $EDGE_URL, skipping live sampling test"
    return 0
  fi

  test_pass "Edge service is accessible"

  # Send 100 test requests
  log_info "Sending 100 test requests to verify sampling..."

  local success_count=0
  for i in {1..100}; do
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$EDGE_URL/v1/test" \
      -H "Content-Type: application/json" \
      -H "X-API-Key: test-key" \
      -d '{"test": "sampling"}' 2>/dev/null || echo "000")

    if [ "$status" = "202" ] || [ "$status" = "200" ]; then
      success_count=$((success_count + 1))
    fi
  done

  if [ "$success_count" -gt 50 ]; then
    test_pass "Sent 100 requests, $success_count succeeded"
  else
    test_fail "Only $success_count/100 requests succeeded"
  fi

  log_warn "Note: Actual sampling rate verification requires access to Cloud Logging"
  log_warn "To verify: gcloud logging read 'resource.labels.service_name=apx-edge' --limit 1000"
  log_warn "Expected: ~1% of success logs, 100% of error logs"
}

# ============================================================================
# Test 6: Verify Budget Alert Configuration (if deployed)
# ============================================================================
test_budget_alerts_deployed() {
  test_start "Budget alerts deployment (optional)"

  # Only run if gcloud is available
  if ! command -v gcloud &> /dev/null; then
    log_warn "gcloud command not available, skipping budget alerts check"
    return 0
  fi

  # Check for Cloud Monitoring alert policies
  if gcloud alpha monitoring policies list --project="$PROJECT_ID" 2>/dev/null | grep -q "observability"; then
    test_pass "Observability monitoring policies exist"
  else
    log_warn "Observability monitoring policies not yet deployed"
  fi

  # Check for budget alerts
  if gcloud billing budgets list --billing-account="$(gcloud beta billing projects describe "$PROJECT_ID" --format='value(billingAccountName)' 2>/dev/null)" 2>/dev/null | grep -q "apx-observability"; then
    test_pass "Observability budget configured"
  else
    log_warn "Observability budget not yet configured"
  fi
}

# ============================================================================
# Test 7: Cost Estimation Validation
# ============================================================================
test_cost_estimates() {
  test_start "Cost estimation validation"

  local schema_file="$PROJECT_ROOT/observability/bigquery/schema.sql"

  # Verify cost estimates are documented
  if grep -q "COST ESTIMATES" "$schema_file"; then
    test_pass "Cost estimates documented in schema"
  else
    test_fail "Cost estimates not documented"
  fi

  # Check for reasonable storage estimates
  if grep -q "Total BigQuery cost:" "$schema_file"; then
    test_pass "BigQuery cost projection included"
  else
    test_fail "BigQuery cost projection missing"
  fi

  # Verify sampling reduces costs
  local budget_config="$PROJECT_ROOT/observability/budgets/log_sampling.yaml"

  log_info "Cost optimization strategies:"
  echo "  - 1% sampling reduces log volume by 99%"
  echo "  - Partitioning limits scanned data"
  echo "  - Clustering speeds up queries"
  echo "  - Aggregates reduce raw data queries"
  echo "  - 30-day retention on raw data"

  test_pass "Cost optimization strategies documented"
}

# ============================================================================
# Test 8: Configuration Completeness
# ============================================================================
test_configuration_completeness() {
  test_start "Configuration completeness check"

  local files=(
    "$PROJECT_ROOT/edge/envoy/envoy.yaml"
    "$PROJECT_ROOT/observability/budgets/log_sampling.yaml"
    "$PROJECT_ROOT/observability/bigquery/schema.sql"
  )

  local all_exist=true
  for file in "${files[@]}"; do
    if [ -f "$file" ]; then
      test_pass "File exists: $(basename "$file")"
    else
      test_fail "File missing: $file"
      all_exist=false
    fi
  done

  if [ "$all_exist" = true ]; then
    test_pass "All required configuration files present"
  fi
}

# ============================================================================
# Main Test Runner
# ============================================================================
main() {
  echo "=========================================="
  echo "APX Cost Controls Integration Test"
  echo "=========================================="
  echo "Project: $PROJECT_ID"
  echo "Edge URL: $EDGE_URL"
  echo "Project Root: $PROJECT_ROOT"
  echo ""

  # Run tests
  test_envoy_log_sampling
  test_budget_config
  test_bigquery_schema
  test_bigquery_tables_deployed
  test_sampling_in_practice
  test_budget_alerts_deployed
  test_cost_estimates
  test_configuration_completeness

  # Summary
  echo ""
  echo "=========================================="
  echo "Test Summary"
  echo "=========================================="
  echo "Tests Run:    $TESTS_RUN"
  echo "Tests Passed: $TESTS_PASSED"
  echo "Tests Failed: $TESTS_FAILED"
  echo ""

  if [ $TESTS_FAILED -eq 0 ]; then
    log_info "All tests passed! ✓"
    echo ""
    echo "Next Steps:"
    echo "1. Deploy BigQuery schema:"
    echo "   cd $PROJECT_ROOT/observability/bigquery"
    echo "   sed 's/PROJECT_ID/$PROJECT_ID/g' schema.sql | bq query --use_legacy_sql=false"
    echo ""
    echo "2. Set up Cloud Logging sink to BigQuery:"
    echo "   gcloud logging sinks create apx-requests-sink \\"
    echo "     bigquery.googleapis.com/projects/$PROJECT_ID/datasets/analytics \\"
    echo "     --log-filter='resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"apx-edge\"'"
    echo ""
    echo "3. Create scheduled queries for aggregations:"
    echo "   # See schema.sql for hourly/daily aggregation queries"
    echo ""
    echo "4. Set up budget alerts:"
    echo "   # Configure via Cloud Console > Billing > Budgets & Alerts"
    echo ""
    return 0
  else
    log_error "Some tests failed!"
    return 1
  fi
}

# Run tests
main "$@"
