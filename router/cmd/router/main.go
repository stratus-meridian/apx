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
	apxratelimit "github.com/stratus-meridian/apx-private/control/pkg/ratelimit"
	"github.com/stratus-meridian/apx-private/control/tenant"
	"github.com/stratus-meridian/apx-private/control/usage"
	"github.com/stratus-meridian/apx/router/internal/auth"
	"github.com/stratus-meridian/apx/router/internal/config"
	"github.com/stratus-meridian/apx/router/internal/middleware"
	"github.com/stratus-meridian/apx/router/internal/policy"
	"github.com/stratus-meridian/apx/router/internal/routes"
	"github.com/stratus-meridian/apx/router/pkg/health"
	"github.com/stratus-meridian/apx/router/pkg/observability"
	"github.com/stratus-meridian/apx/router/pkg/status"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	// Use fmt.Println for early errors before logger is initialized
	fmt.Println("Starting APX Router...")

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Logger initialized successfully")

	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	logger.Info("Configuration loaded",
		zap.String("project_id", cfg.ProjectID),
		zap.String("region", cfg.Region))

	// Initialize observability (OTEL, metrics) - make non-fatal in dev
	shutdown, err := observability.Init(ctx, cfg, logger)
	observabilityInit := err == nil
	if err != nil {
		logger.Warn("failed to initialize observability (continuing)", zap.Error(err))
		shutdown = func() {} // no-op shutdown
	} else {
		logger.Info("Observability initialized successfully")
	}
	defer shutdown()

	// Initialize policy store (loads compiled artifacts from GCS/Firestore)
	// Make non-fatal if no policies exist yet
	policyStore, err := policy.NewStore(ctx, cfg, logger)
	if err != nil {
		logger.Warn("failed to initialize policy store (will use default policies)", zap.Error(err))
		// Continue without policy store - routes will use defaults
		policyStore = nil
	} else {
		logger.Info("Policy store initialized successfully")
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

	// Test Redis connection (non-blocking)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to connect to Redis, rate limiting will be degraded",
			zap.Error(err),
			zap.String("redis_addr", cfg.RedisAddr))
	} else {
		logger.Info("connected to Redis",
			zap.String("addr", cfg.RedisAddr))
	}

	// Initialize status store
	statusStore := status.NewRedisStore(redisClient, 24*time.Hour)

	// Initialize rate limiter with token bucket algorithm
	rlConfig := apxratelimit.DefaultConfig()
	rlConfig.RedisAddr = cfg.RedisAddr
	rlConfig.RedisPassword = cfg.RedisPassword
	rlConfig.RedisDB = cfg.RedisDB

	rateLimiter, err := apxratelimit.NewRedisLimiter(rlConfig)
	if err != nil {
		logger.Fatal("failed to initialize rate limiter", zap.Error(err))
	}
	defer rateLimiter.Close()

	// Initialize quota enforcer for monthly quota limits
	quotaEnforcer := apxratelimit.NewQuotaEnforcer(redisClient)

	// Initialize tenant repository (Firestore)
	tenantRepo, err := tenant.NewFirestoreRepository(ctx, cfg.ProjectID, tenant.DefaultRepositoryOptions())
	if err != nil {
		logger.Fatal("failed to initialize tenant repository", zap.Error(err))
	}
	defer tenantRepo.Close()

	// Initialize tenant resolver with Redis caching
	tenantResolver := auth.NewFirestoreTenantResolver(tenantRepo, redisClient, logger)
	defer tenantResolver.Close()

	// Initialize usage tracker for BigQuery analytics
	var usageTracker usage.UsageTracker
	usageConfig := usage.DefaultTrackerConfig(cfg.ProjectID)
	usageConfig.Dataset = "usage"
	usageConfig.Table = "events"

	tracker, err := usage.NewBigQueryUsageTracker(ctx, usageConfig, logger)
	if err != nil {
		logger.Warn("failed to initialize BigQuery usage tracker, using mock tracker",
			zap.Error(err))
		usageTracker = usage.NewMockUsageTracker(logger)
	} else {
		logger.Info("BigQuery usage tracker initialized successfully",
			zap.String("dataset", usageConfig.Dataset),
			zap.String("table", usageConfig.Table))
		usageTracker = tracker
		defer tracker.Close(ctx)
	}

	// Initialize Pub/Sub client and topic
	var pubsubTopic *pubsub.Topic

	// Create Pub/Sub client
	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		logger.Fatal("failed to create pubsub client", zap.Error(err))
	}
	defer client.Close()

	// Get or create topic
	topicName := cfg.PubSubTopic
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

	// Construct base URL for status/stream URLs
	baseURL := cfg.PublicURL
	if baseURL == "" {
		// Fallback to localhost if PUBLIC_URL not set
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
		logger.Warn("PUBLIC_URL not set, using localhost", zap.String("baseURL", baseURL))
	} else {
		logger.Info("using configured PUBLIC_URL", zap.String("baseURL", baseURL))
	}

	// Initialize route matcher with real topic (async mode)
	routeMatcher := routes.NewMatcher(pubsubTopic, statusStore, logger, baseURL)

	// Load route configurations (sync/async modes)
	routeConfigs := config.LoadRoutesFromEnv()
	if len(routeConfigs) == 0 {
		logger.Info("no route configurations found, using defaults (async-only mode)")
	} else {
		logger.Info("loaded route configurations",
			zap.Int("count", len(routeConfigs)))
		for _, rc := range routeConfigs {
			logger.Info("route registered",
				zap.String("path", rc.Path),
				zap.String("backend", rc.Backend),
				zap.String("mode", rc.Mode))
		}
	}

	// Initialize sync proxy for configured routes
	syncProxyMulti := routes.NewSyncProxyMulti(routeConfigs, logger)
	defer syncProxyMulti.Close()

	// Initialize dynamic config loader (polls control-API for gateway configs)
	controlAPIURL := os.Getenv("CONTROL_API_URL")
	tenantID := os.Getenv("TENANT_ID")

	var dynamicLoader *config.DynamicLoader
	if controlAPIURL != "" && tenantID != "" {
		logger.Info("initializing dynamic config loader",
			zap.String("control_api_url", controlAPIURL),
			zap.String("tenant_id", tenantID))

		dynamicLoader = config.NewDynamicLoader(config.DynamicLoaderConfig{
			ControlAPIURL:  controlAPIURL,
			TenantID:       tenantID,
			ReloadInterval: 60 * time.Second,
			Logger:         logger,
			OnChange: func(newRoutes []config.RouteConfig) error {
				// Reload sync proxy with new routes
				logger.Info("reloading sync proxy with new routes",
					zap.Int("route_count", len(newRoutes)))

				// Create new sync proxy with updated routes
				newProxy := routes.NewSyncProxyMulti(newRoutes, logger)

				// Replace the old proxy (graceful swap)
				// Note: We can't close the old proxy immediately as there might be
				// in-flight requests. In production, use a more sophisticated approach.
				oldProxy := syncProxyMulti
				syncProxyMulti = newProxy

				// Close old proxy after a grace period
				go func() {
					time.Sleep(30 * time.Second)
					oldProxy.Close()
				}()

				return nil
			},
		})

		// Start dynamic loader in background
		go func() {
			if err := dynamicLoader.Start(ctx); err != nil {
				logger.Error("dynamic config loader stopped", zap.Error(err))
			}
		}()
	} else {
		logger.Info("dynamic config loader not enabled (set CONTROL_API_URL and TENANT_ID to enable)")
	}

	// Create HTTP router
	r := mux.NewRouter()

	// Enable detailed per-middleware timing logs when requested.
	debugMiddleware := middleware.IsDebugEnabled()
	if debugMiddleware {
		logger.Info("ROUTER_DEBUG_MIDDLEWARE enabled - middleware step timing will be logged")
	}

	// Initialize health checker with component clients
	healthChecker := health.NewChecker(policyStore, pubsubTopic, observabilityInit, logger)

	// Health check endpoint - matches portal expectations
	r.HandleFunc("/health", healthChecker.Handler()).Methods(http.MethodGet)

	// Readiness check (verifies policy store is ready)
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if policyStore != nil && policyStore.IsReady() {
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
	// Supports both sync (direct proxy) and async (pub/sub) modes
	// Middleware order:
	//   1. RequestID - Generate unique request ID
	//   2. TenantContext - Resolve tenant from API key (security-critical)
	//   3. QuotaEnforcement - Check monthly quota limits (returns 402 if exceeded)
	//   4. RateLimit - Check per-minute rate limits (returns 429 if exceeded)
	//   5. PolicyVersionTag - Add policy version metadata
	//   6. UsageTracker - Track usage events to BigQuery (async, non-blocking)
	//   7. Metrics - Record metrics
	//   8. Logging - Log request details
	//   9. Tracing - Add distributed tracing
	asyncHandler := middleware.Chain(
		http.HandlerFunc(routeMatcher.Handle),
		middleware.WithStepLogging("RequestID", logger, middleware.RequestID(logger)),
		middleware.WithStepLogging("TenantContext", logger, middleware.TenantContext(tenantResolver, logger)), // Secure tenant resolution
		middleware.WithStepLogging("QuotaEnforcement", logger, middleware.QuotaEnforcement(quotaEnforcer, logger)), // Monthly quota enforcement
		middleware.WithStepLogging("RateLimit", logger, middleware.RateLimit(rateLimiter, logger)), // Per-minute rate limiting
		middleware.WithStepLogging("PolicyVersionTag", logger, middleware.PolicyVersionTag(policyStore, logger)),
		middleware.WithStepLogging("UsageTracker", logger, middleware.UsageTracker(usageTracker, logger)), // BigQuery usage tracking
		middleware.WithStepLogging("Metrics", logger, middleware.Metrics()),
		middleware.WithStepLogging("Logging", logger, middleware.Logging(logger)),
		middleware.WithStepLogging("Tracing", logger, middleware.Tracing()),
	)

	// Use sync proxy with fallback to async
	// If a route is configured as "sync", it will proxy directly
	// Otherwise, falls back to async (pub/sub) mode
	r.PathPrefix("/").Handler(
		middleware.Chain(
			syncProxyMulti.HandleWithFallback(asyncHandler),
			middleware.WithStepLogging("RequestID", logger, middleware.RequestID(logger)),
			middleware.WithStepLogging("TenantContext", logger, middleware.TenantContext(tenantResolver, logger)), // Secure tenant resolution
			middleware.WithStepLogging("QuotaEnforcement", logger, middleware.QuotaEnforcement(quotaEnforcer, logger)), // Monthly quota enforcement
			middleware.WithStepLogging("RateLimit", logger, middleware.RateLimit(rateLimiter, logger)), // Per-minute rate limiting
			middleware.WithStepLogging("PolicyVersionTag", logger, middleware.PolicyVersionTag(policyStore, logger)),
			middleware.WithStepLogging("UsageTracker", logger, middleware.UsageTracker(usageTracker, logger)), // BigQuery usage tracking
			middleware.WithStepLogging("Metrics", logger, middleware.Metrics()),
			middleware.WithStepLogging("Logging", logger, middleware.Logging(logger)),
			middleware.WithStepLogging("Tracing", logger, middleware.Tracing()),
		),
	)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,  // Increased for long-running requests
		WriteTimeout: 300 * time.Second, // Match Envoy timeout (5 minutes)
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
