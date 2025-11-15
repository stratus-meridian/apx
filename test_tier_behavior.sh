#!/bin/bash
# ============================================================================
# Tier Behavior Black-Box Test Script
# Tests the complete tier → runtime mapping end-to-end
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test configuration
PROVISIONING_API_URL="http://localhost:8080"
TEST_CUSTOMER_ID="test-customer-123"
AUTH_TOKEN="test-auth-token"

# Helper functions
log_test() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

log_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

log_error() {
    echo -e "${RED}✗ $1${NC}"
}

log_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Test 1: Free Tier - Shared Cloud Run (Valid)
test_free_tier_shared() {
    log_test "Test 1: Free Tier - Shared Cloud Run"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 1,
                    "machine_type": "n1-standard-1",
                    "disk_size_gb": 50,
                    "min_nodes": 1,
                    "max_nodes": 3
                },
                "redis": {
                    "memory_gb": 1,
                    "version": "6.2",
                    "tier": "BASIC"
                },
                "network": {
                    "vpc_name": "apx-shared-vpc",
                    "subnet_cidr": "10.0.0.0/24",
                    "enable_nat": true
                }
            },
            "tenant_tier": "free",
            "isolation_mode": "shared",
            "runtime": "cloudrun"
        }')
    
    if echo "$response" | grep -q '"deployment_id"'; then
        local deployment_id=$(echo "$response" | jq -r '.deployment_id')
        log_success "Free tier deployment created: $deployment_id"
        
        # Check decision snapshot
        local decision=$(echo "$response" | jq -r '.tier_decision // empty')
        if [ -n "$decision" ]; then
            log_info "Tier decision recorded: $(echo "$decision" | jq -c '.effective')"
        fi
    else
        log_error "Free tier deployment failed: $response"
        return 1
    fi
}

# Test 2: Free Tier - Invalid Dedicated Request (Should Fail)
test_free_tier_dedicated_rejection() {
    log_test "Test 2: Free Tier - Invalid Dedicated Request (Should Be Rejected)"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 3,
                    "machine_type": "n1-standard-4",
                    "disk_size_gb": 200,
                    "min_nodes": 3,
                    "max_nodes": 20
                },
                "redis": {
                    "memory_gb": 8,
                    "version": "6.2",
                    "tier": "STANDARD_HA"
                },
                "network": {
                    "vpc_name": "invalid-dedicated-vpc",
                    "subnet_cidr": "10.3.0.0/16"
                }
            },
            "tenant_tier": "free",
            "isolation_mode": "dedicated",
            "runtime": "gke"
        }')
    
    if echo "$response" | grep -q "free tier cannot use dedicated isolation mode"; then
        log_success "Correctly rejected free tier dedicated request"
    else
        log_error "Should have rejected free tier dedicated request: $response"
        return 1
    fi
}

# Test 3: Pro Tier - Namespace Isolation (Valid)
test_pro_tier_namespace() {
    log_test "Test 3: Pro Tier - Namespace Isolation"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 2,
                    "machine_type": "n1-standard-2",
                    "disk_size_gb": 100,
                    "min_nodes": 2,
                    "max_nodes": 10
                },
                "redis": {
                    "memory_gb": 2,
                    "version": "6.2",
                    "tier": "STANDARD_HA"
                },
                "network": {
                    "vpc_name": "apx-namespace-vpc",
                    "subnet_cidr": "10.1.0.0/24",
                    "enable_nat": true
                }
            },
            "tenant_tier": "pro",
            "isolation_mode": "namespace",
            "runtime": "cloudrun"
        }')
    
    if echo "$response" | grep -q '"deployment_id"'; then
        local deployment_id=$(echo "$response" | jq -r '.deployment_id')
        log_success "Pro tier deployment created: $deployment_id"
    else
        log_error "Pro tier deployment failed: $response"
        return 1
    fi
}

# Test 4: Pro Tier - Invalid Dedicated Request (Should Fail)
test_pro_tier_dedicated_rejection() {
    log_test "Test 4: Pro Tier - Invalid Dedicated Request (Should Be Rejected)"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 3,
                    "machine_type": "n1-standard-4",
                    "disk_size_gb": 200,
                    "min_nodes": 3,
                    "max_nodes": 20
                },
                "redis": {
                    "memory_gb": 8,
                    "version": "6.2",
                    "tier": "STANDARD_HA"
                },
                "network": {
                    "vpc_name": "invalid-dedicated-vpc",
                    "subnet_cidr": "10.4.0.0/16"
                }
            },
            "tenant_tier": "pro",
            "isolation_mode": "dedicated",
            "runtime": "gke"
        }')
    
    if echo "$response" | grep -q "pro tier cannot use dedicated isolation mode"; then
        log_success "Correctly rejected pro tier dedicated request"
    else
        log_error "Should have rejected pro tier dedicated request: $response"
        return 1
    fi
}

