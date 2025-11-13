module github.com/apx/streaming-aggregator

go 1.22

require (
	github.com/apx/router v0.0.0
	github.com/gorilla/mux v1.8.1
	github.com/redis/go-redis/v9 v9.16.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	go.uber.org/multierr v1.11.0 // indirect
)

replace github.com/apx/router => ../../router
