// Package observability provides OpenTelemetry integration
package observability

import (
	"context"
	"fmt"
	"os"

	"github.com/apx/router/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.uber.org/zap"
)

// Init initializes OpenTelemetry tracing, metrics, and logging
func Init(ctx context.Context, cfg *config.Config, logger *zap.Logger) (func(), error) {
	// Get OTEL endpoint from environment
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		logger.Warn("OTEL_EXPORTER_OTLP_ENDPOINT not set, tracing disabled")
		return func() {}, nil
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("apx-router"),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure OTLP trace exporter with insecure connection
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Set global propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("observability initialized",
		zap.String("otel_endpoint", otelEndpoint),
		zap.String("service", "apx-router"),
		zap.String("environment", cfg.Environment),
	)

	// Return cleanup function
	cleanup := func() {
		logger.Info("shutting down observability")
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error("failed to shutdown trace provider", zap.Error(err))
		}
		logger.Info("observability shutdown complete")
	}

	return cleanup, nil
}
