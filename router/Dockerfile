# Multi-stage Dockerfile for APX Router Service
# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o router ./cmd/router

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS and curl for health checks
RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/router .

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
