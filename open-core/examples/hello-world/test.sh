#!/bin/bash

echo "Testing APX Router Open-Core..."
echo ""

# Test 1: Basic request
echo "1️⃣  Basic Request:"
curl -s http://localhost:8080/api/hello | jq .
echo ""

# Test 2: Health check
echo "2️⃣  Health Check:"
curl -s http://localhost:8080/health | jq .
echo ""

# Test 3: With API key
echo "3️⃣  Request with API Key:"
curl -s -H "Authorization: Bearer demo-key-123" \
  http://localhost:8080/api/hello | jq .
echo ""

# Test 4: Rate limiting (send 12 requests to trigger limit)
echo "4️⃣  Rate Limiting Test (12 requests):"
for i in {1..12}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/test)
  if [ "$STATUS" = "429" ]; then
    echo "   Request $i: ❌ Rate limited (HTTP 429)"
  else
    echo "   Request $i: ✅ Allowed (HTTP $STATUS)"
  fi
done
echo ""

# Test 5: Metrics endpoint
echo "5️⃣  Metrics (sample):"
curl -s http://localhost:8080/metrics | grep "apx_" | head -5
echo "   ..."
echo ""

echo "✅ Tests complete!"
