FROM golang:1.23-alpine AS builder

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

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /streaming-aggregator .

EXPOSE 8083

CMD ["./streaming-aggregator"]
