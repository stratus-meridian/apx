package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/stratus-meridian/apx-router-open-core/internal/config"
	"github.com/stratus-meridian/apx-router-open-core/internal/middleware"
	"github.com/stratus-meridian/apx-router-open-core/pkg/proxy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// SyncProxy handles synchronous HTTP proxying to backends
type SyncProxy struct {
	client    *proxy.Client
	logger    *zap.Logger
	backend   string // Backend URL (e.g., https://mocktarget.apigee.net)
	pathStrip string // Path prefix to strip before proxying
}

// NewSyncProxy creates a new synchronous proxy handler
func NewSyncProxy(backend string, pathStrip string, logger *zap.Logger) *SyncProxy {
	// Create proxy client with default config
	cfg := proxy.DefaultConfig()
	client := proxy.NewClient(cfg, logger)

	return &SyncProxy{
		client:    client,
		logger:    logger,
		backend:   backend,
		pathStrip: pathStrip,
	}
}

// Handle processes the request synchronously
func (sp *SyncProxy) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer("apx-router")

	// Start span for backend request
	ctx, span := tracer.Start(ctx, "proxy.backend_request")
	defer span.End()

	// Get request metadata from context
	requestID := middleware.GetRequestID(ctx)
	tenantID := middleware.GetTenantID(ctx)

	span.SetAttributes(
		attribute.String("request.id", requestID),
		attribute.String("tenant.id", tenantID),
		attribute.String("backend.url", sp.backend),
		attribute.String("http.method", r.Method),
		attribute.String("http.path", r.URL.Path),
	)

	sp.logger.Info("proxying request synchronously",
		zap.String("request_id", requestID),
		zap.String("tenant_id", tenantID),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("backend", sp.backend),
	)

	// Add timeout to context (30 seconds default)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Update request context
	r = r.WithContext(ctx)

	// Proxy the request (with path stripping if configured)
	startTime := time.Now()
	resp, err := sp.client.ProxyRequestWithPathStrip(ctx, r, sp.backend, sp.pathStrip)
	duration := time.Since(startTime)

	if err != nil {
		span.RecordError(err)
		sp.logger.Error("backend request failed",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.Duration("duration", duration),
		)

		// Return error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "backend_error",
			"message":    "Failed to reach backend service",
			"request_id": requestID,
		})
		return
	}
	defer resp.Body.Close()

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.Int64("http.response_size", resp.ContentLength),
		attribute.Int64("backend.duration_ms", duration.Milliseconds()),
	)

	sp.logger.Info("backend response received",
		zap.String("request_id", requestID),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration),
	)

	// Copy response from backend to client
	err = proxy.CopyResponse(w, resp)
	if err != nil {
		sp.logger.Error("failed to copy response",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		// Response already started, can't change status
		return
	}

	sp.logger.Debug("request completed successfully",
		zap.String("request_id", requestID),
		zap.Duration("total_duration", duration),
	)
}

// Close cleans up resources
func (sp *SyncProxy) Close() error {
	return sp.client.Close()
}

// SyncProxyMulti handles multiple backend routes
type SyncProxyMulti struct {
	routes map[string]*SyncProxy // path -> proxy
	logger *zap.Logger
}

// NewSyncProxyMulti creates a multi-route proxy
func NewSyncProxyMulti(routes []config.RouteConfig, logger *zap.Logger) *SyncProxyMulti {
	proxies := make(map[string]*SyncProxy)

	for _, route := range routes {
		if route.Mode == "sync" {
			proxy := NewSyncProxy(route.Backend, route.PathStrip, logger)
			proxies[route.Path] = proxy
			logger.Info("registered sync route",
				zap.String("path", route.Path),
				zap.String("backend", route.Backend),
				zap.String("path_strip", route.PathStrip),
			)
		}
	}

	return &SyncProxyMulti{
		routes: proxies,
		logger: logger,
	}
}

// GetProxy returns the proxy for a given path
func (spm *SyncProxyMulti) GetProxy(path string) (*SyncProxy, bool) {
	proxy, ok := spm.routes[path]
	return proxy, ok
}

// Close cleans up all proxies
func (spm *SyncProxyMulti) Close() error {
	var lastErr error
	for _, proxy := range spm.routes {
		if err := proxy.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// HandleWithFallback tries sync proxy first, falls back to async
func (spm *SyncProxyMulti) HandleWithFallback(asyncHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to find a sync proxy for this path
		// Look for exact match or prefix match
		for path, proxy := range spm.routes {
			if r.URL.Path == path || matchesPrefix(r.URL.Path, path) {
				// Handle synchronously
				proxy.Handle(w, r)
				return
			}
		}

		// Fall back to async handler
		asyncHandler.ServeHTTP(w, r)
	}
}

// matchesPrefix checks if reqPath matches routePath pattern
// Supports wildcards like /mock/** matching /mock/anything
func matchesPrefix(reqPath, routePath string) bool {
	if len(routePath) > 2 && routePath[len(routePath)-2:] == "**" {
		prefix := routePath[:len(routePath)-2]
		return len(reqPath) >= len(prefix) && reqPath[:len(prefix)] == prefix
	}
	return reqPath == routePath
}
