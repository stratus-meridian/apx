package middleware

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/stratus-meridian/apx-private/control/usage"
	"go.uber.org/zap"
)

var defaultUsageRegion = func() string {
	if region := os.Getenv("GCP_REGION"); region != "" {
		return region
	}
	if region := os.Getenv("REGION"); region != "" {
		return region
	}
	return "us-central1"
}()

// UsageTracker creates middleware that records request metadata via the shared usage tracker.
func UsageTracker(tracker usage.UsageTracker, logger *zap.Logger) Middleware {
	if tracker == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := newResponseRecorder(w)
			start := time.Now()

			next.ServeHTTP(recorder, r)

			tenantCtx, ok := GetTenant(r.Context())
			if !ok || tenantCtx == nil {
				logger.Warn("usage tracking skipped: tenant context missing",
					zap.String("path", r.URL.Path))
				return
			}

			event := usage.UsageEvent{
				Timestamp:     time.Now(),
				TenantID:      tenantCtx.ResourceID,
				OrgID:         tenantCtx.Organization.ID,
				ProductID:     tenantCtx.Product.ID,
				EnvironmentID: tenantCtx.Environment.ID,
				Tier:          string(tenantCtx.Organization.Tier),
				RequestCount:  1,
				Endpoint:      r.URL.Path,
				Method:        r.Method,
				StatusCode:    recorder.statusCode,
				ResponseTime:  time.Since(start).Milliseconds(),
				Cached:        false,
				Region:        headerOrDefault(r, "X-APX-Region", defaultUsageRegion),
				Version:       headerOrDefault(r, "X-APX-Version", "v1"),
			}

			go func(ev usage.UsageEvent) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := tracker.TrackRequest(ctx, ev); err != nil {
					logger.Error("failed to track usage event",
						zap.Error(err),
						zap.String("tenant_id", ev.TenantID),
						zap.String("path", ev.Endpoint))
				}
			}(event)
		})
	}
}

func headerOrDefault(r *http.Request, key, fallback string) string {
	if value := r.Header.Get(key); value != "" {
		return value
	}
	return fallback
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseRecorder) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseRecorder) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
