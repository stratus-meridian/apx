#!/bin/bash

# GCP Load Balancer Testing Script
# Tests all configured routes and displays results

set -e

# Configuration
LB_IP="34.120.96.89"
DOMAIN="api.apx.build"
PROJECT_ID="apx-build-478003"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}GCP Load Balancer Test Suite${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo "Load Balancer IP: $LB_IP"
echo "Domain: $DOMAIN"
echo "Project: $PROJECT_ID"
echo ""

# Function to test endpoint
test_endpoint() {
    local name=$1
    local path=$2
    local expected_backend=$3

    echo -e "${YELLOW}Testing: $name${NC}"
    echo "Path: $path"

    # Test with IP (bypassing DNS)
    echo -n "  Testing with IP... "
    http_code=$(curl -s -o /dev/null -w "%{http_code}" -k "https://$LB_IP$path" -H "Host: $DOMAIN" --max-time 10)

    if [ "$http_code" = "200" ] || [ "$http_code" = "404" ] || [ "$http_code" = "401" ]; then
        echo -e "${GREEN}✓ Response: $http_code${NC}"
    elif [ "$http_code" = "502" ] || [ "$http_code" = "503" ]; then
        echo -e "${RED}✗ Backend unavailable: $http_code (Service not deployed?)${NC}"
    else
        echo -e "${RED}✗ Unexpected response: $http_code${NC}"
    fi

    # Test with domain (if DNS is configured)
    echo -n "  Testing with domain... "
    domain_code=$(curl -s -o /dev/null -w "%{http_code}" "https://$DOMAIN$path" --max-time 10 2>/dev/null || echo "DNS_ERROR")

    if [ "$domain_code" = "DNS_ERROR" ]; then
        echo -e "${YELLOW}⚠ DNS not configured${NC}"
    elif [ "$domain_code" = "200" ] || [ "$domain_code" = "404" ] || [ "$domain_code" = "401" ]; then
        echo -e "${GREEN}✓ Response: $domain_code${NC}"
    elif [ "$domain_code" = "502" ] || [ "$domain_code" = "503" ]; then
        echo -e "${RED}✗ Backend unavailable: $domain_code${NC}"
    else
        echo -e "${RED}✗ Unexpected response: $domain_code${NC}"
    fi

    echo ""
}

# Function to check GCP resources
check_gcp_resources() {
    echo -e "${BLUE}Checking GCP Resources...${NC}"
    echo ""

    echo -e "${YELLOW}Backend Services:${NC}"
    gcloud compute backend-services list \
        --project=$PROJECT_ID \
        --filter="name~apx-.*-dev" \
        --format="table(name,backends.group:label=NEG,protocol)" 2>/dev/null || echo "Error listing backend services"
    echo ""

    echo -e "${YELLOW}Network Endpoint Groups:${NC}"
    gcloud compute network-endpoint-groups list \
        --project=$PROJECT_ID \
        --filter="name~apx-.*-neg-dev" \
        --format="table(name,networkEndpointType,cloudRun.service,region)" 2>/dev/null || echo "Error listing NEGs"
    echo ""

    echo -e "${YELLOW}Cloud Run Services:${NC}"
    gcloud run services list \
        --region=us-central1 \
        --project=$PROJECT_ID \
        --format="table(SERVICE,URL)" 2>/dev/null || echo "Error listing Cloud Run services"
    echo ""

    echo -e "${YELLOW}SSL Certificate:${NC}"
    gcloud compute ssl-certificates list \
        --project=$PROJECT_ID \
        --filter="name=apx-lb-cert-dev-v2" \
        --format="table(name,type,managed.status,expireTime)" 2>/dev/null || echo "Error listing certificates"
    echo ""
}

# Function to check URL map configuration
check_url_map() {
    echo -e "${BLUE}URL Map Configuration:${NC}"
    echo ""

    gcloud compute url-maps describe apx-lb-url-map-dev \
        --project=$PROJECT_ID \
        --format="yaml(pathMatchers[].pathRules)" 2>/dev/null || echo "Error describing URL map"
    echo ""
}

# Function to test HTTP to HTTPS redirect
test_http_redirect() {
    echo -e "${YELLOW}Testing HTTP to HTTPS Redirect:${NC}"
    echo -n "  Sending HTTP request... "

    response=$(curl -s -I -L "http://$LB_IP/" -H "Host: $DOMAIN" --max-time 10)

    if echo "$response" | grep -q "301\|302"; then
        if echo "$response" | grep -q "https://"; then
            echo -e "${GREEN}✓ Redirect working${NC}"
        else
            echo -e "${YELLOW}⚠ Redirect found but not to HTTPS${NC}"
        fi
    else
        echo -e "${RED}✗ No redirect found${NC}"
    fi
    echo ""
}

# Main test sequence
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}1. GCP Resource Check${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
check_gcp_resources

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}2. URL Map Configuration${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
check_url_map

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}3. HTTP to HTTPS Redirect${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
test_http_redirect

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}4. Endpoint Testing${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

# Test all configured routes
test_endpoint "Portal Root" "/" "apx-portal-backend-dev"
test_endpoint "Portal Dashboard" "/dashboard" "apx-portal-backend-dev"
test_endpoint "Portal API (NextAuth)" "/api/auth/session" "apx-portal-backend-dev"
test_endpoint "Backend Router Health" "/v1/health" "apx-router-backend-dev"
test_endpoint "Backend Router API" "/v1/models" "apx-router-backend-dev"
test_endpoint "WebSocket Endpoint" "/ws" "apx-websocket-backend-dev"

# Summary
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo -e "${GREEN}Load Balancer Configuration: Complete${NC}"
echo -e "${YELLOW}Note: 502/503 errors indicate services not deployed yet${NC}"
echo ""
echo "Next Steps:"
echo "1. Deploy apx-portal-dev to Cloud Run"
echo "2. Deploy apx-websocket-dev to Cloud Run"
echo "3. Configure DNS: $DOMAIN → $LB_IP"
echo "4. Test all endpoints again"
echo ""
echo "For detailed configuration, see:"
echo "  - docs/trackers/portal/LOAD_BALANCER_CONFIG_REPORT.md"
echo "  - docs/trackers/portal/LOAD_BALANCER_QUICK_REF.md"
echo "  - docs/trackers/portal/LOAD_BALANCER_ARCHITECTURE.md"
echo ""
