package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestHealthResponse_Format(t *testing.T) {
	// Test that the health response matches expected JSON schema
	response := HealthResponse{
		Status:    StatusHealthy,
		Version:   "1.0.0",
		Timestamp: "2025-01-14T12:00:00Z",
		Components: Components{
			Firestore: StatusHealthy,
			PubSub:    StatusHealthy,
			BigQuery:  StatusHealthy,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal health response: %v", err)
	}

	// Verify JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}

	// Verify top-level fields
	if parsed["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %v", parsed["status"])
	}
	if parsed["version"] != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %v", parsed["version"])
	}
	if parsed["timestamp"] != "2025-01-14T12:00:00Z" {
		t.Errorf("expected timestamp=2025-01-14T12:00:00Z, got %v", parsed["timestamp"])
	}

	// Verify components object
	components, ok := parsed["components"].(map[string]interface{})
	if !ok {
		t.Fatalf("components is not an object")
	}
	if components["firestore"] != "healthy" {
		t.Errorf("expected firestore=healthy, got %v", components["firestore"])
	}
	if components["pubsub"] != "healthy" {
		t.Errorf("expected pubsub=healthy, got %v", components["pubsub"])
	}
	if components["bigquery"] != "healthy" {
		t.Errorf("expected bigquery=healthy, got %v", components["bigquery"])
	}
}

func TestDetermineOverallStatus(t *testing.T) {
	logger := zap.NewNop()
	checker := NewChecker(nil, nil, false, logger)

	tests := []struct {
		name      string
		firestore ComponentStatus
		pubsub    ComponentStatus
		bigquery  ComponentStatus
		expected  ComponentStatus
	}{
		{
			name:      "all healthy",
			firestore: StatusHealthy,
			pubsub:    StatusHealthy,
			bigquery:  StatusHealthy,
			expected:  StatusHealthy,
		},
		{
			name:      "firestore down",
			firestore: StatusDown,
			pubsub:    StatusHealthy,
			bigquery:  StatusHealthy,
			expected:  StatusDown,
		},
		{
			name:      "pubsub down",
			firestore: StatusHealthy,
			pubsub:    StatusDown,
			bigquery:  StatusHealthy,
			expected:  StatusDown,
		},
		{
			name:      "bigquery degraded",
			firestore: StatusHealthy,
			pubsub:    StatusHealthy,
			bigquery:  StatusDegraded,
			expected:  StatusDegraded,
		},
		{
			name:      "firestore degraded",
			firestore: StatusDegraded,
			pubsub:    StatusHealthy,
			bigquery:  StatusHealthy,
			expected:  StatusDegraded,
		},
		{
			name:      "multiple degraded",
			firestore: StatusDegraded,
			pubsub:    StatusDegraded,
			bigquery:  StatusDegraded,
			expected:  StatusDegraded,
		},
		{
			name:      "down takes precedence over degraded",
			firestore: StatusDown,
			pubsub:    StatusDegraded,
			bigquery:  StatusHealthy,
			expected:  StatusDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.determineOverallStatus(tt.firestore, tt.pubsub, tt.bigquery)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCheckFirestore(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name        string
		policyStore interface{} // nil or mock
		expected    ComponentStatus
	}{
		{
			name:        "nil policy store",
			policyStore: nil,
			expected:    StatusDown,
		},
		// Note: Testing with real policy store would require Firestore setup
		// In practice, we test this through integration tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker(nil, nil, false, logger)
			result := checker.checkFirestore()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCheckPubSub(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	tests := []struct {
		name     string
		topic    interface{} // nil or mock
		expected ComponentStatus
	}{
		{
			name:     "nil topic",
			topic:    nil,
			expected: StatusDown,
		},
		// Note: Testing with real topic would require Pub/Sub setup
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker(nil, nil, false, logger)
			result := checker.checkPubSub(ctx)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCheckBigQuery(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name              string
		observabilityInit bool
		expected          ComponentStatus
	}{
		{
			name:              "observability initialized",
			observabilityInit: true,
			expected:          StatusHealthy,
		},
		{
			name:              "observability not initialized",
			observabilityInit: false,
			expected:          StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker(nil, nil, tt.observabilityInit, logger)
			result := checker.checkBigQuery()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name               string
		observabilityInit  bool
		expectedStatus     int
		expectedHealthEnum ComponentStatus
	}{
		{
			name:               "all components down (nil)",
			observabilityInit:  false,
			expectedStatus:     http.StatusServiceUnavailable,
			expectedHealthEnum: StatusDown,
		},
		{
			name:               "observability init (bigquery degraded)",
			observabilityInit:  true,
			expectedStatus:     http.StatusServiceUnavailable, // Still down because firestore/pubsub nil
			expectedHealthEnum: StatusDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker(nil, nil, tt.observabilityInit, logger)
			handler := checker.Handler()

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify response structure
			var response HealthResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Status != tt.expectedHealthEnum {
				t.Errorf("expected status %s, got %s", tt.expectedHealthEnum, response.Status)
			}

			// Verify required fields are present
			if response.Version == "" {
				t.Error("version is empty")
			}
			if response.Timestamp == "" {
				t.Error("timestamp is empty")
			}
		})
	}
}

func TestVersionFromEnvironment(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	// Test with VERSION env var set
	os.Setenv("VERSION", "1.2.3")
	defer os.Unsetenv("VERSION")

	checker := NewChecker(nil, nil, false, logger)
	health := checker.CheckHealth(ctx)

	if health.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", health.Version)
	}

	// Test without VERSION env var
	os.Unsetenv("VERSION")
	health = checker.CheckHealth(ctx)

	if health.Version != "dev" {
		t.Errorf("expected version dev, got %s", health.Version)
	}
}

func TestTimestampFormat(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	checker := NewChecker(nil, nil, false, logger)
	health := checker.CheckHealth(ctx)

	// Verify timestamp is in ISO 8601 format (RFC3339)
	if health.Timestamp == "" {
		t.Error("timestamp is empty")
	}

	// Try to parse it back using time.Parse with RFC3339 layout
	_, err := time.Parse(time.RFC3339, health.Timestamp)
	if err != nil {
		t.Errorf("timestamp is not valid RFC3339: %v", err)
	}
}
