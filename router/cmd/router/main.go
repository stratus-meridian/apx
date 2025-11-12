package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/apx/router/internal/config"
	"github.com/apx/router/internal/middleware"
	"github.com/apx/router/internal/policy"
	"github.com/apx/router/internal/ratelimit"
	"github.com/apx/router/internal/routes"
	"github.com/apx/router/pkg/observability"
	"github.com/apx/router/pkg/status"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer logger.Sync()

	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Initialize observability (OTEL, metrics)
	shutdown, err := observability.Init(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize observability", zap.Error(err))
	}
	defer shutdown()

	// Initialize policy store (loads compiled artifacts from GCS/Firestore)
	policyStore, err := policy.NewStore(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize policy store", zap.Error(err))
	}

	// Initialize Redis client for status storage
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to connect to Redis, status storage will not work", zap.Error(err))
	} else {
		logger.Info("connected to Redis", zap.String("addr", cfg.RedisAddr))
	}

	// Initialize status store
	statusStore := status.NewRedisStore(redisClient, 24*time.Hour)

	// Initialize rate limiter with token bucket algorithm
	rateLimiter := ratelimit.NewRedisLimiter(redisClient, logger)

	// Initialize Pub/Sub client and topic
	var pubsubTopic *pubsub.Topic

	if cfg.Environment == "dev" || cfg.Environment == "local" {
		// Use Pub/Sub emulator
		emulatorHost := os.Getenv("PUBSUB_EMULATOR_HOST")
		logger.Info("connecting to Pub/Sub emulator",
			zap.String("host", emulatorHost),
			zap.String("project", cfg.ProjectID))

		client, err := pubsub.NewClient(ctx, cfg.ProjectID)
		if err != nil {
			logger.Fatal("failed to create pubsub client", zap.Error(err))
		}
		defer client.Close()

		// Get or create topic
		topicName := fmt.Sprintf("apx-requests-%s", cfg.Region)
		topic := client.Topic(topicName)

		exists, err := topic.Exists(ctx)
		if err != nil {
			logger.Fatal("failed to check topic existence", zap.Error(err))
		}

		if !exists {
			logger.Info("creating Pub/Sub topic", zap.String("topic", topicName))
			topic, err = client.CreateTopic(ctx, topicName)
			if err != nil {
				logger.Fatal("failed to create topic", zap.Error(err))
			}
		}

		// Enable message ordering for FIFO per tenant
		topic.EnableMessageOrdering = true

		pubsubTopic = topic
		logger.Info("pub/sub topic ready",
			zap.String("topic", topicName),
			zap.Bool("ordering_enabled", true))
	} else {
		// Production would use real GCP Pub/Sub
		logger.Warn("pub/sub not configured for production environment")
	}

	// Construct base URL for status/stream URLs
	baseURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
	if cfg.Environment == "production" {
		baseURL = "https://api.apx.dev" // Production URL
	}

	// Initialize route matcher with real topic
	routeMatcher := routes.NewMatcher(pubsubTopic, statusStore, logger, baseURL)

	// Create HTTP router
	r := mux.NewRouter()

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"apx-router"}`))
	}).Methods(http.MethodGet)

	// Readiness check (verifies policy store is ready)
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if policyStore.IsReady() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ready"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready"}`))
		}
	}).Methods(http.MethodGet)

	// Status endpoint - GET /status/{request_id}
	// This endpoint is exempt from rate limiting (per V-004 requirements)
	r.HandleFunc("/status/{request_id}", routeMatcher.HandleStatus).Methods(http.MethodGet)

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	// Main routing handler
	// Note: Rate limiting is applied AFTER tenant context extraction
	r.PathPrefix("/").Handler(
		middleware.Chain(
			http.HandlerFunc(routeMatcher.Handle),
			middleware.RequestID(logger),
			middleware.TenantContext(logger),
			middleware.RateLimit(rateLimiter, logger), // Token bucket rate limiting
			middleware.PolicyVersionTag(policyStore, logger),
			middleware.Metrics(),
			middleware.Logging(logger),
			middleware.Tracing(),
		),
	)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting router service",
			zap.Int("port", cfg.Port),
			zap.String("environment", cfg.Environment),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down router service")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("router service stopped")
}
