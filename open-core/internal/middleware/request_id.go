package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const RequestIDKey contextKey = "request_id"

// RequestID ensures every request has a unique ID
func RequestID(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request ID already exists (from Envoy)
			requestID := r.Header.Get("X-Request-ID")

			// Generate new ID if not present
			if requestID == "" {
				requestID = uuid.New().String()
				logger.Debug("generated request ID", zap.String("request_id", requestID))
			}

			// Add to context
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

			// Add to response headers
			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
