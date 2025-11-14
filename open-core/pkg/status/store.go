package status

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RequestStatus represents the state of an async request
type RequestStatus string

const (
	StatusPending    RequestStatus = "pending"
	StatusProcessing RequestStatus = "processing"
	StatusComplete   RequestStatus = "complete"
	StatusFailed     RequestStatus = "failed"
)

// StatusRecord represents the full status of a request
type StatusRecord struct {
	RequestID   string         `json:"request_id"`
	TenantID    string         `json:"tenant_id"`
	Status      RequestStatus  `json:"status"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	Progress    int            `json:"progress"` // 0-100
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	StreamURL   string         `json:"stream_url,omitempty"`
}

// Store manages request status storage
type Store interface {
	// Create creates a new status record
	Create(ctx context.Context, record *StatusRecord) error

	// Get retrieves a status record by request ID
	Get(ctx context.Context, requestID string) (*StatusRecord, error)

	// Update updates an existing status record
	Update(ctx context.Context, record *StatusRecord) error

	// UpdateStatus updates just the status field
	UpdateStatus(ctx context.Context, requestID string, status RequestStatus) error

	// UpdateProgress updates progress percentage
	UpdateProgress(ctx context.Context, requestID string, progress int) error

	// SetResult sets the final result
	SetResult(ctx context.Context, requestID string, result json.RawMessage) error

	// SetError sets an error message and marks as failed
	SetError(ctx context.Context, requestID string, errMsg string) error

	// Delete removes a status record
	Delete(ctx context.Context, requestID string) error

	// List lists all status records for a tenant
	List(ctx context.Context, tenantID string, limit int) ([]*StatusRecord, error)
}

// RedisStore implements Store using Redis
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration // Time-to-live for status records (24 hours default)
}

// NewRedisStore creates a new Redis-backed status store
func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	if ttl == 0 {
		ttl = 24 * time.Hour // Default 24 hours
	}
	return &RedisStore{
		client: client,
		ttl:    ttl,
	}
}

// statusKey returns the Redis key for a request status
func (s *RedisStore) statusKey(requestID string) string {
	return fmt.Sprintf("status:%s", requestID)
}

// tenantIndexKey returns the Redis key for tenant's request index
func (s *RedisStore) tenantIndexKey(tenantID string) string {
	return fmt.Sprintf("tenant:%s:requests", tenantID)
}

// Create creates a new status record
func (s *RedisStore) Create(ctx context.Context, record *StatusRecord) error {
	if record.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}
	if record.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	// Set timestamps
	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now

	// Marshal to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal status record: %w", err)
	}

	// Store in Redis with TTL
	key := s.statusKey(record.RequestID)
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store status: %w", err)
	}

	// Add to tenant index (sorted set by created_at timestamp)
	tenantKey := s.tenantIndexKey(record.TenantID)
	score := float64(now.Unix())
	if err := s.client.ZAdd(ctx, tenantKey, redis.Z{
		Score:  score,
		Member: record.RequestID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to tenant index: %w", err)
	}

	// Set TTL on tenant index too
	s.client.Expire(ctx, tenantKey, s.ttl)

	return nil
}

// Get retrieves a status record by request ID
func (s *RedisStore) Get(ctx context.Context, requestID string) (*StatusRecord, error) {
	if requestID == "" {
		return nil, fmt.Errorf("request_id is required")
	}

	key := s.statusKey(requestID)
	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("status not found for request_id: %s", requestID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var record StatusRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &record, nil
}

// Update updates an existing status record
func (s *RedisStore) Update(ctx context.Context, record *StatusRecord) error {
	if record.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}

	// Check if record exists
	key := s.statusKey(record.RequestID)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("status not found for request_id: %s", record.RequestID)
	}

	// Update timestamp
	record.UpdatedAt = time.Now()

	// If status is complete or failed, set completed_at
	if record.CompletedAt == nil && (record.Status == StatusComplete || record.Status == StatusFailed) {
		now := time.Now()
		record.CompletedAt = &now
	}

	// Marshal to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal status record: %w", err)
	}

	// Update in Redis, keeping original TTL
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// UpdateStatus updates just the status field
func (s *RedisStore) UpdateStatus(ctx context.Context, requestID string, status RequestStatus) error {
	record, err := s.Get(ctx, requestID)
	if err != nil {
		return err
	}

	record.Status = status
	return s.Update(ctx, record)
}

// UpdateProgress updates progress percentage
func (s *RedisStore) UpdateProgress(ctx context.Context, requestID string, progress int) error {
	if progress < 0 || progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100")
	}

	record, err := s.Get(ctx, requestID)
	if err != nil {
		return err
	}

	record.Progress = progress
	return s.Update(ctx, record)
}

// SetResult sets the final result
func (s *RedisStore) SetResult(ctx context.Context, requestID string, result json.RawMessage) error {
	record, err := s.Get(ctx, requestID)
	if err != nil {
		return err
	}

	record.Result = result
	record.Status = StatusComplete
	record.Progress = 100
	return s.Update(ctx, record)
}

// SetError sets an error message and marks as failed
func (s *RedisStore) SetError(ctx context.Context, requestID string, errMsg string) error {
	record, err := s.Get(ctx, requestID)
	if err != nil {
		return err
	}

	record.Error = errMsg
	record.Status = StatusFailed
	return s.Update(ctx, record)
}

// Delete removes a status record
func (s *RedisStore) Delete(ctx context.Context, requestID string) error {
	if requestID == "" {
		return fmt.Errorf("request_id is required")
	}

	// Get record first to get tenant_id
	record, err := s.Get(ctx, requestID)
	if err != nil {
		return err
	}

	// Remove from Redis
	key := s.statusKey(requestID)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete status: %w", err)
	}

	// Remove from tenant index
	tenantKey := s.tenantIndexKey(record.TenantID)
	if err := s.client.ZRem(ctx, tenantKey, requestID).Err(); err != nil {
		return fmt.Errorf("failed to remove from tenant index: %w", err)
	}

	return nil
}

// List lists all status records for a tenant
func (s *RedisStore) List(ctx context.Context, tenantID string, limit int) ([]*StatusRecord, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Get request IDs from tenant index (most recent first)
	tenantKey := s.tenantIndexKey(tenantID)
	requestIDs, err := s.client.ZRevRange(ctx, tenantKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant requests: %w", err)
	}

	// Fetch each status record
	records := make([]*StatusRecord, 0, len(requestIDs))
	for _, requestID := range requestIDs {
		record, err := s.Get(ctx, requestID)
		if err != nil {
			// Skip records that can't be fetched (might have expired)
			continue
		}
		records = append(records, record)
	}

	return records, nil
}
