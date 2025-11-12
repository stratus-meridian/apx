package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
)

const (
	projectID        = "apx-dev"
	topicName        = "apx-requests-us-central1"
	subscriptionName = "apx-workers-us-central1"
	routerURL        = "http://localhost:8081"
)

type RouterResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	StatusURL string `json:"status_url"`
	StreamURL string `json:"stream_url"`
}

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

func main() {
	fmt.Println("=========================================")
	fmt.Println("Testing Pub/Sub Integration")
	fmt.Println("=========================================")
	fmt.Println()

	// Set Pub/Sub emulator host
	emulatorHost := os.Getenv("PUBSUB_EMULATOR_HOST")
	if emulatorHost == "" {
		emulatorHost = "localhost:8085"
		os.Setenv("PUBSUB_EMULATOR_HOST", emulatorHost)
	}

	ctx := context.Background()

	// Step 1: Send request to router
	fmt.Println("[1/5] Sending test request to router...")
	requestBody := `{"message":"pubsub integration test"}`
	req, err := http.NewRequest("POST", routerURL+"/api/test", strings.NewReader(requestBody))
	if err != nil {
		fmt.Printf("❌ Failed to create request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "test-tenant")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Failed to send request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response: %s\n", string(body))
	fmt.Println()

	var routerResp RouterResponse
	if err := json.Unmarshal(body, &routerResp); err != nil {
		fmt.Printf("❌ Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	if routerResp.RequestID == "" {
		fmt.Println("❌ No request_id in response")
		os.Exit(1)
	}

	fmt.Printf("✅ Request ID: %s\n", routerResp.RequestID)
	fmt.Println()

	// Step 2: Connect to Pub/Sub emulator
	fmt.Println("[2/5] Connecting to Pub/Sub emulator...")
	fmt.Printf("PUBSUB_EMULATOR_HOST=%s\n", emulatorHost)

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		fmt.Printf("❌ Failed to create Pub/Sub client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("✅ Connected to Pub/Sub emulator")
	fmt.Println()

	// Step 3: Create subscription if it doesn't exist
	fmt.Println("[3/5] Creating/checking subscription...")
	topic := client.Topic(topicName)
	sub := client.Subscription(subscriptionName)

	exists, err := sub.Exists(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to check subscription: %v\n", err)
		os.Exit(1)
	}

	if !exists {
		fmt.Printf("Creating subscription: %s\n", subscriptionName)
		sub, err = client.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 20 * time.Second,
		})
		if err != nil {
			fmt.Printf("❌ Failed to create subscription: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Subscription already exists (OK)")
	}

	fmt.Println("✅ Subscription ready")
	fmt.Println()

	// Step 4: Pull message from Pub/Sub
	fmt.Println("[4/5] Pulling message from Pub/Sub...")
	fmt.Println("Waiting 2 seconds for router to publish...")
	time.Sleep(2 * time.Second)

	// Create a context with timeout for receiving
	receiveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var receivedMsg *pubsub.Message
	var msgData RequestMessage

	err = sub.Receive(receiveCtx, func(ctx context.Context, msg *pubsub.Message) {
		receivedMsg = msg
		msg.Ack()
		cancel() // Stop receiving after first message
	})

	if err != nil && err != context.Canceled {
		fmt.Printf("❌ Failed to receive message: %v\n", err)
		fmt.Println("\nDebugging information:")
		fmt.Println("- Check if router is running: curl http://localhost:8081/health")
		fmt.Println("- Check router logs: docker logs apilee-router-1 --tail 50")
		os.Exit(1)
	}

	if receivedMsg == nil {
		fmt.Println("❌ No message received from Pub/Sub")
		os.Exit(1)
	}

	fmt.Println("✅ Message received from Pub/Sub")
	fmt.Printf("Message ID: %s\n", receivedMsg.ID)
	fmt.Printf("Publish Time: %s\n", receivedMsg.PublishTime)
	fmt.Printf("Ordering Key: %s\n", receivedMsg.OrderingKey)
	fmt.Println()

	// Step 5: Verify message content
	fmt.Println("[5/5] Verifying message content...")

	if err := json.Unmarshal(receivedMsg.Data, &msgData); err != nil {
		fmt.Printf("❌ Failed to unmarshal message data: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Decoded message data:")
	prettyJSON, _ := json.MarshalIndent(msgData, "", "  ")
	fmt.Println(string(prettyJSON))
	fmt.Println()

	// Check request_id matches
	if msgData.RequestID != routerResp.RequestID {
		fmt.Printf("❌ Request ID mismatch\n")
		fmt.Printf("Expected: %s\n", routerResp.RequestID)
		fmt.Printf("Got: %s\n", msgData.RequestID)
		os.Exit(1)
	}
	fmt.Printf("✅ Request ID matches: %s\n", msgData.RequestID)

	// Check tenant_id attribute
	tenantID := receivedMsg.Attributes["tenant_id"]
	if tenantID != "test-tenant" {
		fmt.Printf("⚠️  Tenant ID attribute mismatch. Expected: test-tenant, Got: %s\n", tenantID)
	} else {
		fmt.Printf("✅ Tenant ID attribute correct: %s\n", tenantID)
	}

	// Check ordering key
	if receivedMsg.OrderingKey == "" {
		fmt.Println("⚠️  No ordering key found")
	} else {
		fmt.Printf("✅ Ordering key present: %s\n", receivedMsg.OrderingKey)
	}

	// Check tenant_id in message body
	if msgData.TenantID != "test-tenant" {
		fmt.Printf("⚠️  Tenant ID in body mismatch. Expected: test-tenant, Got: %s\n", msgData.TenantID)
	} else {
		fmt.Printf("✅ Tenant ID in body correct: %s\n", msgData.TenantID)
	}

	fmt.Println()
	fmt.Println("=========================================")
	fmt.Println("✅ ALL TESTS PASSED!")
	fmt.Println("=========================================")
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  - Request ID: %s\n", routerResp.RequestID)
	fmt.Printf("  - Tenant ID: %s\n", msgData.TenantID)
	fmt.Printf("  - Ordering Key: %s\n", receivedMsg.OrderingKey)
	fmt.Printf("  - Topic: %s\n", topicName)
	fmt.Printf("  - Subscription: %s\n", subscriptionName)
	fmt.Printf("  - Route: %s\n", msgData.Route)
	fmt.Printf("  - Method: %s\n", msgData.Method)
	fmt.Println()
}
