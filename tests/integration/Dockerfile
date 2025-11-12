FROM golang:1.22-alpine

WORKDIR /test

COPY go.mod go.sum* ./
RUN go mod download

COPY test_pubsub.go ./

RUN go build -o test_pubsub test_pubsub.go

CMD ["./test_pubsub"]
