// Package health provides health check functionality for the APX router
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/stratus-meridian/apx/router/internal/policy"
	"go.uber.org/zap"
)

// ComponentStatus represents the health status of a component
type ComponentStatus string

const (
	StatusHealthy  ComponentStatus = "healthy"
	StatusDegraded ComponentStatus = "degraded"
	StatusDown     ComponentStatus = "down"
)

// HealthResponse represents the complete health check response
type HealthResponse struct {
	Status     ComponentStatus `json:"status"`
	Version    string          `json:"version"`
	Timestamp  string          `json:"timestamp"`
	Components Components      `json:"components"`
}

// Components holds the health status of each component
type Components struct {
	Firestore ComponentStatus `json:"firestore"`
	PubSub    ComponentStatus `json:"pubsub"`
	BigQuery  ComponentStatus `json:"bigquery"`
}

// Checker performs health checks on router components
type Checker struct {
	policyStore       *policy.Store
	pubsubTopic       *pubsub.Topic
	observabilityInit bool
	logger            *zap.Logger
}

// NewChecker creates a new health checker
func NewChecker(policyStore *policy.Store, pubsubTopic *pubsub.Topic, observabilityInit bool, logger *zap.Logger) *Checker {
	return &Checker{
		policyStore:       policyStore,
		pubsubTopic:       pubsubTopic,
		observabilityInit: observabilityInit,
		logger:            logger,
	}
}

// CheckHealth performs all health checks and returns the complete health response
func (c *Checker) CheckHealth(ctx context.Context) HealthResponse {
	// Get component health statuses
	firestoreStatus := c.checkFirestore()
	pubsubStatus := c.checkPubSub(ctx)
	bigqueryStatus := c.checkBigQuery()

	// Determine overall status
	overallStatus := c.determineOverallStatus(firestoreStatus, pubsubStatus, bigqueryStatus)

	// Get version from environment or default
	version := os.Getenv("VERSION")
	if version == "" {
		version = "dev"
	}

	return HealthResponse{
		Status:    overallStatus,
		Version:   version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Components: Components{
			Firestore: firestoreStatus,
			PubSub:    pubsubStatus,
			BigQuery:  bigqueryStatus,
		},
	}
}

// checkFirestore checks Firestore health via policy store
func (c *Checker) checkFirestore() ComponentStatus {
	// If policy store is nil, Firestore is down
	if c.policyStore == nil {
		c.logger.Debug("firestore health check: policy store is nil")
		return StatusDown
	}

	// Check if policy store is ready
	if !c.policyStore.IsReady() {
		c.logger.Debug("firestore health check: policy store not ready")
		return StatusDown
	}

	c.logger.Debug("firestore health check: healthy")
	return StatusHealthy
}

// checkPubSub checks Pub/Sub health
func (c *Checker) checkPubSub(ctx context.Context) ComponentStatus {
	// If topic is nil, Pub/Sub is down
	if c.pubsubTopic == nil {
		c.logger.Debug("pubsub health check: topic is nil")
		return StatusDown
	}

	// Create a timeout context for non-blocking health check
	checkCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// Try to get topic configuration (lightweight check)
	_, err := c.pubsubTopic.Config(checkCtx)
	if err != nil {
		// If we get a timeout or error, consider it degraded
		c.logger.Debug("pubsub health check: failed to get config",
			zap.Error(err))
		return StatusDegraded
	}

	// If we can get the config, topic is healthy
	c.logger.Debug("pubsub health check: healthy")
	return StatusHealthy
}

// checkBigQuery checks BigQuery health
// Note: Router doesn't directly use BigQuery (workers do), so we check if
// observability is initialized since that's where BigQuery logging happens
func (c *Checker) checkBigQuery() ComponentStatus {
	// BigQuery is used asynchronously by workers, not directly by router
	// We consider it healthy if observability was initialized successfully
	// since observability may use BigQuery for logs/metrics
	if c.observabilityInit {
		c.logger.Debug("bigquery health check: healthy (observability initialized)")
		return StatusHealthy
	}

	// If observability failed to init, BigQuery might be unavailable
	// but this is non-critical for router operation
	c.logger.Debug("bigquery health check: degraded (observability not initialized)")
	return StatusDegraded
}

// determineOverallStatus calculates overall health based on component statuses
func (c *Checker) determineOverallStatus(firestore, pubsub, bigquery ComponentStatus) ComponentStatus {
	// If any critical component is down, overall is down
	// Critical components: Firestore (policies) and Pub/Sub (message routing)
	if firestore == StatusDown || pubsub == StatusDown {
		return StatusDown
	}

	// If any component is degraded, overall is degraded
	if firestore == StatusDegraded || pubsub == StatusDegraded || bigquery == StatusDegraded {
		return StatusDegraded
	}

	// All components healthy
	return StatusHealthy
}

// Handler returns an HTTP handler for the health endpoint
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Perform health check
		health := c.CheckHealth(ctx)

		// Determine HTTP status code based on health
		statusCode := http.StatusOK
		if health.Status == StatusDown {
			statusCode = http.StatusServiceUnavailable
		} else if health.Status == StatusDegraded {
			statusCode = http.StatusOK // Still return 200 for degraded
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(health); err != nil {
			c.logger.Error("failed to encode health response", zap.Error(err))
		}
	}
}
