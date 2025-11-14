package policy

import (
	"context"

	"go.uber.org/zap"
)

// Store defines the interface for loading and managing policy bundles.
// The open-core version uses a no-op implementation.
// The commercial version loads compiled WASM policies from GCS/Firestore.
type Store interface {
	// IsReady returns true if the policy store is initialized and ready
	IsReady() bool

	// GetPolicyVersion returns the current policy version for a tenant
	GetPolicyVersion(tenantID string) string
}

// NoOpStore is a stub implementation that always returns "default".
type NoOpStore struct {
	logger *zap.Logger
}

// NewNoOpStore creates a new no-op policy store.
func NewNoOpStore(logger *zap.Logger) *NoOpStore {
	return &NoOpStore{logger: logger}
}

// IsReady always returns true for the no-op store.
func (s *NoOpStore) IsReady() bool {
	return true
}

// GetPolicyVersion always returns "default" for the no-op store.
func (s *NoOpStore) GetPolicyVersion(tenantID string) string {
	return "default"
}

// NewStore creates a new policy store.
// In the open-core version, this returns a no-op store.
// In the commercial version, this loads policies from GCS/Firestore.
func NewStore(ctx context.Context, logger *zap.Logger) (Store, error) {
	logger.Info("using no-op policy store (open-core edition)")
	return NewNoOpStore(logger), nil
}
