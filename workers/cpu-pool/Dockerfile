FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git

# Copy the entire workspace (needed for replace directive)
COPY . .

# Set working directory to worker
WORKDIR /build/workers/cpu-pool

# Download dependencies
RUN go mod download
RUN go mod tidy

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o worker .

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app
COPY --from=builder /build/workers/cpu-pool/worker .

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s \
  CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080

ENTRYPOINT ["./worker"]
