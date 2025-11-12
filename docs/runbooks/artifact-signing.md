# Artifact Signing - APX Platform

**Status:** Implemented (Dev Mode)
**Owner:** Infrastructure Team
**Last Updated:** 2025-11-11

---

## Overview

APX Platform implements **cryptographic signing** for all policy artifacts to ensure:

- **Integrity:** Artifacts cannot be tampered with
- **Authenticity:** Only artifacts from trusted sources are loaded
- **Non-repudiation:** Signing provides audit trail
- **Security:** Workers reject unsigned or tampered artifacts

This runbook explains how artifact signing works, how to use it, and how to troubleshoot issues.

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                      Artifact Signing Flow                        │
└──────────────────────────────────────────────────────────────────┘

   Policy Config (YAML)
          │
          ↓
   ┌─────────────┐
   │  Compiler   │  1. Compile policy to WASM
   └──────┬──────┘
          │
          ↓
   ┌─────────────┐
   │   Signer    │  2. Sign artifact with private key
   └──────┬──────┘     (Secret Manager in prod, local in dev)
          │
          ├──→ artifact.wasm      (Binary artifact)
          └──→ artifact.wasm.sig  (Signature file)
          │
          ↓
   ┌─────────────┐
   │   Storage   │  3. Store both files in GCS
   │     GCS     │
   └──────┬──────┘
          │
          ↓
   ┌─────────────┐
   │   Worker    │  4. Download artifact + signature
   └──────┬──────┘
          │
          ↓
   ┌─────────────┐
   │  Verifier   │  5. Verify signature with public key
   └──────┬──────┘     (Reject if verification fails)
          │
          ↓
   ┌─────────────┐
   │   Execute   │  6. Load verified artifact into WASM runtime
   └─────────────┘
```

---

## Key Management

### Development Environment

**Location:** Local filesystem (`keys/` directory)

```bash
# Generate key pair (first time setup)
cd /Users/agentsy/APILEE
mkdir -p keys
cosign generate-key-pair
# Or use the script:
./tools/cli/sign_artifact.sh --help

# Keys created:
# - keys/cosign.key (private key - DO NOT COMMIT)
# - keys/cosign.pub (public key - safe to share)
```

**Security Notes:**
- Private keys stored locally
- Keys NOT committed to git (in `.gitignore`)
- Password protection optional for dev keys

### Staging/Production Environment

**Location:** GCP Secret Manager

```bash
# 1. Generate production keys with KMS
gcloud kms keyrings create apx-signing \
  --location=us-central1 \
  --project=$GCP_PROJECT_ID

gcloud kms keys create artifact-signing \
  --location=us-central1 \
  --keyring=apx-signing \
  --purpose=asymmetric-signing \
  --default-algorithm=ec-sign-p256-sha256

# 2. Get public key
gcloud kms keys versions get-public-key 1 \
  --key=artifact-signing \
  --keyring=apx-signing \
  --location=us-central1 \
  --output-file=cosign.pub

# 3. Store public key in Secret Manager (for workers)
gcloud secrets create apx-artifact-pubkey \
  --data-file=cosign.pub \
  --replication-policy=automatic

# 4. Grant permissions
gcloud kms keys add-iam-policy-binding artifact-signing \
  --location=us-central1 \
  --keyring=apx-signing \
  --member=serviceAccount:apx-compiler@$GCP_PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/cloudkms.signerVerifier
```

**Security Notes:**
- Private key never leaves KMS
- Public key distributed to workers
- IAM controls access to signing operations
- Audit logs for all signing operations

---

## How to Sign Artifacts

### Option 1: Automatic Signing (Recommended)

When using the APX compiler, artifacts are automatically signed:

```bash
# Compile and sign in one step
./tools/cli/apx compile configs/samples/payments-api.yaml

# Output:
# ✓ Compiled: pb-pay-v1@1.0.0.wasm
# ✓ Signed:    pb-pay-v1@1.0.0.wasm.sig
```

The compiler automatically:
1. Compiles policy to WASM
2. Signs the artifact
3. Stores both files
4. Uploads to GCS (if configured)

### Option 2: Manual Signing

Use the signing script for manual signing:

```bash
# Sign an existing artifact
./tools/cli/sign_artifact.sh pb-pay-v1@1.0.0.wasm \
  --version 1.0.0 \
  --policy-id pb-pay-v1

# Verify signature
cosign verify-blob \
  --key keys/cosign.pub \
  --signature pb-pay-v1@1.0.0.wasm.sig \
  pb-pay-v1@1.0.0.wasm
```

### Option 3: Using Cosign CLI Directly

```bash
# Sign with local key
COSIGN_PASSWORD="" cosign sign-blob \
  --key keys/cosign.key \
  --output-signature artifact.wasm.sig \
  artifact.wasm

# Sign with KMS (production)
cosign sign-blob \
  --key gcpkms://projects/$PROJECT_ID/locations/us-central1/keyRings/apx-signing/cryptoKeys/artifact-signing \
  --output-signature artifact.wasm.sig \
  artifact.wasm