# Test 5: Enterprise Tier - Dedicated GKE (Valid)
test_enterprise_tier_dedicated() {
    log_test "Test 5: Enterprise Tier - Dedicated GKE"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 3,
                    "machine_type": "n1-standard-4",
                    "disk_size_gb": 200,
                    "min_nodes": 3,
                    "max_nodes": 20
                },
                "redis": {
                    "memory_gb": 8,
                    "version": "6.2",
                    "tier": "STANDARD_HA",
                    "replica_count": 2
                },
                "network": {
                    "vpc_name": "acme-dedicated-vpc",
                    "subnet_cidr": "10.2.0.0/16",
                    "enable_nat": true,
                    "allowed_ip_ranges": ["10.0.0.0/8"]
                }
            },
            "tenant_tier": "enterprise",
            "isolation_mode": "dedicated",
            "runtime": "gke"
        }')
    
    if echo "$response" | grep -q '"deployment_id"'; then
        local deployment_id=$(echo "$response" | jq -r '.deployment_id')
        log_success "Enterprise tier deployment created: $deployment_id"
        
        # Check decision snapshot
        local decision=$(echo "$response" | jq -r '.tier_decision // empty')
        if [ -n "$decision" ]; then
            log_info "Enterprise tier decision: $(echo "$decision" | jq -c '.effective')"
        fi
    else
        log_error "Enterprise tier deployment failed: $response"
        return 1
    fi
}

# Test 6: Enterprise Tier - Defaults Applied
test_enterprise_tier_defaults() {
    log_test "Test 6: Enterprise Tier - Defaults Applied (No explicit runtime/isolation)"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 3,
                    "machine_type": "n1-standard-4",
                    "disk_size_gb": 200,
                    "min_nodes": 3,
                    "max_nodes": 20
                },
                "redis": {
                    "memory_gb": 8,
                    "version": "6.2",
                    "tier": "STANDARD_HA"
                },
                "network": {
                    "vpc_name": "enterprise-default-vpc",
                    "subnet_cidr": "10.5.0.0/16"
                }
            },
            "tenant_tier": "enterprise"
        }')
    
    if echo "$response" | grep -q '"deployment_id"'; then
        local deployment_id=$(echo "$response" | jq -r '.deployment_id')
        log_success "Enterprise tier deployment with defaults created: $deployment_id"
        
        # Check that defaults were applied
        local decision=$(echo "$response" | jq -r '.tier_decision // empty')
        if [ -n "$decision" ]; then
            local effective_isolation=$(echo "$decision" | jq -r '.effective.isolation_mode')
            local effective_runtime=$(echo "$decision" | jq -r '.effective.runtime')
            
            if [ "$effective_isolation" = "namespace" ] && [ "$effective_runtime" = "gke" ]; then
                log_success "Correct defaults applied: namespace + gke"
            else
                log_error "Incorrect defaults applied: $effective_isolation + $effective_runtime"
                return 1
            fi
        fi
    else
        log_error "Enterprise tier deployment failed: $response"
        return 1
    fi
}

# Test 7: Tier Mismatch Validation
test_tier_mismatch() {
    log_test "Test 7: Tier Mismatch Validation (Customer is Pro but requests Enterprise)"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 3,
                    "machine_type": "n1-standard-4",
                    "disk_size_gb": 200,
                    "min_nodes": 3,
                    "max_nodes": 20
                },
                "redis": {
                    "memory_gb": 8,
                    "version": "6.2",
                    "tier": "STANDARD_HA"
                },
                "network": {
                    "vpc_name": "tier-mismatch-vpc",
                    "subnet_cidr": "10.6.0.0/16"
                }
            },
            "tenant_tier": "enterprise"
        }')
    
    if echo "$response" | grep -q "does not match customer's actual tier"; then
        log_success "Correctly rejected tier mismatch"
    else
        log_error "Should have rejected tier mismatch: $response"
        return 1
    fi
}

