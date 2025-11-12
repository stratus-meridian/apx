#!/bin/bash

set -e

echo "=========================================="
echo "Testing Redis-Based Rate Limiting"
echo "=========================================="
echo ""

# Test 1: Free tier rate limit (1 req/s, 10 burst)
echo "Test 1: Free tier rate limit (1 req/s sustained)..."
SUCCESS=0
RATE_LIMITED=0

for i in {1..15}; do
    RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/rl_response.json \
        http://localhost:8081/api/test \
        -H "X-Tenant-ID: free-tier-test" \
        -H "X-Tenant-Tier: free" \
        -H "Content-Type: application/json" \
        -d '{}' 2>/dev/null || echo "000")

    if [ "$RESPONSE" = "202" ]; then
        ((SUCCESS++))
    elif [ "$RESPONSE" = "429" ]; then
        ((RATE_LIMITED++))
    fi

    sleep 0.05  # 50ms between requests = 20 req/s (faster than allowed rate)
done

echo "Results: $SUCCESS allowed, $RATE_LIMITED blocked"
if [ $RATE_LIMITED -gt 0 ]; then
    echo "✅ Rate limiting working for free tier"
else
    echo "❌ Rate limiting not working: all requests allowed"
    exit 1
fi
echo ""

# Test 2: Pro tier rate limit (10 req/s, 100 burst)
echo "Test 2: Pro tier higher limit (10 req/s sustained)..."
SUCCESS=0
RATE_LIMITED=0

for i in {1..30}; do
    RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null \
        http://localhost:8081/api/test \
        -H "X-Tenant-ID: pro-tier-test" \
        -H "X-Tenant-Tier: pro" \
        -H "Content-Type: application/json" \
        -d '{}' 2>/dev/null || echo "000")

    if [ "$RESPONSE" = "202" ]; then
        ((SUCCESS++))
    elif [ "$RESPONSE" = "429" ]; then
        ((RATE_LIMITED++))
    fi

    sleep 0.05
done

echo "Results: $SUCCESS allowed, $RATE_LIMITED blocked"
if [ $SUCCESS -gt 15 ]; then
    echo "✅ Pro tier allows more requests: $SUCCESS/30"
else
    echo "⚠️  Pro tier may be limited: $SUCCESS/30"
fi
echo ""

# Test 3: Enterprise tier (100 req/s, 1000 burst)
echo "Test 3: Enterprise tier highest limit (100 req/s sustained)..."
SUCCESS=0

for i in {1..50}; do
    RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null \
        http://localhost:8081/api/test \
        -H "X-Tenant-ID: enterprise-tier-test" \
        -H "X-Tenant-Tier: enterprise" \
        -H "Content-Type: application/json" \
        -d '{}' 2>/dev/null || echo "000")

    if [ "$RESPONSE" = "202" ]; then
        ((SUCCESS++))
    fi

    sleep 0.01  # 10ms = 100 req/s
done

echo "Results: $SUCCESS/50 allowed"
if [ $SUCCESS -gt 40 ]; then
    echo "✅ Enterprise tier allows high throughput: $SUCCESS/50"
else
    echo "⚠️  Enterprise tier limited: $SUCCESS/50"
fi
echo ""

# Test 4: Check 429 response format
echo "Test 4: Verify 429 response format..."
# Exhaust free tier limit
for i in {1..15}; do
    curl -s http://localhost:8081/api/test \
        -H "X-Tenant-ID: response-test" \
        -H "X-Tenant-Tier: free" \
        -H "Content-Type: application/json" \
        -d '{}' > /dev/null 2>&1
done

