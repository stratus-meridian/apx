# Multi-stage Dockerfile for APX Router Service
# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy module files and local replacements before downloading deps
COPY router/go.mod router/go.sum ./router/
COPY .private/control/pkg/ratelimit ./.private/control/pkg/ratelimit
COPY .private/control/tenant ./.private/control/tenant
COPY .private/control/usage ./.private/control/usage

WORKDIR /build/router
RUN go mod download

# Copy router source
COPY router/ .

# Tidy dependencies and build
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o router ./cmd/router

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS and curl for health checks
RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/router/router .

# Add non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Health check
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8081/health || exit 1

# Expose port
EXPOSE 8081

# Run the binary
ENTRYPOINT ["./router"]
