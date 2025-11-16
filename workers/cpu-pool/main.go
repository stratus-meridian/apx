package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/firestore"
	"github.com/redis/go-redis/v9"
	"github.com/apx/control/pkg/opa"
	"github.com/stratus-meridian/apx/router/pkg/status"
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
	httpClient  *http.Client
	statusStore status.Store
	policyEval  *PolicyEvaluator
}

// PolicyEvaluator loads policy metadata from Firestore and evaluates Rego policies.
type PolicyEvaluator struct {
	client      *firestore.Client
	collection  string
	logger      *zap.Logger
	enabled     bool
}

// Evaluate returns true if the request is allowed by the active policy.
// It fails open on errors (logs and returns allowed=true) and only enforces
// explicit deny decisions when evaluation succeeds and returns false.
func (p *PolicyEvaluator) Evaluate(ctx context.Context, req *RequestMessage) (bool, error) {
	if !p.enabled || p.client == nil || req.PolicyVersion == "" {
		// No policy configured for this request.
		return true, nil
	}

	policyRef := req.PolicyVersion

	doc, err := p.client.Collection(p.collection).Doc(policyRef).Get(ctx)
	if err != nil {
		// If policy metadata is missing, treat as no policy.
		p.logger.Warn("policy document not found; allowing request",
			zap.String("policy_ref", policyRef),
			zap.Error(err))
		return true, nil
	}

	var meta struct {
		AuthzRego string `firestore:"authz_rego"`
	}
	if err := doc.DataTo(&meta); err != nil {
		p.logger.Warn("failed to decode policy document; allowing request",
			zap.String("policy_ref", policyRef),
			zap.Error(err))
		return true, nil
	}

	if strings.TrimSpace(meta.AuthzRego) == "" {
		// No authz Rego configured; allow.
		return true, nil
	}

	engine, err := opa.NewEngine(ctx, meta.AuthzRego, "data.apx.allow")
	if err != nil {
		p.logger.Warn("failed to create OPA engine; allowing request",
			zap.String("policy_ref", policyRef),
			zap.Error(err))
		return true, nil
	}

	input := map[string]interface{}{
		"method": req.Method,
		"route":  req.Route,
		"tenant": map[string]interface{}{
			"id":   req.TenantID,
			"tier": req.TenantTier,
		},
		"headers": req.Headers,
	}

	allowed, err := engine.Eval(ctx, input)
	if err != nil {
		p.logger.Warn("OPA evaluation error; allowing request",
			zap.String("policy_ref", policyRef),
			zap.Error(err))
		return true, nil
	}

	return allowed, nil
}

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := context.Background()

	// Configuration
	projectID := getEnv("GCP_PROJECT_ID", "apx-dev")
	region := getEnv("GCP_REGION", "us-central1")
	subscriptionID := getEnv("PUBSUB_SUBSCRIPTION", "apx-workers-us")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	logger.Info("connected to Redis", zap.String("addr", redisAddr))

	// Initialize status store (shared format with router + streaming aggregator)
	statusStore := status.NewRedisStore(redisClient, 24*time.Hour)

	// Initialize policy evaluator (fail-open if Firestore is unavailable)
	var policyEval *PolicyEvaluator
	fsClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		logger.Warn("policy evaluation disabled (failed to create firestore client)",
			zap.Error(err),
			zap.String("project_id", projectID),
		)
	} else {
		policyEval = &PolicyEvaluator{
			client:     fsClient,
			collection: getEnv("POLICY_COLLECTION", "policies"),
			logger:     logger,
			enabled:    true,
		}
		logger.Info("policy evaluation enabled",
			zap.String("project_id", projectID),
			zap.String("collection", policyEval.collection),
		)
		defer fsClient.Close()
	}

	// Initialize worker
	worker := &Worker{
		logger:      logger,
		redisClient: redisClient,
		projectID:   projectID,
		region:      region,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		statusStore: statusStore,
		policyEval:  policyEval,
	}

	// Initialize Pub/Sub client
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		logger.Fatal("failed to create pubsub client", zap.Error(err))
	}
	defer client.Close()

	// Get subscription (already created by Terraform)
	sub := client.Subscription(subscriptionID)

	// Verify subscription exists
	exists, err := sub.Exists(ctx)
	if err != nil {
		logger.Fatal("failed to check subscription existence", zap.Error(err))
	}
	if !exists {
		logger.Fatal("subscription does not exist - must be created by Terraform",
			zap.String("subscription", subscriptionID))
	}

	logger.Info("worker started",
		zap.String("subscription", subscriptionID),
		zap.String("region", region))

	// Handle shutdown gracefully
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start HTTP health endpoint for Cloud Run
	port := getEnv("PORT", "8080")
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "apx-worker-cpu",
		})
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check if Pub/Sub subscription is accessible
		exists, err := sub.Exists(ctx)
		if err != nil || !exists {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not ready",
				"reason": "subscription unavailable",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	})

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Start HTTP server in background
	go func() {
		logger.Info("starting health endpoint", zap.String("port", port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", zap.Error(err))
		}
	}()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("shutdown signal received")
		cancel()
		httpServer.Shutdown(context.Background())
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

	// Evaluate policy if enabled; deny if explicitly not allowed.
	if w.policyEval != nil && w.policyEval.enabled {
		allowed, err := w.policyEval.Evaluate(ctx, &reqMsg)
		if err != nil {
			w.logger.Warn("policy evaluation error - failing open",
				zap.String("request_id", reqMsg.RequestID),
				zap.Error(err))
		} else if !allowed {
			w.logger.Info("request denied by policy",
				zap.String("request_id", reqMsg.RequestID),
				zap.String("tenant_id", reqMsg.TenantID),
				zap.String("policy_version", reqMsg.PolicyVersion))
			_ = w.updateStatus(ctx, reqMsg.RequestID, "failed", 100, nil, "request denied by policy")
			msg.Ack()
			return
		}
	}

	// Mark as processing (0% initially)
	if err := w.updateStatus(ctx, reqMsg.RequestID, "processing", 0, nil, ""); err != nil {
		w.logger.Error("failed to set status=processing", zap.Error(err))
	}

	// Perform backend call
	result, err := w.callBackend(ctx, &reqMsg)
	if err != nil {
		w.logger.Error("backend call failed",
			zap.String("request_id", reqMsg.RequestID),
			zap.Error(err))
		_ = w.updateStatus(ctx, reqMsg.RequestID, "failed", 100, nil, err.Error())
		msg.Ack() // Ack to avoid redelivery storms; status carries failure
		return
	}

	// Mark as complete with result
	if err := w.updateStatus(ctx, reqMsg.RequestID, "complete", 100, result, ""); err != nil {
		w.logger.Error("failed to set status=complete", zap.Error(err))
		msg.Nack()
		return
	}

	w.logger.Info("request completed",
		zap.String("request_id", reqMsg.RequestID),
		zap.String("tenant_id", reqMsg.TenantID))

	msg.Ack()
}