# Should get 429 now
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8081/api/test \
    -H "X-Tenant-ID: response-test" \
    -H "X-Tenant-Tier: free" \
    -H "Content-Type: application/json" \
    -d '{}')

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "429" ]; then
    echo "✅ Returns 429 status code"

    if echo "$BODY" | grep -q "rate_limit_exceeded"; then
        echo "✅ Response contains error code"
    else
        echo "❌ Response missing error code"
        echo "Response: $BODY"
    fi

    if echo "$BODY" | grep -q "Too many requests"; then
        echo "✅ Response contains error message"
    else
        echo "❌ Response missing error message"
        echo "Response: $BODY"
    fi
else
    echo "⚠️  Expected 429, got $HTTP_CODE"
fi
echo ""

# Test 5: Check Redis keys
echo "Test 5: Redis rate limit keys..."
REDIS_KEYS=$(docker exec apilee-redis-1 redis-cli KEYS "apx:rl:*" 2>/dev/null || echo "")

if [ -n "$REDIS_KEYS" ]; then
    echo "✅ Rate limit keys in Redis:"
    echo "$REDIS_KEYS" | head -n 5
    KEY_COUNT=$(echo "$REDIS_KEYS" | wc -l | tr -d ' ')
    echo "Total keys: $KEY_COUNT"
else
    echo "❌ No rate limit keys found in Redis"
fi
echo ""

# Test 6: Token refill verification
echo "Test 6: Token refill verification..."
# Exhaust tokens
for i in {1..12}; do
    curl -s http://localhost:8081/api/test \
        -H "X-Tenant-ID: refill-test" \
        -H "X-Tenant-Tier: free" \
        -H "Content-Type: application/json" \
        -d '{}' > /dev/null 2>&1
done

# Should be rate limited now
RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null \
    http://localhost:8081/api/test \
    -H "X-Tenant-ID: refill-test" \
    -H "X-Tenant-Tier: free" \
    -H "Content-Type: application/json" \
    -d '{}')

if [ "$RESPONSE" = "429" ]; then
    echo "✅ Tokens exhausted (rate limited)"

    # Wait for 2 seconds (should refill 2 tokens for free tier)
    echo "Waiting 2 seconds for token refill..."
    sleep 2

    # Should succeed now
    RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null \
        http://localhost:8081/api/test \
        -H "X-Tenant-ID: refill-test" \
        -H "X-Tenant-Tier: free" \
        -H "Content-Type: application/json" \
        -d '{}')

    if [ "$RESPONSE" = "202" ]; then
        echo "✅ Token refill working (request allowed after wait)"
    else
        echo "⚠️  Token refill may not be working: got $RESPONSE"
    fi
else
    echo "⚠️  Tokens not exhausted as expected"
fi
echo ""

# Test 7: Tenant isolation
echo "Test 7: Tenant isolation verification..."
# Exhaust tenant A
for i in {1..12}; do
    curl -s http://localhost:8081/api/test \
        -H "X-Tenant-ID: tenant-a" \
        -H "X-Tenant-Tier: free" \
        -H "Content-Type: application/json" \
        -d '{}' > /dev/null 2>&1
done

# Tenant A should be limited
RESPONSE_A=$(curl -s -w "%{http_code}" -o /dev/null \
    http://localhost:8081/api/test \
    -H "X-Tenant-ID: tenant-a" \
    -H "X-Tenant-Tier: free" \
    -H "Content-Type: application/json" \
    -d '{}')

# Tenant B should still work
RESPONSE_B=$(curl -s -w "%{http_code}" -o /dev/null \
    http://localhost:8081/api/test \
    -H "X-Tenant-ID: tenant-b" \
    -H "X-Tenant-Tier: free" \
    -H "Content-Type: application/json" \
    -d '{}')

if [ "$RESPONSE_A" = "429" ] && [ "$RESPONSE_B" = "202" ]; then
    echo "✅ Tenant isolation working (A limited, B allowed)"
else
    echo "❌ Tenant isolation issue: A=$RESPONSE_A, B=$RESPONSE_B"
fi
echo ""

echo "=========================================="
echo "Rate Limiting Tests Complete"
echo "=========================================="
