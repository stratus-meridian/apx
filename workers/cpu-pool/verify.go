package worker

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// VerificationConfig holds configuration for artifact verification
type VerificationConfig struct {
	// Environment determines key source
	Environment string // dev, staging, production

	// For development: local public key file path
	LocalPubKeyPath string

	// For production: GCP Secret Manager
	ProjectID  string
	SecretName string // Public key stored in Secret Manager
	Version    string

	// Allowed policy versions (version whitelist)
	AllowedVersions []string

	// Strict mode: reject if signature missing
	StrictMode bool
}

// ArtifactVerifier handles verification of signed artifacts
type ArtifactVerifier struct {
	config    *VerificationConfig
	publicKey crypto.PublicKey
}

// NewArtifactVerifier creates a new artifact verifier
func NewArtifactVerifier(config *VerificationConfig) (*ArtifactVerifier, error) {
	verifier := &ArtifactVerifier{
		config: config,
	}

	// Load public key based on environment
	if err := verifier.loadPublicKey(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return verifier, nil
}

// loadPublicKey loads the public key from appropriate source
func (v *ArtifactVerifier) loadPublicKey(ctx context.Context) error {
	if v.config.Environment == "production" || v.config.Environment == "staging" {
		// Load from Secret Manager
		return v.loadKeyFromSecretManager(ctx)
	}

	// Load from local file for development
	return v.loadKeyFromFile()
}

// loadKeyFromSecretManager loads public key from GCP Secret Manager
func (v *ArtifactVerifier) loadKeyFromSecretManager(ctx context.Context) error {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer client.Close()

	// Access the secret version
	secretName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s",
		v.config.ProjectID,
		v.config.SecretName,
		v.config.Version,
	)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to access secret: %w", err)
	}

	// Parse the PEM-encoded public key
	return v.parsePublicKey(result.Payload.Data)
}

// loadKeyFromFile loads public key from local file (dev mode)
func (v *ArtifactVerifier) loadKeyFromFile() error {
	keyPath := v.config.LocalPubKeyPath
	if keyPath == "" {
		keyPath = "keys/cosign.pub" // Default dev path
	}

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("public key file not found at %s", keyPath)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key file: %w", err)
	}

	return v.parsePublicKey(keyData)
}

// parsePublicKey parses PEM-encoded public key
func (v *ArtifactVerifier) parsePublicKey(pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	// Parse public key
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	v.publicKey = pub
	return nil
}

// VerifyArtifact verifies an artifact's signature
func (v *ArtifactVerifier) VerifyArtifact(artifactPath string) error {
	// Check if signature file exists
	sigPath := artifactPath + ".sig"
	if _, err := os.Stat(sigPath); os.IsNotExist(err) {
		if v.config.StrictMode {
			return fmt.Errorf("signature file not found: %s (strict mode enabled)", sigPath)
		}
		// In non-strict mode, allow unsigned artifacts (for backwards compatibility)
		return nil
	}

	// Read artifact
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}

	// Read signature
	signatureData, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Errorf("failed to read signature: %w", err)
	}

	// Verify signature
	if err := v.verifyBlob(artifactData, signatureData); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// verifyBlob verifies a signature against data
func (v *ArtifactVerifier) verifyBlob(data, signature []byte) error {
	// Hash the data
	hash := sha256.Sum256(data)

	// Verify based on public key type
	switch pub := v.publicKey.(type) {
	case *ecdsa.PublicKey:
		// Parse signature (r and s concatenated)
		sigLen := len(signature)
		if sigLen%2 != 0 {
			return fmt.Errorf("invalid signature length: %d", sigLen)
		}

		mid := sigLen / 2
		r := new(big.Int).SetBytes(signature[:mid])
		s := new(big.Int).SetBytes(signature[mid:])

		// Verify signature
		if !ecdsa.Verify(pub, hash[:], r, s) {
			return fmt.Errorf("ECDSA signature verification failed")
		}

		return nil

	default:
		return fmt.Errorf("unsupported public key type: %T", pub)
	}
}

// VerifyArtifactWithMetadata verifies artifact with metadata validation
func (v *ArtifactVerifier) VerifyArtifactWithMetadata(artifactPath string) (map[string]string, error) {
	// Check if signature file exists
	sigPath := artifactPath + ".sig"
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		if v.config.StrictMode {
			return nil, fmt.Errorf("signature file not found: %w", err)
		}
		return nil, nil
	}

	// Parse signature file (metadata + signature)
	parts := strings.Split(string(sigData), "\n---\n")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid signature file format")
	}

	metadata := make(map[string]string)
	// TODO: Parse JSON metadata properly
	// For now, simple key-value extraction

	// Read artifact
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	// Verify signature
	if err := v.verifyBlob(artifactData, []byte(parts[1])); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return metadata, nil
}

