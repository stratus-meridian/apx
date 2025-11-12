package compiler

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// SignatureConfig holds configuration for artifact signing
type SignatureConfig struct {
	// Environment determines key source
	Environment string // dev, staging, production

	// For development: local key file path
	LocalKeyPath string

	// For production: GCP Secret Manager
	ProjectID  string
	SecretName string
	Version    string
}

// ArtifactSigner handles signing of policy artifacts
type ArtifactSigner struct {
	config     *SignatureConfig
	privateKey crypto.PrivateKey
}

// NewArtifactSigner creates a new artifact signer
func NewArtifactSigner(config *SignatureConfig) (*ArtifactSigner, error) {
	signer := &ArtifactSigner{
		config: config,
	}

	// Load private key based on environment
	if err := signer.loadPrivateKey(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	return signer, nil
}

// loadPrivateKey loads the private key from appropriate source
func (s *ArtifactSigner) loadPrivateKey(ctx context.Context) error {
	if s.config.Environment == "production" || s.config.Environment == "staging" {
		// Load from Secret Manager
		return s.loadKeyFromSecretManager(ctx)
	}

	// Load from local file for development
	return s.loadKeyFromFile()
}

// loadKeyFromSecretManager loads private key from GCP Secret Manager
func (s *ArtifactSigner) loadKeyFromSecretManager(ctx context.Context) error {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer client.Close()

	// Access the secret version
	secretName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s",
		s.config.ProjectID,
		s.config.SecretName,
		s.config.Version,
	)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to access secret: %w", err)
	}

	// Parse the PEM-encoded private key
	return s.parsePrivateKey(result.Payload.Data)
}

// loadKeyFromFile loads private key from local file (dev mode)
func (s *ArtifactSigner) loadKeyFromFile() error {
	keyPath := s.config.LocalKeyPath
	if keyPath == "" {
		keyPath = "keys/cosign.key" // Default dev path
	}

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("key file not found at %s (run: cosign generate-key-pair)", keyPath)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	return s.parsePrivateKey(keyData)
}

// parsePrivateKey parses PEM-encoded private key
func (s *ArtifactSigner) parsePrivateKey(pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	// Try parsing as PKCS8 first (cosign default)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try parsing as EC private key
		key, err = x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	s.privateKey = key
	return nil
}

// SignArtifact signs a policy artifact and creates a signature file
func (s *ArtifactSigner) SignArtifact(artifactPath string) error {
	// Read the artifact
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}

	// Create signature
	signature, err := s.signBlob(artifactData)
	if err != nil {
		return fmt.Errorf("failed to sign artifact: %w", err)
	}

	// Write signature to .sig file
	sigPath := artifactPath + ".sig"
	if err := os.WriteFile(sigPath, signature, 0644); err != nil {
		return fmt.Errorf("failed to write signature: %w", err)
	}

	return nil
}

// signBlob creates a signature for the given data
func (s *ArtifactSigner) signBlob(data []byte) ([]byte, error) {
	// Hash the data
	hash := sha256.Sum256(data)

	// Sign based on key type
	switch key := s.privateKey.(type) {
	case *ecdsa.PrivateKey:
		// Sign with ECDSA
		r, s, err := ecdsa.Sign(rand.Reader, key, hash[:])
		if err != nil {
			return nil, fmt.Errorf("failed to sign with ECDSA: %w", err)
		}

		// Encode signature (simple concatenation of r and s)
		// In production, use a proper signature format (DER, etc.)
		signature := append(r.Bytes(), s.Bytes()...)
		return signature, nil

	default:
		return nil, fmt.Errorf("unsupported key type: %T", key)
	}
}

// SignArtifactWithMetadata signs an artifact and includes metadata
func (s *ArtifactSigner) SignArtifactWithMetadata(artifactPath, version, policyID string) error {
	// Read the artifact
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}

	// Create metadata JSON
	metadata := fmt.Sprintf(`{
  "artifact_path": "%s",
  "version": "%s",
  "policy_id": "%s",
  "algorithm": "ECDSA-SHA256",
  "timestamp": "%s"
}`, filepath.Base(artifactPath), version, policyID, fmt.Sprintf("%d", os.Getuid()))

	// Combine artifact hash + metadata for signing
	hash := sha256.Sum256(append(artifactData, []byte(metadata)...))

	// Sign
	signature, err := s.signBlobHash(hash[:])
	if err != nil {
		return fmt.Errorf("failed to sign artifact: %w", err)
	}

	// Write signature file with metadata
	sigContent := fmt.Sprintf("%s\n---\n%s", metadata, string(signature))
	sigPath := artifactPath + ".sig"
	if err := os.WriteFile(sigPath, []byte(sigContent), 0644); err != nil {
		return fmt.Errorf("failed to write signature: %w", err)
	}

	return nil
}

// signBlobHash signs a pre-computed hash
func (s *ArtifactSigner) signBlobHash(hash []byte) ([]byte, error) {
	switch key := s.privateKey.(type) {
	case *ecdsa.PrivateKey:
		r, s, err := ecdsa.Sign(rand.Reader, key, hash)
		if err != nil {
			return nil, fmt.Errorf("failed to sign: %w", err)
		}
		signature := append(r.Bytes(), s.Bytes()...)
		return signature, nil

	default:
		return nil, fmt.Errorf("unsupported key type: %T", key)
	}
}

// GenerateDevKey generates a development key pair (for testing only)
func GenerateDevKey(outputDir string) error {
	// Generate ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(crypto.SHA256, rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Marshal private key
	privBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Create PEM block
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privBytes,
	})

	// Write private key
	privPath := filepath.Join(outputDir, "cosign.key")
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Marshal public key
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Create PEM block for public key
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	// Write public key
	pubPath := filepath.Join(outputDir, "cosign.pub")
	if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// TODO: Production integration with cosign CLI
// For production, we should use the official cosign tool:
// - cosign generate-key-pair --kms gcpkms://projects/PROJECT/locations/LOCATION/keyRings/RING/cryptoKeys/KEY
// - cosign sign-blob --key gcpkms://... artifact.wasm
// - This implementation provides a working foundation for development and testing
