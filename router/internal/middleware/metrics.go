package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/apx/router/internal/metrics"
)

// Metrics records HTTP request metrics using Prometheus
func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &metricsResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start).Seconds()

			// Extract context values
			tenantTier := GetTenantTier(r.Context())
			if tenantTier == "" {
				tenantTier = "unknown"
			}

			// Record metrics
			metrics.RequestsTotal.WithLabelValues(
				r.Method,
				r.URL.Path,
				strconv.Itoa(wrapped.statusCode),
				tenantTier,
			).Inc()

			metrics.RequestDuration.WithLabelValues(
				r.Method,
				r.URL.Path,
				tenantTier,
			).Observe(duration)
		})
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