// VerifyPolicyVersion checks if a policy version is in the allowed list
func (v *ArtifactVerifier) VerifyPolicyVersion(version string) error {
	if len(v.config.AllowedVersions) == 0 {
		// No whitelist configured, allow all
		return nil
	}

	for _, allowed := range v.config.AllowedVersions {
		if version == allowed {
			return nil
		}
	}

	return fmt.Errorf("policy version %s not in allowed list: %v", version, v.config.AllowedVersions)
}

// PolicyBundle represents a loaded policy artifact
type PolicyBundle struct {
	ID          string
	Version     string
	Path        string
	Data        []byte
	Verified    bool
	VerifiedAt  string
	Metadata    map[string]string
}

// LoadPolicy loads and verifies a policy artifact before use
func (v *ArtifactVerifier) LoadPolicy(ref string) (*PolicyBundle, error) {
	// Download or locate artifact
	artifactPath, err := v.locateArtifact(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to locate artifact: %w", err)
	}

	// Extract version from reference or filename
	version := v.extractVersion(ref)

	// CRITICAL: Verify policy version is allowed
	if err := v.VerifyPolicyVersion(version); err != nil {
		return nil, err
	}

	// CRITICAL: Verify signature before loading
	if err := v.VerifyArtifact(artifactPath); err != nil {
		return nil, fmt.Errorf("artifact verification failed for %s: %w", ref, err)
	}

	// Read verified artifact
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read verified artifact: %w", err)
	}

	// Create policy bundle
	bundle := &PolicyBundle{
		ID:       ref,
		Version:  version,
		Path:     artifactPath,
		Data:     data,
		Verified: true,
	}

	return bundle, nil
}

// locateArtifact finds the artifact file (placeholder implementation)
func (v *ArtifactVerifier) locateArtifact(ref string) (string, error) {
	// TODO: Implement actual artifact resolution
	// - Check local cache
	// - Download from GCS if not cached
	// - Verify checksum

	// For now, assume ref is a local path
	if _, err := os.Stat(ref); err == nil {
		return ref, nil
	}

	return "", fmt.Errorf("artifact not found: %s", ref)
}

// extractVersion extracts version from artifact reference
func (v *ArtifactVerifier) extractVersion(ref string) string {
	// TODO: Implement proper version extraction
	// Expected format: pb-pay-v1@1.0.0.wasm or similar

	parts := strings.Split(ref, "@")
	if len(parts) >= 2 {
		version := strings.TrimSuffix(parts[1], ".wasm")
		return version
	}

	return "unknown"
}

// RejectTamperedArtifact is called when verification fails
func (v *ArtifactVerifier) RejectTamperedArtifact(artifactPath string, err error) error {
	// Log security event
	fmt.Printf("[SECURITY] Tampered artifact rejected: %s - %v\n", artifactPath, err)

	// TODO: Send alert to security monitoring
	// - Report to Cloud Logging with severity=CRITICAL
	// - Trigger incident response
	// - Block artifact in cache

	return fmt.Errorf("SECURITY: artifact tampering detected in %s: %w", artifactPath, err)
}

// SecurityChecks performs additional security validations
func (v *ArtifactVerifier) SecurityChecks(bundle *PolicyBundle) error {
	// Check artifact size (prevent DoS)
	maxSize := 10 * 1024 * 1024 // 10MB
	if len(bundle.Data) > maxSize {
		return fmt.Errorf("artifact too large: %d bytes (max: %d)", len(bundle.Data), maxSize)
	}

	// Check magic bytes for WASM
	if len(bundle.Data) < 4 {
		return fmt.Errorf("invalid artifact: too small")
	}

	wasmMagic := []byte{0x00, 0x61, 0x73, 0x6D} // "\0asm"
	if string(bundle.Data[:4]) != string(wasmMagic) {
		return fmt.Errorf("invalid artifact: not a WASM file")
	}

	return nil
}

// TODO: Production enhancements
// 1. Implement cosign integration: cosign verify-blob --key <key> --signature <sig> <artifact>
// 2. Add support for keyless signing (Sigstore Fulcio/Rekor)
// 3. Implement artifact caching with signature verification
// 4. Add support for certificate chain validation
// 5. Implement SBOM (Software Bill of Materials) verification
// 6. Add support for policy attestations
// 7. Implement artifact revocation checks
