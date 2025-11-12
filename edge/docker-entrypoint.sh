#!/bin/sh
# Entrypoint script for Edge Gateway on Cloud Run
# Substitutes environment variables in Envoy config and starts Envoy

set -e

# Default values if not set
ROUTER_HOST=${ROUTER_HOST:-"router-placeholder.example.com"}
ROUTER_URL=${ROUTER_URL:-"https://${ROUTER_HOST}"}

echo "Starting APX Edge Gateway..."
echo "Router Host: ${ROUTER_HOST}"
echo "Router URL: ${ROUTER_URL}"

# Substitute environment variables in config
envsubst '${ROUTER_HOST}' < /etc/envoy/envoy-cloud.yaml > /tmp/envoy-runtime.yaml

echo "Envoy configuration prepared. Starting Envoy..."

# Start Envoy
exec /usr/local/bin/envoy -c /tmp/envoy-runtime.yaml --service-cluster apx-edge "$@"
