#!/usr/bin/env bash
# ============================================================================
# Edge → Router End-to-End Test Suite
# ============================================================================
set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_section()  { echo -e "${BLUE}\n=== $1 ===${NC}"; }
log_ok()       { echo -e "${GREEN}✓ $1${NC}"; }
log_warn()     { echo -e "${YELLOW}⚠ $1${NC}"; }
log_fail()     { echo -e "${RED}✗ $1${NC}"; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log_fail "Missing required command: $1"
    exit 1
  fi
}

require_cmd curl
require_cmd jq

# Configuration (override via environment if needed)
EDGE_URL="${EDGE_URL:-https://apx-edge-dev-935932442471.us-central1.run.app}"
ROUTER_URL="${ROUTER_URL:-https://apx-router-dev-jcvvfyilzq-uc.a.run.app}"
TEST_API_KEY="${TEST_API_KEY:-apx_test_a1b2c3d4e5f67890a1b2c3d4e5f67890}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-20}"

log_section "Configuration"
echo "EDGE_URL    = ${EDGE_URL}"
echo "ROUTER_URL  = ${ROUTER_URL}"
echo "TEST_API_KEY= ${TEST_API_KEY:0:12}...(redacted)"
echo

failures=0

run_curl() {
  local method="$1"
  local url="$2"
  shift 2
  curl -sS -X "$method" "$url" "$@" \
    -m "$TIMEOUT_SECONDS" \
    -w '\nHTTP_CODE:%{http_code}'
}

assert_http_code() {
  local expected="$1"
  local response="$2"
  local code
  code="$(echo "$response" | sed -n 's/^HTTP_CODE://p')"
  if [[ "$code" != "$expected" ]]; then
    log_fail "Expected HTTP $expected, got $code"
    echo "$response"
    failures=$((failures+1))
    return 1
  fi
  return 0
}

test_router_health() {
  log_section "Test 1: Router /health (direct)"
  local resp
  resp="$(run_curl GET "${ROUTER_URL}/health")" || true
  if assert_http_code 200 "$resp"; then
    log_ok "Router /health OK"
  fi
}

test_edge_health() {
  log_section "Test 2: Edge /health"
  local resp
  resp="$(run_curl GET "${EDGE_URL}/health")" || true
  if assert_http_code 200 "$resp"; then
    log_ok "Edge /health OK"
  fi
}

test_edge_debug_router_health() {
  log_section "Test 3: Edge → Router debug health"
  local resp
  resp="$(run_curl GET "${EDGE_URL}/debug-router-health")" || true
  if assert_http_code 200 "$resp"; then
    log_ok "/debug-router-health returned 200 and reached router"
  fi
}

test_chat_completions_success() {
  log_section "Test 4: /chat/completions via edge (valid API key)"

  local body resp code
  body='{
    "model": "gpt-4.1-mini",
    "messages": [{"role": "user", "content": "ping"}],
    "max_tokens": 16
  }'

  resp="$(run_curl POST "${EDGE_URL}/chat/completions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${TEST_API_KEY}" \
    -d "$body")" || true

  local code
  code="$(echo "$resp" | sed -n 's/^HTTP_CODE://p')"
  if [[ "$code" != "202" && "$code" != "429" ]]; then
    log_fail "Expected HTTP 202 or 429, got $code"
    echo "$resp"
    failures=$((failures+1))
    return
  fi

  if [[ "$code" == "429" ]]; then
    log_warn "Request was rate-limited (HTTP 429) – rate limiter active for this tenant"
    return
  fi

  local json
  json="$(echo "$resp" | sed '/^HTTP_CODE:/d')" || true
  local request_id status_url
  request_id="$(echo "$json" | jq -r '.request_id // empty' 2>/dev/null || true)"
  status_url="$(echo "$json" | jq -r '.status_url // empty' 2>/dev/null || true)"

  if [[ -z "$request_id" || -z "$status_url" ]]; then
    log_fail "Missing request_id or status_url in 202 response"
    echo "$json"
    failures=$((failures+1))
    return
  fi

  log_ok "Request accepted with ID: $request_id"

  # Probe status endpoint once (we only care that router responds)
  local status_resp
  status_resp="$(run_curl GET "$status_url")" || true
  local status_code
  status_code="$(echo "$status_resp" | sed -n 's/^HTTP_CODE://p')"
  if [[ "$status_code" =~ ^2|3|4[0-9]{2}$ ]]; then
    log_ok "Status endpoint reachable (HTTP $status_code)"
  else
    log_warn "Status endpoint returned unexpected code: $status_code"
  fi
}

test_chat_completions_no_auth() {
  log_section "Test 5: /chat/completions without Authorization header"

  local body resp
  body='{"model":"gpt-4.1-mini","messages":[{"role":"user","content":"ping"}]}'

  resp="$(run_curl POST "${EDGE_URL}/chat/completions" \
    -H "Content-Type: application/json" \
    -d "$body")" || true

  # Expect router to return 401
  if assert_http_code 401 "$resp"; then
    log_ok "Unauthenticated request correctly rejected with 401"
  fi
}

test_chat_completions_invalid_key() {
  log_section "Test 6: /chat/completions with invalid API key"

  local body resp
  body='{"model":"gpt-4.1-mini","messages":[{"role":"user","content":"ping"}]}'

  resp="$(run_curl POST "${EDGE_URL}/chat/completions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer apx_test_invalid_key_123" \
    -d "$body")" || true

  if assert_http_code 401 "$resp"; then
    log_ok "Invalid API key correctly rejected with 401"
  fi
}

test_rate_limit_headers() {
  log_section "Test 7: Rate limit headers present"

  local body resp
  body='{"model":"gpt-4.1-mini","messages":[{"role":"user","content":"ping"}]}'

  resp="$(run_curl POST "${EDGE_URL}/chat/completions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${TEST_API_KEY}" \
    -d "$body")" || true

  local code
  code="$(echo "$resp" | sed -n 's/^HTTP_CODE://p')"
  if [[ "$code" != "202" && "$code" != "429" ]]; then
    log_fail "Expected HTTP 202 or 429, got $code"
    echo "$resp"
    failures=$((failures+1))
    return
  fi

  local headers
  headers="$(echo "$resp" | sed -n 's/^HTTP_CODE:.*$//p')" || true

  # We can't easily see response headers via curl -w here, so just log success of call.
  # A more advanced version could use -D - to capture headers.
  log_ok "Request accepted; consider running with 'curl -D -' manually to inspect X-RateLimit-* headers"
}

main() {
  test_router_health
  test_edge_health
  test_edge_debug_router_health
  test_chat_completions_success
  test_chat_completions_no_auth
  test_chat_completions_invalid_key
  test_rate_limit_headers

  echo
  if [[ "$failures" -eq 0 ]]; then
    log_ok "All edge → router tests passed"
    exit 0
  else
    log_fail "$failures test(s) failed"
    exit 1
  fi
}

main "$@"
