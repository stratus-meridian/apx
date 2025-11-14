package health_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stratus-meridian/apx-router-open-core/pkg/health"
	"go.uber.org/zap"
)

// Example demonstrates how to use the health checker
func Example() {
	// Create a logger (use zap.NewNop() for silent mode)
	logger := zap.NewNop()

	// Initialize health checker with nil components for demonstration
	// In production, pass real policyStore and pubsubTopic instances
	healthChecker := health.NewChecker(
		nil,   // policyStore
		nil,   // pubsubTopic
		false, // observabilityInit
		logger,
	)

	// Perform health check
	ctx := context.Background()
	healthResponse := healthChecker.CheckHealth(ctx)

	// Marshal to JSON for display
	data, _ := json.MarshalIndent(healthResponse, "", "  ")
	fmt.Println(string(data))

	// Output will vary based on timestamp, so we don't include it in example output
}

// ExampleChecker_determineOverallStatus demonstrates the overall status logic
func ExampleChecker_determineOverallStatus() {
	// When all components are healthy, overall status is healthy
	fmt.Println("All healthy:", determineStatus("healthy", "healthy", "healthy"))

	// When a critical component is down, overall is down
	fmt.Println("Firestore down:", determineStatus("down", "healthy", "healthy"))
	fmt.Println("PubSub down:", determineStatus("healthy", "down", "healthy"))

	// When any component is degraded (but none down), overall is degraded
	fmt.Println("BigQuery degraded:", determineStatus("healthy", "healthy", "degraded"))

	// Output:
	// All healthy: healthy
	// Firestore down: down
	// PubSub down: down
	// BigQuery degraded: degraded
}

// Helper function to simulate status determination
func determineStatus(firestore, pubsub, bigquery string) string {
	if firestore == "down" || pubsub == "down" {
		return "down"
	}
	if firestore == "degraded" || pubsub == "degraded" || bigquery == "degraded" {
		return "degraded"
	}
	return "healthy"
}