```

---

## How Verification Works

Workers verify artifacts before loading:

```go
// In workers/cpu-pool/verify.go

// 1. Create verifier
config := &VerificationConfig{
    Environment:     "production",
    ProjectID:       "apx-prod",
    SecretName:      "apx-artifact-pubkey",
    AllowedVersions: []string{"1.0.0", "1.1.0"},
    StrictMode:      true, // Reject unsigned artifacts
}

verifier, err := NewArtifactVerifier(config)

// 2. Load and verify policy
bundle, err := verifier.LoadPolicy("pb-pay-v1@1.0.0")
if err != nil {
    // Verification failed - reject artifact
    log.Error("Artifact verification failed", "error", err)
    return err
}

// 3. Use verified artifact
wasmModule := loadWASM(bundle.Data)
```

### Verification Steps

1. **Signature File Check:** Ensure `.sig` file exists
2. **Version Check:** Verify version is in allowed list
3. **Signature Verification:** Verify cryptographic signature
4. **Security Checks:** Validate file size, magic bytes
5. **Load Artifact:** Only if all checks pass

### Rejection Scenarios

Artifacts are rejected if:

- ❌ Signature file missing (strict mode)
- ❌ Signature verification fails (tampering detected)
- ❌ Policy version not in allowed list
- ❌ Artifact too large (> 10MB)
- ❌ Invalid WASM magic bytes
- ❌ Public key not available

---

## Verification Flow

```
Worker receives policy reference: pb-pay-v1@1.0.0
         │
         ↓
   ┌─────────────────┐
   │ Check cache     │ ← Artifact cached locally?
   └────┬────────────┘
        │ No
        ↓
   ┌─────────────────┐
   │ Download from   │ ← Get artifact.wasm + artifact.wasm.sig
   │      GCS        │
   └────┬────────────┘
        │
        ↓
   ┌─────────────────┐
   │ Verify version  │ ← Is version in allowed list?
   └────┬────────────┘
        │ Yes
        ↓
   ┌─────────────────┐
   │ Verify sig file │ ← Does .sig file exist?
   └────┬────────────┘
        │ Yes
        ↓
   ┌─────────────────┐
   │ Load public key │ ← From Secret Manager or local
   └────┬────────────┘
        │
        ↓
   ┌─────────────────┐
   │ Compute hash    │ ← SHA-256 of artifact.wasm
   │   of artifact   │
   └────┬────────────┘
        │
        ↓
   ┌─────────────────┐
   │ Verify ECDSA    │ ← Verify(hash, signature, pubkey)
   │   signature     │
   └────┬────────────┘
        │
        ├─ Valid ──→ ┌─────────────────┐
        │            │ Load into WASM  │ ✓
        │            │     runtime     │
        │            └─────────────────┘
        │
        └─ Invalid → ┌─────────────────┐
                     │  REJECT         │ ✗
                     │  Log security   │
                     │  alert          │
                     └─────────────────┘
```

---

## Troubleshooting

### Problem: Signature Verification Failed

**Symptoms:**
```
ERROR: signature verification failed: ECDSA signature verification failed
```

**Possible Causes:**

1. **Artifact was modified after signing**
   ```bash
   # Re-sign the artifact
   ./tools/cli/sign_artifact.sh artifact.wasm --version 1.0.0
   ```

2. **Wrong public key used for verification**
   ```bash
   # Verify you're using the correct public key
   ls -la keys/cosign.pub

   # Re-generate keys if needed
   cosign generate-key-pair
   ```

3. **Signature file corrupted**
   ```bash
   # Check signature file exists and is not empty
   ls -lh artifact.wasm.sig
   cat artifact.wasm.sig

   # Re-sign if corrupted
   rm artifact.wasm.sig
   ./tools/cli/sign_artifact.sh artifact.wasm --version 1.0.0
   ```

### Problem: Keys Not Found

**Symptoms:**
```
ERROR: key file not found at keys/cosign.key
```

**Solution:**
```bash
# Generate keys for development
cd /Users/agentsy/APILEE
mkdir -p keys
cosign generate-key-pair

# Or use the signing script which will offer to generate keys
./tools/cli/sign_artifact.sh artifact.wasm --version 1.0.0
```

### Problem: Unknown Policy Version

**Symptoms:**
```
ERROR: policy version 1.2.0 not in allowed list: [1.0.0 1.1.0]
```

**Solution:**
```bash
# Update allowed versions in worker configuration
# Edit workers/cpu-pool/config.go or environment variables

export ALLOWED_POLICY_VERSIONS="1.0.0,1.1.0,1.2.0"

# Or deploy with updated configuration
./deploy.sh --update-allowed-versions
```

### Problem: Unsigned Artifact in Production

**Symptoms:**
```
ERROR: signature file not found: artifact.wasm.sig (strict mode enabled)
```

**Solution:**
```bash
# In production, all artifacts MUST be signed
# Sign the artifact before deploying:

