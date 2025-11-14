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
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/stratus-meridian/apx-router-open-core/internal/auth"
	"github.com/stratus-meridian/apx-router-open-core/internal/config"
	"github.com/stratus-meridian/apx-router-open-core/internal/middleware"
	"github.com/stratus-meridian/apx-router-open-core/internal/policy"
	"github.com/stratus-meridian/apx-router-open-core/internal/routes"
	"github.com/stratus-meridian/apx-router-open-core/pkg/health"
	"github.com/stratus-meridian/apx-router-open-core/pkg/observability"
	"github.com/stratus-meridian/apx-router-open-core/pkg/status"
)

func main() {
	// Use fmt.Println for early errors before logger is initialized
	fmt.Println("Starting APX Router (Open-Core Edition)...")
	fmt.Println("This is the open-source gateway. For commercial features (billing, analytics, control plane), visit https://apx.build")

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("APX Router Open-Core Edition starting")

	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	logger.Info("Configuration loaded",
		zap.String("project_id", cfg.ProjectID),
		zap.String("region", cfg.Region))

	// Initialize observability (OTEL, metrics) - optional for open-core
	shutdown, err := observability.Init(ctx, cfg, logger)
	observabilityInit := err == nil
	if err != nil {
		logger.Warn("observability disabled (optional in open-core)", zap.Error(err))
		shutdown = func() {}
	} else {
		logger.Info("Observability initialized")
	}
	defer shutdown()

	// Initialize policy store (no-op in open-core)
	policyStore, err := policy.NewStore(ctx, logger)
	if err != nil {
		logger.Warn("policy store initialization failed, using no-op", zap.Error(err))
		policyStore = policy.NewNoOpStore(logger)
	}

	// Initialize Redis client for status storage (optional)
	var redisClient *redis.Client
	var statusStore *status.RedisStore

	if cfg.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:         cfg.RedisAddr,
			Password:     "",
			DB:           0,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})

		// Test Redis connection
		if err := redisClient.Ping(ctx).Err(); err != nil {
			logger.Warn("Redis not available, status tracking disabled",
				zap.Error(err),
				zap.String("redis_addr", cfg.RedisAddr))
			redisClient = nil
		} else {
			logger.Info("connected to Redis", zap.String("addr", cfg.RedisAddr))
			statusStore = status.NewRedisStore(redisClient, 24*time.Hour)
		}
	} else {
		logger.Info("Redis not configured, status tracking disabled")
	}

	// Initialize simple in-memory rate limiter (open-core implementation)
	rateLimiter := middleware.NewSimpleRateLimiter()
	logger.Info("initialized in-memory rate limiter (open-core edition)")

	// Initialize simple tenant resolver (open-core implementation)
	// WARNING: This is demo-only and not production-secure
	tenantResolver := auth.NewSimpleTenantResolver()
	defer tenantResolver.Close()
	logger.Info("initialized simple tenant resolver (demo mode - not production secure)")

	// Initialize no-op usage tracker (open-core implementation)
	usageTracker := middleware.NewNoOpUsageTracker(logger)
	logger.Info("initialized no-op usage tracker (demo mode)")

	// Initialize Pub/Sub client and topic (optional for async routing)
	var pubsubTopic *pubsub.Topic

	if cfg.ProjectID != "" && cfg.PubSubTopic != "" {
		client, err := pubsub.NewClient(ctx, cfg.ProjectID)
		if err != nil {
			logger.Warn("Pub/Sub not available, async routing disabled",
				zap.Error(err))
		} else {
			defer client.Close()

			topic := client.Topic(cfg.PubSubTopic)
			exists, err := topic.Exists(ctx)
			if err != nil {
				logger.Warn("failed to check Pub/Sub topic", zap.Error(err))
			} else if !exists {
				logger.Info("creating Pub/Sub topic", zap.String("topic", cfg.PubSubTopic))
				topic, err = client.CreateTopic(ctx, cfg.PubSubTopic)
				if err != nil {
					logger.Warn("failed to create Pub/Sub topic", zap.Error(err))
				}
			}

			if err == nil {
				topic.EnableMessageOrdering = true
				pubsubTopic = topic
				logger.Info("Pub/Sub topic ready", zap.String("topic", cfg.PubSubTopic))
			}
		}
	} else {
		logger.Info("Pub/Sub not configured, async routing disabled")
	}

	// Construct base URL for status/stream URLs
	baseURL := cfg.PublicURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
		logger.Warn("PUBLIC_URL not set, using localhost", zap.String("baseURL", baseURL))
	}

	// Initialize route matcher
	var routeMatcher *routes.Matcher
	if pubsubTopic != nil && statusStore != nil {
		routeMatcher = routes.NewMatcher(pubsubTopic, statusStore, logger, baseURL)
	} else {
		logger.Info("async routing disabled (requires both Pub/Sub and Redis)")
	}

	// Load route configurations
	routeConfigs := config.LoadRoutesFromEnv()
	if len(routeConfigs) == 0 {
		logger.Info("no route configurations found")
		if routeMatcher == nil {
			logger.Warn("no routing configured (neither sync nor async)")
		}
	} else {
		logger.Info("loaded route configurations", zap.Int("count", len(routeConfigs)))
	}

	// Initialize sync proxy
	syncProxyMulti := routes.NewSyncProxyMulti(routeConfigs, logger)
	defer syncProxyMulti.Close()

	// Create HTTP router
	r := mux.NewRouter()

	// Health check endpoint
	healthChecker := health.NewChecker(policyStore, pubsubTopic, observabilityInit, logger)
	r.HandleFunc("/health", healthChecker.Handler()).Methods(http.MethodGet)

	// Readiness check
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if policyStore != nil && policyStore.IsReady() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ready","edition":"open-core"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready","edition":"open-core"}`))
		}
	}).Methods(http.MethodGet)

	// Status endpoint (if async routing enabled)
	if routeMatcher != nil {
		r.HandleFunc("/status/{request_id}", routeMatcher.HandleStatus).Methods(http.MethodGet)
	}

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	// Main routing handler with middleware chain
	// Middleware order:
	//   1. RequestID - Generate unique request ID
	//   2. TenantContext - Resolve tenant (demo mode: always default)
	//   3. RateLimit - Simple in-memory rate limiting
	//   4. PolicyVersionTag - Add policy version (always "default" in open-core)
	//   5. UsageTracker - Log usage events (no-op in open-core)
	//   6. Metrics - Record Prometheus metrics
	//   7. Logging - Log request details
	//   8. Tracing - Add distributed tracing (if OTEL configured)

	// Setup routing based on available components
	if routeMatcher != nil {
		// Async routing available - use sync proxy with fallback to async
		asyncHandler := middleware.Chain(
			http.HandlerFunc(routeMatcher.Handle),
			middleware.RequestID(logger),
			middleware.TenantContext(tenantResolver, logger),
			middleware.RateLimit(rateLimiter, logger),
			middleware.PolicyVersionTag(policyStore, logger),
			middleware.UsageTrackerMiddleware(usageTracker, logger),
			middleware.Metrics(),
			middleware.Logging(logger),
			middleware.Tracing(),
		)

		r.PathPrefix("/").Handler(
			middleware.Chain(
				syncProxyMulti.HandleWithFallback(asyncHandler),
				middleware.RequestID(logger),
				middleware.TenantContext(tenantResolver, logger),
				middleware.RateLimit(rateLimiter, logger),
				middleware.PolicyVersionTag(policyStore, logger),
				middleware.UsageTrackerMiddleware(usageTracker, logger),
				middleware.Metrics(),
				middleware.Logging(logger),
				middleware.Tracing(),
			),
		)
	} else {
		// Sync-only mode - use a 404 fallback
		notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":{"code":"not_found","message":"No route configured for this path"}}`))
		})

		r.PathPrefix("/").Handler(
			middleware.Chain(
				syncProxyMulti.HandleWithFallback(notFoundHandler),
				middleware.RequestID(logger),
				middleware.TenantContext(tenantResolver, logger),
				middleware.RateLimit(rateLimiter, logger),
				middleware.PolicyVersionTag(policyStore, logger),
				middleware.UsageTrackerMiddleware(usageTracker, logger),
				middleware.Metrics(),
				middleware.Logging(logger),
				middleware.Tracing(),
			),
		)
	}

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
		logger.Info("APX Router Open-Core Edition ready",
			zap.Int("port", cfg.Port),
			zap.String("environment", cfg.Environment),
			zap.String("edition", "open-core"),
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
