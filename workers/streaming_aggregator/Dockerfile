FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum from router (for shared dependencies)
COPY router/go.mod router/go.sum ./router/
COPY workers/streaming_aggregator/go.mod workers/streaming_aggregator/go.sum* ./workers/streaming_aggregator/

# Copy router pkg packages (needed for status store)
COPY router/pkg ./router/pkg/

# Copy streaming aggregator source
COPY workers/streaming_aggregator/*.go ./workers/streaming_aggregator/

# Build from streaming aggregator directory
WORKDIR /app/workers/streaming_aggregator
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /streaming-aggregator .

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /streaming-aggregator .

# Add non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 8083

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8083/health || exit 1

ENTRYPOINT ["./streaming-aggregator"]