./tools/cli/sign_artifact.sh artifact.wasm \
  --env production \
  --version 1.0.0 \
  --policy-id pb-pay-v1

# Upload both files to GCS
gsutil cp artifact.wasm gs://apx-policy-artifacts/
gsutil cp artifact.wasm.sig gs://apx-policy-artifacts/
```

### Problem: GCP Secret Manager Access Denied

**Symptoms:**
```
ERROR: failed to access secret: rpc error: code = PermissionDenied
```

**Solution:**
```bash
# Grant worker service account access to secrets
gcloud secrets add-iam-policy-binding apx-artifact-pubkey \
  --member=serviceAccount:apx-worker@$GCP_PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/secretmanager.secretAccessor

# Verify permissions
gcloud secrets get-iam-policy apx-artifact-pubkey
```

---

## Security Best Practices

### Key Management

✅ **DO:**
- Use KMS for production keys
- Rotate keys regularly (every 90 days)
- Store private keys in Secret Manager
- Use different keys per environment
- Enable audit logging for key access

❌ **DON'T:**
- Commit private keys to git
- Share private keys via email/Slack
- Use dev keys in production
- Store keys in container images
- Skip key rotation

### Signing Process

✅ **DO:**
- Sign all artifacts before deployment
- Verify signatures in CI/CD pipeline
- Use strict mode in production
- Maintain version whitelist
- Log all signature failures

❌ **DON'T:**
- Skip signing in dev (practice good habits)
- Allow unsigned artifacts in production
- Ignore verification failures
- Disable strict mode without approval
- Allow unknown versions

### Incident Response

If a signature verification fails in production:

1. **Immediate:** Block the artifact from loading
2. **Investigate:** Check audit logs for tampering
3. **Alert:** Notify security team
4. **Verify:** Compare artifact hash with source
5. **Re-deploy:** Use verified signed artifact
6. **Post-mortem:** Document what happened

---

## Monitoring and Alerts

### Key Metrics

Monitor these metrics in Cloud Monitoring:

```yaml
# Signature verification failures
metric: apx.artifacts.verification_failures
alert_threshold: > 0 in 5 minutes

# Unsigned artifacts blocked
metric: apx.artifacts.unsigned_blocked
alert_threshold: > 5 in 1 hour

# Unknown version attempts
metric: apx.artifacts.unknown_versions
alert_threshold: > 0 in 5 minutes
```

### Log Queries

```bash
# Find verification failures
gcloud logging read '
  severity=ERROR
  AND jsonPayload.message=~"signature verification failed"
' --limit 50 --format json

# Find unsigned artifact attempts
gcloud logging read '
  severity=WARNING
  AND jsonPayload.message=~"signature file not found"
' --limit 50

# Security events
gcloud logging read '
  jsonPayload.security_event="artifact_tampering"
' --limit 10
```

---

## Production Deployment Checklist

Before enabling artifact signing in production:

- [ ] KMS key ring created
- [ ] Signing key created in KMS
- [ ] Public key stored in Secret Manager
- [ ] IAM permissions configured
- [ ] Workers updated to verify signatures
- [ ] Strict mode enabled
- [ ] Version whitelist configured
- [ ] All existing artifacts signed
- [ ] Monitoring alerts configured
- [ ] Runbook reviewed by team
- [ ] Incident response plan defined
- [ ] Key rotation schedule set

---

## References

### Internal Documentation
- [Security Architecture](../architecture/security.md)
- [Policy Compilation Guide](./policy-compilation.md)
- [Worker Configuration](./worker-configuration.md)

### External Resources
- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)
- [Sigstore Project](https://www.sigstore.dev/)
- [GCP KMS Signing](https://cloud.google.com/kms/docs/digital-signatures)
- [WASM Security](https://webassembly.org/docs/security/)

### Support
- **Security Issues:** security@apx.platform
- **Slack:** #apx-security
- **On-call:** PagerDuty escalation

---

## Future Enhancements

### Planned Features

1. **Keyless Signing** (Q1 2026)
   - Sigstore Fulcio integration
   - Transparency log (Rekor)
   - No long-lived keys to manage

2. **SBOM Integration** (Q2 2026)
   - Sign SBOMs alongside artifacts
   - Verify dependency chains
   - CVE scanning integration

3. **Policy Attestations** (Q2 2026)
   - Attest policy approval
   - Multi-party signing
   - Compliance verification

4. **Automated Key Rotation** (Q3 2026)
   - Scheduled key rotation
   - Zero-downtime key updates
   - Historical key storage

### TODO for Production

- [ ] Implement cosign CLI integration
- [ ] Add keyless signing support
- [ ] Implement artifact revocation checks
- [ ] Add certificate chain validation
- [ ] Create automated key rotation
- [ ] Add SBOM verification
- [ ] Implement policy attestations
- [ ] Add hardware security module (HSM) support

---

**Last Updated:** 2025-11-11
**Next Review:** 2025-12-11
**Version:** 1.0
