#!/usr/bin/env bash

#############################################################################
# Artifact Signing Script for APX Platform
#
# Purpose: Sign policy artifacts (.wasm files) with cosign for verification
# Usage:   ./sign_artifact.sh <artifact_path> [options]
#
# This script supports both development (local keys) and production (GCP KMS)
#############################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Default configuration
ENVIRONMENT="${ENVIRONMENT:-dev}"
KEY_PATH="${KEY_PATH:-${PROJECT_ROOT}/keys/cosign.key}"
PUB_KEY_PATH="${PUB_KEY_PATH:-${PROJECT_ROOT}/keys/cosign.pub}"

# GCP Configuration (for production)
GCP_PROJECT="${GCP_PROJECT_ID:-}"
GCP_KMS_KEY="${GCP_KMS_KEY:-}"
SECRET_NAME="${SECRET_NAME:-apx-artifact-signing-key}"

#############################################################################
# Helper Functions
#############################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

usage() {
    cat <<EOF
Artifact Signing Script for APX Platform

Usage:
    $0 <artifact_path> [options]

Arguments:
    artifact_path       Path to the policy artifact (.wasm file)

Options:
    --env <env>         Environment: dev, staging, production (default: dev)
    --key <path>        Path to private key (dev mode only)
    --version <ver>     Policy version (e.g., 1.0.0)
    --policy-id <id>    Policy ID (e.g., pb-pay-v1)
    --use-cosign        Use cosign CLI instead of Go implementation
    --help              Show this help message

Examples:
    # Development (local key)
    $0 pb-pay-v1@1.0.0.wasm --version 1.0.0 --policy-id pb-pay-v1

    # Using cosign CLI
    $0 pb-pay-v1@1.0.0.wasm --use-cosign

    # Production (GCP KMS)
    $0 pb-pay-v1@1.0.0.wasm --env production --version 1.0.0

Environment Variables:
    ENVIRONMENT         Environment (dev, staging, production)
    KEY_PATH            Path to private key file
    GCP_PROJECT_ID      GCP project ID (production)
    GCP_KMS_KEY         GCP KMS key URI (production)

EOF
    exit 1
}

#############################################################################
# Key Management Functions
#############################################################################

generate_dev_keys() {
    local key_dir="${PROJECT_ROOT}/keys"

    log_info "Generating development key pair..."

    mkdir -p "${key_dir}"

    if command -v cosign &> /dev/null; then
        # Use cosign to generate keys
        cd "${key_dir}"
        COSIGN_PASSWORD="" cosign generate-key-pair
        log_success "Keys generated with cosign"
    else
        # Use Go implementation
        go run "${PROJECT_ROOT}/control/compiler/cmd/keygen.go" "${key_dir}"
        log_success "Keys generated with Go implementation"
    fi

    log_info "Keys stored in: ${key_dir}"
    log_info "  Private key: ${key_dir}/cosign.key"
    log_info "  Public key:  ${key_dir}/cosign.pub"
}

check_keys() {
    if [[ "${ENVIRONMENT}" == "dev" ]]; then
        # Check for local keys
        if [[ ! -f "${KEY_PATH}" ]]; then
            log_warning "Private key not found: ${KEY_PATH}"
            read -p "Generate keys now? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                generate_dev_keys
            else
                log_error "Cannot sign without keys. Exiting."
                exit 1
            fi
        fi

        if [[ ! -f "${PUB_KEY_PATH}" ]]; then
            log_error "Public key not found: ${PUB_KEY_PATH}"
            exit 1
        fi

        log_success "Keys found"
    else
        # Production: check GCP configuration
        if [[ -z "${GCP_PROJECT}" ]]; then
            log_error "GCP_PROJECT_ID not set (required for ${ENVIRONMENT})"
            exit 1
        fi

        log_info "Using GCP Secret Manager: ${GCP_PROJECT}"
    fi
}

#############################################################################
# Signing Functions
#############################################################################

sign_with_go() {
    local artifact_path="$1"
    local version="$2"
    local policy_id="$3"

    log_info "Signing with Go implementation..."

    # Create temporary Go program to sign
    local tmp_dir="$(mktemp -d)"
    local sign_program="${tmp_dir}/sign.go"

    cat > "${sign_program}" <<'GOEOF'
package main

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/apx/control/compiler"
)

