package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal tracks the total number of HTTP requests
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apx_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status", "tenant_tier"},
	)

	// RequestDuration tracks the duration of HTTP requests
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apx_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "tenant_tier"},
	)

	// PubSubPublished tracks messages published to Pub/Sub
	PubSubPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apx_pubsub_published_total",
			Help: "Total messages published to Pub/Sub",
		},
		[]string{"tenant_id", "success"},
	)

	// PolicyEvaluations tracks policy evaluation results
	PolicyEvaluations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apx_policy_evaluations_total",
			Help: "Total number of policy evaluations",
		},
		[]string{"tenant_id", "policy_version", "result"},
	)

	// CacheHits tracks cache hit/miss statistics
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apx_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type", "hit"},
	)

	// RateLimitChecks tracks rate limit checks
	RateLimitChecks = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apx_ratelimit_checks_total",
			Help: "Total number of rate limit checks",
		},
		[]string{"tenant_tier", "allowed"},
	)
)
