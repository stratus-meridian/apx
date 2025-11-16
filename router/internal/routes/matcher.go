package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/stratus-meridian/apx/router/internal/middleware"
	"github.com/stratus-meridian/apx/router/pkg/status"
	"go.uber.org/zap"
)

// RequestMessage represents a request to be processed by workers
type RequestMessage struct {
	RequestID     string            `json:"request_id"`
	TenantID      string            `json:"tenant_id"`
	TenantTier    string            `json:"tenant_tier"`
	Route         string            `json:"route"`
	Method        string            `json:"method"`
	PolicyVersion string            `json:"policy_version"`
	Headers       map[string]string `json:"headers"`
	Body          json.RawMessage   `json:"body"`
	ReceivedAt    time.Time         `json:"received_at"`
}

// Matcher handles route matching and message publishing
type Matcher struct {
	topic       *pubsub.Topic
	statusStore status.Store
	logger      *zap.Logger
	baseURL     string // Base URL for constructing status/stream URLs
}

// NewMatcher creates a new route matcher
func NewMatcher(topic *pubsub.Topic, statusStore status.Store, logger *zap.Logger, baseURL string) *Matcher {
	return &Matcher{
		topic:       topic,
		statusStore: statusStore,
		logger:      logger,
		baseURL:     baseURL,
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

	m.logger.Info("message published to pub/sub",
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

// Handle is an HTTP handler that processes incoming requests
func (m *Matcher) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant_id from context (set by TenantContext middleware)
	tenantID := middleware.GetTenantID(ctx)
	tenantTier := middleware.GetTenantTier(ctx)

	// Extract request_id from context (set by RequestID middleware)
	requestID := middleware.GetRequestID(ctx)

	m.logger.Info("request received",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("request_id", requestID),
		zap.String("tenant_id", tenantID),
	)

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		m.logger.Error("failed to read request body", zap.Error(err))
		http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Determine route (for now, just use the path)
	route, err := m.MatchRoute(ctx, r.URL.Path, r.Method)
	if err != nil {
		m.logger.Error("failed to match route", zap.Error(err))
		http.Error(w, `{"error":"failed to match route"}`, http.StatusInternalServerError)
		return
	}

	// Extract policy version from context (set by PolicyVersionTag middleware)
	policyVersion := r.Header.Get("X-Policy-Version")
	if policyVersion == "" {
		policyVersion = "v1.0.0" // Default
	}

	var rawBody json.RawMessage
	switch {
	case len(bodyBytes) == 0:
		rawBody = nil
	case json.Valid(bodyBytes):
		rawBody = json.RawMessage(bodyBytes)
	default:
		m.logger.Warn("invalid JSON body",
			zap.String("request_id", requestID),
			zap.String("content_type", r.Header.Get("Content-Type")),
		)
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	// Build RequestMessage
	msg := &RequestMessage{
		RequestID:     requestID,
		TenantID:      tenantID,
		TenantTier:    tenantTier,
		Route:         route,
		Method:        r.Method,
		PolicyVersion: policyVersion,
		Headers:       extractHeaders(r),
		Body:          rawBody,
		ReceivedAt:    time.Now(),
	}

	// Create initial status record
	statusRecord := &status.StatusRecord{
		RequestID: requestID,
		TenantID:  tenantID,
		Status:    status.StatusPending,
		Progress:  0,
		StreamURL: fmt.Sprintf("%s/stream/%s", m.baseURL, requestID),
	}

	if err := m.statusStore.Create(ctx, statusRecord); err != nil {
		m.logger.Error("failed to create status record",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		// Continue anyway - don't fail the request
	}

	// Publish to Pub/Sub (if topic is configured)
	if m.topic != nil {
		if err := m.PublishRequest(ctx, msg); err != nil {
			m.logger.Error("failed to publish message",
				zap.String("request_id", requestID),
				zap.Error(err),
			)
			// Mark status as failed
			m.statusStore.SetError(ctx, requestID, fmt.Sprintf("failed to publish: %v", err))
			
			// Return helpful error message
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":      "publish_failed",
				"message":    "Failed to queue request for async processing",
				"request_id": requestID,
				"details":    err.Error(),
			})
			return
		}
	} else {
		m.logger.Warn("pub/sub topic not configured, request not published",
			zap.String("request_id", requestID),
		)
		
		// Return helpful error for unconfigured route
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "route_not_configured",
			"message": "This route is not configured for sync or async processing",
			"path":    r.URL.Path,
			"hint":    "Check your ROUTES_CONFIG environment variable or use a configured route like /mock/**",
		})
		return
	}

	// Return 202 Accepted with status URL
	response := map[string]interface{}{
		"request_id": requestID,
		"status":     "accepted",
		"status_url": fmt.Sprintf("%s/status/%s", m.baseURL, requestID),
		"stream_url": fmt.Sprintf("%s/stream/%s", m.baseURL, requestID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)

	m.logger.Info("request accepted",
		zap.String("request_id", requestID),
		zap.String("tenant_id", tenantID),
	)
}

// extractHeaders extracts important headers from the request
func extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)

	// List of headers to preserve
	preserveHeaders := []string{
		"Content-Type",
		"Accept",
		"User-Agent",
		"X-Forwarded-For",
		"X-Real-IP",
		"Authorization",
	}

	for _, key := range preserveHeaders {
		if value := r.Header.Get(key); value != "" {
			headers[key] = value
		}
	}

	// Copy all X- headers (custom headers)
	for key, values := range r.Header {
		if len(key) > 2 && key[:2] == "X-" {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
	}

	return headers
}

// Close closes the Pub/Sub topic
func (m *Matcher) Close() error {
	if m.topic != nil {
		m.topic.Stop()
	}
	return nil
}