# Test 7: Pro Tier - GKE Not Allowed (Should Fail)
test_pro_tier_gke_rejection() {
    log_test "Test 7: Pro Tier - GKE Not Allowed (Should Be Rejected)"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 2,
                    "machine_type": "n1-standard-2",
                    "disk_size_gb": 100,
                    "min_nodes": 2,
                    "max_nodes": 10
                },
                "redis": {
                    "memory_gb": 2,
                    "version": "6.2",
                    "tier": "STANDARD_HA"
                },
                "network": {
                    "vpc_name": "pro-gke-vpc",
                    "subnet_cidr": "10.7.0.0/24",
                    "enable_nat": true
                }
            },
            "tenant_tier": "pro",
            "isolation_mode": "namespace",
            "runtime": "gke"
        }')
    
    if echo "$response" | grep -q "pro tier cannot use gke runtime"; then
        log_success "Correctly rejected pro tier GKE runtime request"
    else
        log_error "Should have rejected pro tier GKE runtime: $response"
        return 1
    fi
}

# Test 8: Free Tier - Defaults Applied
test_free_tier_defaults() {
    log_test "Test 8: Free Tier - Defaults Applied (No explicit runtime/isolation)"
    
    local response=$(curl -s -X POST "$PROVISIONING_API_URL/v1/provisioning" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d '{
            "customer_id": "'$TEST_CUSTOMER_ID'",
            "region": "us-central1",
            "environment": "prod",
            "infrastructure_config": {
                "gke": {
                    "node_count": 1,
                    "machine_type": "n1-standard-1",
                    "disk_size_gb": 50,
                    "min_nodes": 1,
                    "max_nodes": 3
                },
                "redis": {
                    "memory_gb": 1,
                    "version": "6.2",
                    "tier": "BASIC"
                },
                "network": {
                    "vpc_name": "free-defaults-vpc",
                    "subnet_cidr": "10.8.0.0/24",
                    "enable_nat": false
                }
            },
            "tenant_tier": "free"
        }')
    
    if echo "$response" | grep -q '"deployment_id"'; then
        local deployment_id=$(echo "$response" | jq -r '.deployment_id')
        log_success "Free tier deployment with defaults created: $deployment_id"
        
        # Check that defaults were applied
        local decision=$(echo "$response" | jq -r '.tier_decision // empty')
        if [ -n "$decision" ]; then
            local effective_isolation=$(echo "$decision" | jq -r '.effective.isolation_mode')
            local effective_runtime=$(echo "$decision" | jq -r '.effective.runtime')
            
            if [ "$effective_isolation" = "shared" ] && [ "$effective_runtime" = "cloudrun" ]; then
                log_success "Correct defaults applied: shared + cloudrun"
            else
                log_error "Incorrect defaults applied: $effective_isolation + $effective_runtime"
                return 1
            fi
        fi
    else
        log_error "Free tier deployment failed: $response"
        return 1
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}🧪 APX Tier Behavior Black-Box Tests${NC}"
    echo -e "${BLUE}Testing tier → runtime mapping end-to-end${NC}"
    echo ""
    
    # Check if provisioning API is running
    if ! curl -s "$PROVISIONING_API_URL/health" > /dev/null 2>&1; then
        log_error "Provisioning API is not running at $PROVISIONING_API_URL"
        echo "Please start the provisioning API first: cd .private/provisioning-api && go run cmd/server/main.go"
        exit 1
    fi
    
    local passed=0
    local failed=0
    
    # Run all tests
    if test_free_tier_shared; then ((passed++)); else ((failed++)); fi
    if test_free_tier_dedicated_rejection; then ((passed++)); else ((failed++)); fi
    if test_pro_tier_namespace; then ((passed++)); else ((failed++)); fi
    if test_pro_tier_dedicated_rejection; then ((passed++)); else ((failed++)); fi
    if test_enterprise_tier_dedicated; then ((passed++)); else ((failed++)); fi
    if test_enterprise_tier_defaults; then ((passed++)); else ((failed++)); fi
    if test_tier_mismatch; then ((passed++)); else ((failed++)); fi
    
    echo ""
    echo -e "${BLUE}📊 Test Results:${NC}"
    echo -e "${GREEN}Passed: $passed${NC}"
    echo -e "${RED}Failed: $failed${NC}"
    
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}🎉 All tier behavior tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some tests failed${NC}"
        exit 1
    fi
}

# Run tests
main "$@"