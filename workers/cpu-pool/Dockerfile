FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod ./

# Copy source (needed for go mod tidy)
COPY *.go ./

# Download dependencies and tidy
RUN go mod tidy && go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o worker .

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app
COPY --from=builder /build/worker .

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s \
  CMD exit 0

EXPOSE 8084

ENTRYPOINT ["./worker"]