// callBackend forwards the request to the configured backend.
// For now it uses the Route field as the absolute URL to call.
func (w *Worker) callBackend(ctx context.Context, req *RequestMessage) (map[string]interface{}, error) {
	backendURL := req.Route
	if backendURL == "" {
		return nil, fmt.Errorf("empty route in request message")
	}

	// Build HTTP request
	var body io.Reader
	if len(req.Body) > 0 && req.Method != http.MethodGet && req.Method != http.MethodHead {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, backendURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build backend request: %w", err)
	}

	// Copy selected headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := w.httpClient.Do(httpReq)
	latency := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read backend response: %w", err)
	}

	w.logger.Info("backend response",
		zap.String("request_id", req.RequestID),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("latency", latency))

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        string(respBody),
		"latency_ms":  latency.Milliseconds(),
		"route":       req.Route,
	}, nil
}

func (w *Worker) updateStatus(ctx context.Context, requestID, status string, progress int, result interface{}, errorMsg string) error {
	switch status {
	case "processing":
		if err := w.statusStore.UpdateStatus(ctx, requestID, status.StatusProcessing); err != nil {
			return err
		}
		if progress > 0 && progress < 100 {
			return w.statusStore.UpdateProgress(ctx, requestID, progress)
		}
	case "complete":
		var raw json.RawMessage
		if result != nil {
			b, err := json.Marshal(result)
			if err != nil {
				return err
			}
			raw = json.RawMessage(b)
		}
		return w.statusStore.SetResult(ctx, requestID, raw)
	case "failed":
		if errorMsg == "" {
			errorMsg = "request failed"
		}
		return w.statusStore.SetError(ctx, requestID, errorMsg)
	default:
		// Unknown status string; do nothing to avoid corrupting records.
		return nil
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
