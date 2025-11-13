package policy

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// Loader downloads policy artifacts from GCS
type Loader struct {
	client     *storage.Client
	bucketName string
}

// NewLoader creates a new policy loader
func NewLoader(ctx context.Context, bucketName string) (*Loader, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &Loader{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// Close closes the storage client
func (l *Loader) Close() error {
	return l.client.Close()
}

// Load downloads a policy artifact from GCS
func (l *Loader) Load(ctx context.Context, name, version, hash string) (*CacheEntry, error) {
	// Construct GCS path: policies/{name}/{version}/{hash}.wasm
	objectPath := fmt.Sprintf("policies/%s/%s/%s.wasm", name, version, hash)

	bucket := l.client.Bucket(l.bucketName)
	obj := bucket.Object(objectPath)

	// Read WASM bytes
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open artifact: %w", err)
	}
	defer reader.Close()

	wasmBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	return &CacheEntry{
		Name:    name,
		Version: version,
		Hash:    hash,
		WASM:    wasmBytes,
	}, nil
}

// LoadLatest loads the latest version of a policy (stub - needs Firestore integration)
func (l *Loader) LoadLatest(ctx context.Context, name string) (*CacheEntry, error) {
	// TODO: Query Firestore for latest version
	// For now, return error indicating this needs implementation
	return nil, fmt.Errorf("LoadLatest not yet implemented - needs Firestore integration")
}
