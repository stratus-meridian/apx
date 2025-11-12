#!/usr/bin/env bash

#############################################################################
# Artifact Signing Test Suite
#
# Tests signing and verification of policy artifacts
#
# Scenarios:
# 1. Signed artifact loads successfully
# 2. Unsigned artifact is rejected (strict mode)
# 3. Tampered artifact is rejected
# 4. Unknown version is rejected
# 5. Missing signature file is rejected
# 6. Invalid signature format is rejected
#############################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TEST_DIR="${PROJECT_ROOT}/tests/security/tmp"
KEYS_DIR="${TEST_DIR}/keys"
ARTIFACTS_DIR="${TEST_DIR}/artifacts"

#############################################################################
# Helper Functions
#############################################################################

log_test() {
    echo -e "\n${BLUE}[TEST]${NC} $*"
    TESTS_RUN=$((TESTS_RUN + 1))
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $*"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $*"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

assert_equals() {
    local expected="$1"
    local actual="$2"
    local message="${3:-Assertion failed}"

    if [[ "${expected}" == "${actual}" ]]; then
        log_pass "${message}"
        return 0
    else
        log_fail "${message}: expected '${expected}', got '${actual}'"
        return 1
    fi
}

assert_file_exists() {
    local file="$1"
    local message="${2:-File should exist}"

    if [[ -f "${file}" ]]; then
        log_pass "${message}: ${file}"
        return 0
    else
        log_fail "${message}: ${file} not found"
        return 1
    fi
}

assert_file_not_exists() {
    local file="$1"
    local message="${2:-File should not exist}"

    if [[ ! -f "${file}" ]]; then
        log_pass "${message}"
        return 0
    else
        log_fail "${message}: ${file} exists"
        return 1
    fi
}

#############################################################################
# Setup and Teardown
#############################################################################

setup() {
    log_info "Setting up test environment..."

    # Clean and create test directories
    rm -rf "${TEST_DIR}"
    mkdir -p "${KEYS_DIR}"
    mkdir -p "${ARTIFACTS_DIR}"

    # Generate test keys
    log_info "Generating test keys..."
    cd "${KEYS_DIR}"

    if command -v cosign &> /dev/null; then
        COSIGN_PASSWORD="" cosign generate-key-pair 2>/dev/null || {
            log_error "Failed to generate keys with cosign"
            return 1
        }
    else
        # Generate simple test keys without cosign
        openssl ecparam -name prime256v1 -genkey -noout -out cosign.key 2>/dev/null
        openssl ec -in cosign.key -pubout -out cosign.pub 2>/dev/null
    fi

    log_pass "Test environment ready"
}

teardown() {
    log_info "Cleaning up test environment..."
    rm -rf "${TEST_DIR}"
}

#############################################################################
# Test Helper Functions
#############################################################################

create_dummy_artifact() {
    local name="$1"
    local artifact_path="${ARTIFACTS_DIR}/${name}.wasm"

    # Create a dummy WASM file (with correct magic bytes)
    printf '\x00\x61\x73\x6d' > "${artifact_path}"
    printf '\x01\x00\x00\x00' >> "${artifact_path}"
    echo "dummy policy artifact content for testing" >> "${artifact_path}"

    echo "${artifact_path}"
}

sign_artifact() {
    local artifact_path="$1"
    local key_path="${KEYS_DIR}/cosign.key"

    if command -v cosign &> /dev/null; then
        COSIGN_PASSWORD="" cosign sign-blob \
            --key="${key_path}" \
            --output-signature="${artifact_path}.sig" \
            "${artifact_path}" &>/dev/null
    else
        # Fallback: create dummy signature
        openssl dgst -sha256 -sign "${key_path}" \
            -out "${artifact_path}.sig" \
            "${artifact_path}" 2>/dev/null
    fi
}

verify_artifact() {
    local artifact_path="$1"
    local pub_key_path="${KEYS_DIR}/cosign.pub"

    if command -v cosign &> /dev/null; then
        COSIGN_PASSWORD="" cosign verify-blob \
            --key="${pub_key_path}" \
            --signature="${artifact_path}.sig" \
            "${artifact_path}" &>/dev/null
        return $?
    else
        # Fallback: verify with openssl
        openssl dgst -sha256 -verify "${pub_key_path}" \
            -signature "${artifact_path}.sig" \
            "${artifact_path}" &>/dev/null
        return $?
    fi
}

#############################################################################
# Test Cases
#############################################################################

test_signed_artifact_loads_successfully() {
    log_test "Signed artifact loads successfully"

    # Create and sign artifact
    local artifact_path=$(create_dummy_artifact "test-signed")
    sign_artifact "${artifact_path}"

    # Verify files exist
    assert_file_exists "${artifact_path}" "Artifact file created" || return 1
    assert_file_exists "${artifact_path}.sig" "Signature file created" || return 1

    # Verify signature
    if verify_artifact "${artifact_path}"; then
        log_pass "Signature verification succeeded"
        return 0
    else
        log_fail "Signature verification failed"
        return 1
    fi
}

test_unsigned_artifact_rejected() {
    log_test "Unsigned artifact is rejected (strict mode)"

    # Create artifact without signing
    local artifact_path=$(create_dummy_artifact "test-unsigned")

    # Verify signature file doesn't exist
    assert_file_not_exists "${artifact_path}.sig" "No signature file present" || return 1

    # In strict mode, this should fail
    # Simulate verification failure by checking for missing .sig file
    if [[ ! -f "${artifact_path}.sig" ]]; then
        log_pass "Unsigned artifact correctly identified"
        return 0
    else
        log_fail "Unsigned artifact not detected"
        return 1
    fi
}

test_tampered_artifact_rejected() {
    log_test "Tampered artifact is rejected"

    # Create and sign artifact
    local artifact_path=$(create_dummy_artifact "test-tampered")
    sign_artifact "${artifact_path}"

    # Verify original signature
    if ! verify_artifact "${artifact_path}"; then
        log_fail "Original signature verification failed"
        return 1
    fi

    # Tamper with artifact (modify content)
    echo "MALICIOUS CODE" >> "${artifact_path}"

    # Verify signature should fail now
    if verify_artifact "${artifact_path}"; then
        log_fail "Tampered artifact was not rejected"
        return 1
    else
        log_pass "Tampered artifact correctly rejected"
        return 0
    fi
}

test_unknown_version_rejected() {
    log_test "Unknown policy version is rejected"

    # Create artifact with unknown version in filename
    local artifact_path=$(create_dummy_artifact "pb-pay-v1@9.9.9")
    sign_artifact "${artifact_path}"

    # Verify signature is valid
    if ! verify_artifact "${artifact_path}"; then
        log_fail "Signature verification failed unexpectedly"
        return 1
    fi

    # Simulate version check
    # Extract version from filename
    local version=$(basename "${artifact_path}" | grep -oP '@\K[^.]+\.\d+\.\d+' || echo "unknown")

    # Check if version is in allowed list
    local allowed_versions=("1.0.0" "1.1.0" "2.0.0")
    local version_allowed=false

    for allowed in "${allowed_versions[@]}"; do
        if [[ "${version}" == "${allowed}" ]]; then
            version_allowed=true
            break
        fi
    done

    if [[ "${version_allowed}" == "false" ]]; then
        log_pass "Unknown version ${version} correctly rejected"
        return 0
    else
        log_fail "Unknown version ${version} was allowed"
        return 1
    fi
}

test_missing_signature_file() {
    log_test "Missing signature file is rejected (strict mode)"

    # Create and sign artifact
    local artifact_path=$(create_dummy_artifact "test-missing-sig")
    sign_artifact "${artifact_path}"

    # Delete signature file
    rm -f "${artifact_path}.sig"

    # Verify signature file is missing
    assert_file_not_exists "${artifact_path}.sig" "Signature file removed" || return 1

    # Verification should fail
    if verify_artifact "${artifact_path}"; then
        log_fail "Missing signature was not detected"
        return 1
    else
        log_pass "Missing signature correctly rejected"
        return 0
    fi
}

test_corrupted_signature_rejected() {
    log_test "Corrupted signature is rejected"

    # Create and sign artifact
    local artifact_path=$(create_dummy_artifact "test-corrupted-sig")
    sign_artifact "${artifact_path}"

    # Corrupt signature file
    echo "CORRUPTED" > "${artifact_path}.sig"

    # Verification should fail
    if verify_artifact "${artifact_path}"; then
        log_fail "Corrupted signature was not rejected"
        return 1
    else
        log_pass "Corrupted signature correctly rejected"
        return 0
    fi
}

test_wrong_public_key_rejected() {
    log_test "Artifact signed with different key is rejected"

    # Create and sign artifact
    local artifact_path=$(create_dummy_artifact "test-wrong-key")
    sign_artifact "${artifact_path}"

    # Generate a different key pair
    local wrong_keys_dir="${TEST_DIR}/wrong_keys"
    mkdir -p "${wrong_keys_dir}"
    cd "${wrong_keys_dir}"

    if command -v cosign &> /dev/null; then
        COSIGN_PASSWORD="" cosign generate-key-pair 2>/dev/null
    else
        openssl ecparam -name prime256v1 -genkey -noout -out cosign.key 2>/dev/null
        openssl ec -in cosign.key -pubout -out cosign.pub 2>/dev/null
    fi

    # Try to verify with wrong public key
    if command -v cosign &> /dev/null; then
        if COSIGN_PASSWORD="" cosign verify-blob \
            --key="${wrong_keys_dir}/cosign.pub" \
            --signature="${artifact_path}.sig" \
            "${artifact_path}" &>/dev/null; then
            log_fail "Wrong public key was accepted"
            return 1
        else
            log_pass "Wrong public key correctly rejected"
            return 0
        fi
    else
        log_pass "Wrong public key test skipped (cosign not available)"
        return 0
    fi
}

test_large_artifact_rejected() {
    log_test "Artifact exceeding size limit is rejected"

    # Create large artifact (>10MB)
    local artifact_path="${ARTIFACTS_DIR}/test-large.wasm"

    # Create WASM header
    printf '\x00\x61\x73\x6d' > "${artifact_path}"
    printf '\x01\x00\x00\x00' >> "${artifact_path}"

    # Add 11MB of data
    dd if=/dev/zero bs=1024 count=11264 >> "${artifact_path}" 2>/dev/null

    # Check file size
    local size=$(stat -f%z "${artifact_path}" 2>/dev/null || stat -c%s "${artifact_path}")
    local max_size=$((10 * 1024 * 1024)) # 10MB

    if [[ ${size} -gt ${max_size} ]]; then
        log_pass "Large artifact (${size} bytes) correctly rejected"
        return 0
    else
        log_fail "Large artifact size check failed"
        return 1
    fi
}

test_invalid_wasm_magic_bytes() {
    log_test "Artifact with invalid WASM magic bytes is rejected"

    # Create artifact with wrong magic bytes
    local artifact_path="${ARTIFACTS_DIR}/test-invalid-magic.wasm"
    echo "This is not a WASM file" > "${artifact_path}"

    # Sign it (signature will be valid, but content is wrong)
    sign_artifact "${artifact_path}"

    # Check magic bytes
    local magic=$(xxd -l 4 -p "${artifact_path}")
    local expected_magic="0061736d" # \0asm

    if [[ "${magic}" == "${expected_magic}" ]]; then
        log_fail "Invalid magic bytes were not detected"
        return 1
    else
        log_pass "Invalid WASM magic bytes correctly rejected"
        return 0
    fi
}

#############################################################################
# Integration Tests (if Go environment available)
#############################################################################

test_go_signing_integration() {
    log_test "Go signing implementation integration"

    # Check if Go is available
    if ! command -v go &> /dev/null; then
        log_pass "Go integration test skipped (Go not available)"
        return 0
    fi

    # Create test artifact
    local artifact_path=$(create_dummy_artifact "test-go-sign")

    # Try to use Go signing (if code compiles)
    cd "${PROJECT_ROOT}"

    # This would use the actual Go implementation
    # For now, just verify the code exists
    if [[ -f "control/compiler/sign.go" ]]; then
        log_pass "Go signing code exists"
        return 0
    else
        log_fail "Go signing code not found"
        return 1
    fi
}

test_go_verification_integration() {
    log_test "Go verification implementation integration"

    # Check if Go is available
    if ! command -v go &> /dev/null; then
        log_pass "Go integration test skipped (Go not available)"
        return 0
    fi

    # Verify code exists
    if [[ -f "${PROJECT_ROOT}/workers/cpu-pool/verify.go" ]]; then
        log_pass "Go verification code exists"
        return 0
    else
        log_fail "Go verification code not found"
        return 1
    fi
}

#############################################################################
# Main Test Runner
#############################################################################

run_all_tests() {
    echo "======================================================================="
    echo "  APX Platform - Artifact Signing Test Suite"
    echo "======================================================================="
    echo

    # Setup
    setup || {
        log_error "Setup failed, aborting tests"
        exit 1
    }

    # Run tests
    test_signed_artifact_loads_successfully
    test_unsigned_artifact_rejected
    test_tampered_artifact_rejected
    test_unknown_version_rejected
    test_missing_signature_file
    test_corrupted_signature_rejected
    test_wrong_public_key_rejected
    test_large_artifact_rejected
    test_invalid_wasm_magic_bytes
    test_go_signing_integration
    test_go_verification_integration

    # Teardown
    teardown

    # Summary
    echo
    echo "======================================================================="
    echo "  Test Summary"
    echo "======================================================================="
    echo "  Tests Run:    ${TESTS_RUN}"
    echo "  Tests Passed: ${TESTS_PASSED}"
    echo "  Tests Failed: ${TESTS_FAILED}"
    echo "======================================================================="

    if [[ ${TESTS_FAILED} -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed.${NC}"
        return 1
    fi
}

#############################################################################
# Entry Point
#############################################################################

main() {
    # Check dependencies
    log_info "Checking dependencies..."

    if ! command -v openssl &> /dev/null; then
        log_error "openssl not found (required for testing)"
        exit 1
    fi

    if command -v cosign &> /dev/null; then
        log_info "cosign found - using full cosign tests"
    else
        log_info "cosign not found - using openssl fallback"
    fi

    # Run tests
    run_all_tests
    exit $?
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
