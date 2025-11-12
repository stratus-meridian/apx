package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("apx-router")

// Tracing adds OpenTelemetry tracing with header attributes
func Tracing() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create span for the request
			ctx, span := tracer.Start(r.Context(), "http.request",
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			// Add all propagated headers as span attributes
			requestID := GetRequestID(ctx)
			tenantID := GetTenantID(ctx)
			tenantTier := GetTenantTier(ctx)
			policyVersion := GetPolicyVersion(ctx)
			region := GetRegion(ctx)

			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.path", r.URL.Path),
				attribute.String("http.host", r.Host),
				attribute.String("request.id", requestID),
				attribute.String("tenant.id", tenantID),
				attribute.String("tenant.tier", tenantTier),
				attribute.String("policy.version", policyVersion),
				attribute.String("deployment.region", region),
			)

			// Continue with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
