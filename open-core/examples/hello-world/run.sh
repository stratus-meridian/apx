#!/bin/bash

# Start backend server
echo "Starting backend server on port 9000..."
go run backend.go &
BACKEND_PID=$!

# Wait for backend to start
sleep 2

# Start router
echo "Starting APX Router on port 8080..."
cd ../..
export ROUTES_CONFIG="/api/**=http://localhost:9000:sync"
export PORT=8080
export LOG_LEVEL=info
go run ./cmd/router &
ROUTER_PID=$!

echo ""
echo "✅ Services started!"
echo "   Backend: http://localhost:9000"
echo "   Router:  http://localhost:8080"
echo ""
echo "Try: curl http://localhost:8080/api/hello"
echo ""
echo "Press Ctrl+C to stop..."

# Wait for interrupt
trap "kill $BACKEND_PID $ROUTER_PID; exit" INT
wait
