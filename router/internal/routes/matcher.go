package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"go.uber.org/zap"
)

// RequestMessage represents a request to be processed by workers
type RequestMessage struct {
	RequestID      string            `json:"request_id"`
	TenantID       string            `json:"tenant_id"`
	TenantTier     string            `json:"tenant_tier"`
	Route          string            `json:"route"`
	Method         string            `json:"method"`
	PolicyVersion  string            `json:"policy_version"`
	Headers        map[string]string `json:"headers"`
	Body           json.RawMessage   `json:"body"`
	ReceivedAt     time.Time         `json:"received_at"`
}

// Matcher handles route matching and message publishing
type Matcher struct {
	topic  *pubsub.Topic
	logger *zap.Logger
}

// NewMatcher creates a new route matcher
func NewMatcher(topic *pubsub.Topic, logger *zap.Logger) *Matcher {
	return &Matcher{
		topic:  topic,
		logger: logger,
	}
}

// PublishRequest publishes a request to Pub/Sub with tenant isolation
func (m *Matcher) PublishRequest(ctx context.Context, msg *RequestMessage) error {
	// CRITICAL: Validate tenant_id is present
	if msg.TenantID == "" {
		return fmt.Errorf("tenant_id is required for request processing")
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create Pub/Sub message with tenant isolation attributes
	pubsubMsg := &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			// CRITICAL: tenant_id attribute ensures tenant-based filtering
			"tenant_id": msg.TenantID,

			// Tenant tier for priority routing
			"tenant_tier": msg.TenantTier,

			// Request metadata
			"request_id":     msg.RequestID,
			"route":          msg.Route,
			"policy_version": msg.PolicyVersion,

			// Timestamp for monitoring
			"received_at": msg.ReceivedAt.Format(time.RFC3339),
		},

		// CRITICAL: Use tenant_id as ordering key for FIFO processing per tenant
		// This ensures:
		// 1. Requests from the same tenant are processed in order
		// 2. Requests from different tenants can be processed in parallel
		// 3. No tenant can block another tenant's request processing
		OrderingKey: msg.TenantID,
	}

	// Publish with retry
	result := m.topic.Publish(ctx, pubsubMsg)

	// Wait for server acknowledgment
	serverID, err := result.Get(ctx)
	if err != nil {
		m.logger.Error("failed to publish message",
			zap.String("tenant_id", msg.TenantID),
			zap.String("request_id", msg.RequestID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	m.logger.Debug("message published",
		zap.String("tenant_id", msg.TenantID),
		zap.String("request_id", msg.RequestID),
		zap.String("server_id", serverID),
		zap.String("ordering_key", msg.TenantID),
	)

	return nil
}

// MatchRoute determines which route a request should take
// This is a placeholder - real implementation would check policy bundles
func (m *Matcher) MatchRoute(ctx context.Context, path string, method string) (string, error) {
	// TODO: Implement actual route matching based on policy bundles
	// For now, return the path as the route
	return path, nil
}

// PublishBatch publishes multiple requests in a batch for efficiency
// While maintaining tenant isolation
func (m *Matcher) PublishBatch(ctx context.Context, messages []*RequestMessage) error {
	if len(messages) == 0 {
		return nil
	}

	var results []*pubsub.PublishResult

	for _, msg := range messages {
		// CRITICAL: Validate each message has tenant_id
		if msg.TenantID == "" {
			m.logger.Warn("skipping message without tenant_id",
				zap.String("request_id", msg.RequestID),
			)
			continue
		}

		data, err := json.Marshal(msg)
		if err != nil {
			m.logger.Error("failed to marshal message",
				zap.String("tenant_id", msg.TenantID),
				zap.String("request_id", msg.RequestID),
				zap.Error(err),
			)
			continue
		}

		pubsubMsg := &pubsub.Message{
			Data: data,
			Attributes: map[string]string{
				"tenant_id":      msg.TenantID,
				"tenant_tier":    msg.TenantTier,
				"request_id":     msg.RequestID,
				"route":          msg.Route,
				"policy_version": msg.PolicyVersion,
				"received_at":    msg.ReceivedAt.Format(time.RFC3339),
			},
			OrderingKey: msg.TenantID,
		}

		result := m.topic.Publish(ctx, pubsubMsg)
		results = append(results, result)
	}

	// Wait for all publishes to complete
	var publishErrors []error
	for i, result := range results {
		if _, err := result.Get(ctx); err != nil {
			publishErrors = append(publishErrors, fmt.Errorf("message %d failed: %w", i, err))
		}
	}

	if len(publishErrors) > 0 {
		return fmt.Errorf("batch publish had %d errors: %v", len(publishErrors), publishErrors)
	}

	m.logger.Info("batch published",
		zap.Int("count", len(results)),
	)

	return nil
}

// Close closes the Pub/Sub topic
func (m *Matcher) Close() error {
	m.topic.Stop()
	return nil
}
