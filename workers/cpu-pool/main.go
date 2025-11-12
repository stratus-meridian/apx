package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RequestMessage matches router's RequestMessage
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

type Worker struct {
	logger      *zap.Logger
	redisClient *redis.Client
	projectID   string
	region      string
}

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := context.Background()

	// Configuration
	projectID := getEnv("GCP_PROJECT_ID", "apx-dev")
	region := getEnv("GCP_REGION", "us-central1")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	logger.Info("connected to Redis", zap.String("addr", redisAddr))

	// Initialize worker
	worker := &Worker{
		logger:      logger,
		redisClient: redisClient,
		projectID:   projectID,
		region:      region,
	}

	// Initialize Pub/Sub client
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		logger.Fatal("failed to create pubsub client", zap.Error(err))
	}
	defer client.Close()

	// Get subscription
	subscriptionName := fmt.Sprintf("apx-workers-%s", region)
	sub := client.Subscription(subscriptionName)

	// Check if subscription exists, create if not
	exists, err := sub.Exists(ctx)
	if err != nil {
		logger.Fatal("failed to check subscription", zap.Error(err))
	}

	if !exists {
		topicName := fmt.Sprintf("apx-requests-%s", region)
		topic := client.Topic(topicName)

		logger.Info("creating subscription",
			zap.String("subscription", subscriptionName),
			zap.String("topic", topicName))

		sub, err = client.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{
			Topic:                 topic,
			EnableMessageOrdering: true,
			AckDeadline:           60 * time.Second,
		})
		if err != nil {
			logger.Fatal("failed to create subscription", zap.Error(err))
		}
	}

	logger.Info("worker started",
		zap.String("subscription", subscriptionName),
		zap.String("region", region))

	// Handle shutdown gracefully
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutdown signal received")
		cancel()
	}()

	// Start receiving messages
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		worker.processMessage(ctx, msg)
	})

	if err != nil && err != context.Canceled {
		logger.Error("subscription receive error", zap.Error(err))
	}

	logger.Info("worker stopped")
}

func (w *Worker) processMessage(ctx context.Context, msg *pubsub.Message) {
	// Parse request message
	var reqMsg RequestMessage
	if err := json.Unmarshal(msg.Data, &reqMsg); err != nil {
		w.logger.Error("failed to unmarshal message", zap.Error(err))
		msg.Nack()
		return
	}

	w.logger.Info("processing request",
		zap.String("request_id", reqMsg.RequestID),
		zap.String("tenant_id", reqMsg.TenantID),
		zap.String("route", reqMsg.Route))

	// Update status to processing
	if err := w.updateStatus(ctx, reqMsg.RequestID, "processing", 0, nil, ""); err != nil {
		w.logger.Error("failed to update status", zap.Error(err))
	}

	// Simulate work (replace with actual AI inference)
	time.Sleep(100 * time.Millisecond)

	// Simulate progress updates
	for progress := 25; progress <= 75; progress += 25 {
		if err := w.updateStatus(ctx, reqMsg.RequestID, "processing", progress, nil, ""); err != nil {
			w.logger.Error("failed to update progress", zap.Error(err))
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Complete with result
	result := map[string]interface{}{
		"message":      "Request processed successfully",
		"request_id":   reqMsg.RequestID,
		"processed_at": time.Now().Format(time.RFC3339),
		"tenant_id":    reqMsg.TenantID,
	}

	if err := w.updateStatus(ctx, reqMsg.RequestID, "complete", 100, result, ""); err != nil {
		w.logger.Error("failed to update final status", zap.Error(err))
		msg.Nack()
		return
	}

	w.logger.Info("request completed",
		zap.String("request_id", reqMsg.RequestID),
		zap.String("tenant_id", reqMsg.TenantID))

	msg.Ack()
}

func (w *Worker) updateStatus(ctx context.Context, requestID, status string, progress int, result interface{}, errorMsg string) error {
	// Build status record
	statusData := map[string]interface{}{
		"request_id": requestID,
		"status":     status,
		"progress":   progress,
		"updated_at": time.Now().Format(time.RFC3339),
	}

	if result != nil {
		statusData["result"] = result
	}
	if errorMsg != "" {
		statusData["error"] = errorMsg
	}

	// Serialize to JSON
	data, err := json.Marshal(statusData)
	if err != nil {
		return err
	}

	// Store in Redis
	key := fmt.Sprintf("status:%s", requestID)
	return w.redisClient.Set(ctx, key, data, 24*time.Hour).Err()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