func main() {
    if len(os.Args) < 5 {
        fmt.Println("Usage: sign <artifact> <key> <version> <policy_id>")
        os.Exit(1)
    }

    artifactPath := os.Args[1]
    keyPath := os.Args[2]
    version := os.Args[3]
    policyID := os.Args[4]

    config := &compiler.SignatureConfig{
        Environment:  "dev",
        LocalKeyPath: keyPath,
    }

    signer, err := compiler.NewArtifactSigner(config)
    if err != nil {
        fmt.Printf("Failed to create signer: %v\n", err)
        os.Exit(1)
    }

    if err := signer.SignArtifactWithMetadata(artifactPath, version, policyID); err != nil {
        fmt.Printf("Failed to sign artifact: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Signed: %s\n", filepath.Base(artifactPath))
}
GOEOF

    # Run the signing program
    cd "${PROJECT_ROOT}"
    go run "${sign_program}" "${artifact_path}" "${KEY_PATH}" "${version}" "${policy_id}"

    rm -rf "${tmp_dir}"
}

sign_with_cosign() {
    local artifact_path="$1"

    log_info "Signing with cosign CLI..."

    if ! command -v cosign &> /dev/null; then
        log_error "cosign not found. Install with: brew install cosign"
        exit 1
    fi

    if [[ "${ENVIRONMENT}" == "production" ]]; then
        # Use GCP KMS
        if [[ -z "${GCP_KMS_KEY}" ]]; then
            log_error "GCP_KMS_KEY not set"
            exit 1
        fi

        cosign sign-blob \
            --key "${GCP_KMS_KEY}" \
            --output-signature "${artifact_path}.sig" \
            "${artifact_path}"
    else
        # Use local key
        COSIGN_PASSWORD="" cosign sign-blob \
            --key "${KEY_PATH}" \
            --output-signature "${artifact_path}.sig" \
            "${artifact_path}"
    fi
}

#############################################################################
# Verification Functions
#############################################################################

verify_signature() {
    local artifact_path="$1"
    local sig_path="${artifact_path}.sig"

    log_info "Verifying signature..."

    if [[ ! -f "${sig_path}" ]]; then
        log_error "Signature file not found: ${sig_path}"
        return 1
    fi

    if command -v cosign &> /dev/null; then
        # Verify with cosign
        if [[ "${ENVIRONMENT}" == "production" ]]; then
            cosign verify-blob \
                --key "${GCP_KMS_KEY}" \
                --signature "${sig_path}" \
                "${artifact_path}"
        else
            COSIGN_PASSWORD="" cosign verify-blob \
                --key "${PUB_KEY_PATH}" \
                --signature "${sig_path}" \
                "${artifact_path}"
        fi
    else
        log_warning "cosign not available, skipping verification"
        return 0
    fi

    log_success "Signature verified"
}

#############################################################################
# Main Script
#############################################################################

main() {
    # Parse arguments
    if [[ $# -lt 1 ]]; then
        usage
    fi

    local artifact_path="$1"
    shift

    local version="1.0.0"
    local policy_id="unknown"
    local use_cosign=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --env)
                ENVIRONMENT="$2"
                shift 2
                ;;
            --key)
                KEY_PATH="$2"
                shift 2
                ;;
            --version)
                version="$2"
                shift 2
                ;;
            --policy-id)
                policy_id="$2"
                shift 2
                ;;
            --use-cosign)
                use_cosign=true
                shift
                ;;
            --help)
                usage
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                ;;
        esac
    done

    # Validate artifact
    if [[ ! -f "${artifact_path}" ]]; then
        log_error "Artifact not found: ${artifact_path}"
        exit 1
    fi

    log_info "Artifact Signing - APX Platform"
    log_info "================================"
    log_info "Environment:  ${ENVIRONMENT}"
    log_info "Artifact:     ${artifact_path}"
    log_info "Version:      ${version}"
    log_info "Policy ID:    ${policy_id}"
    echo

    # Check keys
    check_keys

    # Sign artifact
    if [[ "${use_cosign}" == "true" ]]; then
        sign_with_cosign "${artifact_path}"
    else
        sign_with_go "${artifact_path}" "${version}" "${policy_id}"
    fi

    log_success "Artifact signed successfully"
    log_info "Signature: ${artifact_path}.sig"

    # Verify signature
    verify_signature "${artifact_path}"

    # Show file info
    echo
    log_info "File Information:"
    ls -lh "${artifact_path}"
    ls -lh "${artifact_path}.sig"

    echo
    log_success "Done!"
}

# Run main function
main "$@"
