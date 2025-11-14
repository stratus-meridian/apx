package middleware

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// UsageTracker defines the interface for tracking API usage.
//
// Open-core implementation: NoOpUsageTracker (logs to stdout)
// Commercial implementation: BigQueryUsageTracker (real-time analytics)
type UsageTracker interface {
	TrackRequest(ctx context.Context, event UsageEvent) error
	Close(ctx context.Context) error
}

// UsageEvent represents a single API usage event.
type UsageEvent struct {
	Timestamp     time.Time
	TenantID      string
	Tier          string
	RequestCount  int
	Endpoint      string
	Method        string
	StatusCode    int
	ResponseTime  int64 // milliseconds
	Cached        bool
	Region        string
	Version       string
}

// NoOpUsageTracker is a demonstration implementation that logs usage to stdout.
// The commercial version sends events to BigQuery for analytics and billing.
type NoOpUsageTracker struct {
	logger *zap.Logger
}

// NewNoOpUsageTracker creates a new no-op usage tracker.
func NewNoOpUsageTracker(logger *zap.Logger) *NoOpUsageTracker {
	return &NoOpUsageTracker{logger: logger}
}

// TrackRequest logs the usage event (demo mode - no persistence).
func (t *NoOpUsageTracker) TrackRequest(ctx context.Context, event UsageEvent) error {
	t.logger.Debug("usage tracked (demo mode)",
		zap.String("tenant_id", event.TenantID),
		zap.String("endpoint", event.Endpoint),
		zap.String("method", event.Method),
		zap.Int("status_code", event.StatusCode),
		zap.Int64("response_time_ms", event.ResponseTime))
	return nil
}

// Close releases resources (none in this implementation).
func (t *NoOpUsageTracker) Close(ctx context.Context) error {
	return nil
}

// UsageTrackerMiddleware creates middleware that tracks API usage.
// This middleware should be placed after TenantContext middleware.
func UsageTrackerMiddleware(tracker UsageTracker, logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response wrapper to capture status code
			wrapper := &responseWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(wrapper, r)

			// Track usage event after request completes (async, non-blocking)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Extract tenant from context
				tenant, ok := GetTenant(r.Context())
				if !ok || tenant == nil {
					logger.Warn("usage tracking skipped: tenant not found",
						zap.String("path", r.URL.Path))
					return
				}

				// Create usage event
				event := UsageEvent{
					Timestamp:    time.Now(),
					TenantID:     tenant.ID,
					Tier:         tenant.Tier,
					RequestCount: 1,
					Endpoint:     r.URL.Path,
					Method:       r.Method,
					StatusCode:   wrapper.statusCode,
					ResponseTime: time.Since(start).Milliseconds(),
					Cached:       false,
					Region:       "us-central1", // Default region
					Version:      "v1",
				}

				// Track the event
				if err := tracker.TrackRequest(ctx, event); err != nil {
					logger.Error("failed to track usage event",
						zap.Error(err),
						zap.String("tenant_id", tenant.ID))
				}
			}()
		})
	}
}

// responseWrapper wraps http.ResponseWriter to capture status code.
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWrapper) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
