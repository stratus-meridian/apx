-- APX BigQuery Analytics Schema
-- Optimized for cost-efficient querying and storage
--
-- Design Principles:
-- 1. Partition by DAY to minimize scanned data
-- 2. Cluster by high-cardinality query dimensions (tenant_tier, route_pattern)
-- 3. Use appropriate data types to minimize storage
-- 4. Separate raw data (30 days) from aggregates (2 years)
-- 5. Pre-compute expensive aggregations

-- ============================================================================
-- RAW REQUESTS TABLE
-- ============================================================================
-- Stores individual request records with 30-day retention
-- Partitioned by day, clustered by tenant_tier and route_pattern

CREATE TABLE IF NOT EXISTS `PROJECT_ID.analytics.requests` (
  -- Request Identifiers
  request_id STRING NOT NULL OPTIONS(description="Unique request identifier (UUID)"),
  tenant_id STRING NOT NULL OPTIONS(description="Tenant identifier"),
  tenant_tier STRING NOT NULL OPTIONS(description="Tenant tier (free, pro, enterprise)"),

  -- Timing
  timestamp TIMESTAMP NOT NULL OPTIONS(description="Request start time (UTC)"),
  duration_ms FLOAT64 OPTIONS(description="Total request duration in milliseconds"),

  -- Request Details
  method STRING OPTIONS(description="HTTP method (GET, POST, etc)"),
  path STRING OPTIONS(description="Request path"),
  route_pattern STRING NOT NULL OPTIONS(description="Matched route pattern for aggregation"),

  -- Response
  status_code INT64 OPTIONS(description="HTTP status code"),
  response_bytes INT64 OPTIONS(description="Response body size in bytes"),

  -- Policy Execution
  policy_id STRING OPTIONS(description="Policy bundle ID"),
  policy_version STRING OPTIONS(description="Policy version (semver)"),
  policy_duration_ms FLOAT64 OPTIONS(description="Time spent in policy evaluation"),

  -- Source
  source_ip STRING OPTIONS(description="Client IP address (anonymized)"),
  user_agent STRING OPTIONS(description="Client user agent"),
  source_region STRING OPTIONS(description="Client geographic region"),

  -- Infrastructure
  edge_region STRING OPTIONS(description="Edge gateway region (us-central1, etc)"),
  router_instance STRING OPTIONS(description="Router instance ID"),
  worker_pool STRING OPTIONS(description="Worker pool that processed request"),
  worker_instance STRING OPTIONS(description="Worker instance ID"),

  -- Performance Breakdown
  edge_latency_ms FLOAT64 OPTIONS(description="Time spent in edge gateway"),
  router_latency_ms FLOAT64 OPTIONS(description="Time spent in router"),
  queue_wait_ms FLOAT64 OPTIONS(description="Time spent waiting in Pub/Sub queue"),
  worker_latency_ms FLOAT64 OPTIONS(description="Time spent in worker"),

  -- Errors
  error_type STRING OPTIONS(description="Error classification (auth, rate_limit, policy, upstream)"),
  error_message STRING OPTIONS(description="Error message (PII-redacted)"),

  -- Flags
  is_error BOOL OPTIONS(description="Whether request resulted in error (status >= 400)"),
  is_sampled BOOL NOT NULL DEFAULT TRUE OPTIONS(description="Whether this log was sampled (false = 100% logged)"),
  is_replay BOOL DEFAULT FALSE OPTIONS(description="Whether this is a replayed request"),

  -- Metadata
  trace_id STRING OPTIONS(description="Cloud Trace ID for distributed tracing"),
  span_id STRING OPTIONS(description="Trace span ID"),
  labels ARRAY<STRUCT<key STRING, value STRING>> OPTIONS(description="Custom labels for filtering")
)
PARTITION BY DATE(timestamp)
CLUSTER BY tenant_tier, route_pattern, is_error
OPTIONS(
  description="Raw request logs with 30-day retention",
  partition_expiration_days=30,
  require_partition_filter=true,
  labels=[("env", "production"), ("cost_center", "observability")]
);

-- ============================================================================
-- HOURLY AGGREGATES TABLE
-- ============================================================================
-- Pre-computed hourly metrics for fast dashboard queries
-- 2-year retention for trend analysis

CREATE TABLE IF NOT EXISTS `PROJECT_ID.analytics.requests_hourly_agg` (
  -- Time Window
  hour TIMESTAMP NOT NULL OPTIONS(description="Hour bucket (truncated to hour)"),

  -- Dimensions
  tenant_tier STRING NOT NULL OPTIONS(description="Tenant tier"),
  route_pattern STRING NOT NULL OPTIONS(description="Route pattern"),
  edge_region STRING OPTIONS(description="Edge region"),
  status_code_bucket STRING OPTIONS(description="Status code range (2xx, 4xx, 5xx)"),

  -- Request Counts
  request_count INT64 NOT NULL OPTIONS(description="Total requests in this hour"),
  error_count INT64 OPTIONS(description="Requests with status >= 400"),
  success_count INT64 OPTIONS(description="Requests with status < 400"),

  -- Latency Percentiles (milliseconds)
  duration_p50 FLOAT64 OPTIONS(description="Median duration"),
  duration_p95 FLOAT64 OPTIONS(description="95th percentile duration"),
  duration_p99 FLOAT64 OPTIONS(description="99th percentile duration"),
  duration_max FLOAT64 OPTIONS(description="Maximum duration"),
  duration_avg FLOAT64 OPTIONS(description="Average duration"),

  -- Component Latency Averages
  edge_latency_avg FLOAT64 OPTIONS(description="Average edge latency"),
  router_latency_avg FLOAT64 OPTIONS(description="Average router latency"),
  queue_wait_avg FLOAT64 OPTIONS(description="Average queue wait time"),
  worker_latency_avg FLOAT64 OPTIONS(description="Average worker latency"),
  policy_duration_avg FLOAT64 OPTIONS(description="Average policy evaluation time"),

  -- Data Transfer
  total_bytes INT64 OPTIONS(description="Total response bytes sent"),
  avg_bytes FLOAT64 OPTIONS(description="Average response size"),

  -- Error Breakdown
  auth_errors INT64 DEFAULT 0 OPTIONS(description="Authentication errors"),
  rate_limit_errors INT64 DEFAULT 0 OPTIONS(description="Rate limit errors"),
  policy_errors INT64 DEFAULT 0 OPTIONS(description="Policy evaluation errors"),
  upstream_errors INT64 DEFAULT 0 OPTIONS(description="Upstream API errors"),

  -- Metadata
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP() OPTIONS(description="Last update time")
)
PARTITION BY DATE(hour)
CLUSTER BY tenant_tier, route_pattern, edge_region
OPTIONS(
  description="Hourly aggregated metrics with 2-year retention",
  partition_expiration_days=730,
  require_partition_filter=true,
  labels=[("env", "production"), ("cost_center", "observability")]
);

-- ============================================================================
-- DAILY AGGREGATES TABLE
-- ============================================================================
-- Daily rollups for long-term trend analysis and reporting
-- 7-year retention for compliance and historical analysis

CREATE TABLE IF NOT EXISTS `PROJECT_ID.analytics.requests_daily_agg` (
  -- Time Window
  day DATE NOT NULL OPTIONS(description="Day (UTC)"),

  -- Dimensions
  tenant_tier STRING NOT NULL OPTIONS(description="Tenant tier"),
  route_pattern STRING OPTIONS(description="Route pattern"),
  edge_region STRING OPTIONS(description="Edge region"),

  -- Request Counts
  request_count INT64 NOT NULL OPTIONS(description="Total requests this day"),
  error_count INT64 OPTIONS(description="Total errors"),
  success_count INT64 OPTIONS(description="Total successes"),
  unique_tenants INT64 OPTIONS(description="Unique tenant count"),
  unique_routes INT64 OPTIONS(description="Unique route patterns used"),

  -- Latency Statistics
  duration_p50 FLOAT64 OPTIONS(description="Median duration"),
  duration_p95 FLOAT64 OPTIONS(description="95th percentile duration"),
  duration_p99 FLOAT64 OPTIONS(description="99th percentile duration"),
  duration_avg FLOAT64 OPTIONS(description="Average duration"),

  -- SLO Compliance
  slo_availability FLOAT64 OPTIONS(description="Availability (success_count / request_count)"),
  slo_p99_under_50ms FLOAT64 OPTIONS(description="% of requests with p99 < 50ms"),

  -- Cost Metrics
  estimated_cost_usd FLOAT64 OPTIONS(description="Estimated infrastructure cost for this day"),

  -- Metadata
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()
)
PARTITION BY day
CLUSTER BY tenant_tier, edge_region
OPTIONS(
  description="Daily aggregated metrics with 7-year retention",
  partition_expiration_days=2555,
  require_partition_filter=true,
  labels=[("env", "production"), ("cost_center", "observability")]
);

-- ============================================================================
-- TENANT SUMMARY TABLE
-- ============================================================================
-- Per-tenant summary for billing, usage tracking, and tier migrations

CREATE TABLE IF NOT EXISTS `PROJECT_ID.analytics.tenant_summary` (
  -- Tenant Info
  tenant_id STRING NOT NULL OPTIONS(description="Tenant identifier"),
  tenant_tier STRING NOT NULL OPTIONS(description="Current tier"),

  -- Time Window
  period_start DATE NOT NULL OPTIONS(description="Summary period start"),
  period_end DATE NOT NULL OPTIONS(description="Summary period end"),

  -- Usage Metrics
  total_requests INT64 OPTIONS(description="Total requests in period"),
  total_errors INT64 OPTIONS(description="Total errors"),
  total_bytes INT64 OPTIONS(description="Total data transferred"),

  -- Performance
  avg_duration_ms FLOAT64 OPTIONS(description="Average request duration"),
  p95_duration_ms FLOAT64 OPTIONS(description="95th percentile duration"),

  -- Cost
  estimated_cost_usd FLOAT64 OPTIONS(description="Estimated infrastructure cost"),

  -- Top Routes
  top_routes ARRAY<STRUCT<route STRING, count INT64>> OPTIONS(description="Most used routes"),

  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()
)
PARTITION BY period_start
CLUSTER BY tenant_tier, tenant_id
OPTIONS(
  description="Per-tenant usage summaries for billing and analytics",
  partition_expiration_days=730,
  require_partition_filter=true,
  labels=[("env", "production"), ("cost_center", "billing")]
);

-- ============================================================================
-- ERROR SNAPSHOTS METADATA TABLE
-- ============================================================================
-- Tracks error snapshots stored in GCS for debugging and replay

CREATE TABLE IF NOT EXISTS `PROJECT_ID.analytics.error_snapshots` (
  -- Error Info
  snapshot_id STRING NOT NULL OPTIONS(description="Unique snapshot ID"),
  request_id STRING NOT NULL OPTIONS(description="Original request ID"),
  tenant_id STRING NOT NULL OPTIONS(description="Tenant ID"),

  -- Timing
  occurred_at TIMESTAMP NOT NULL OPTIONS(description="When error occurred"),
  captured_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP() OPTIONS(description="When snapshot was captured"),

  -- Error Details
  error_type STRING OPTIONS(description="Error classification"),
  error_message STRING OPTIONS(description="Error message"),
  status_code INT64 OPTIONS(description="HTTP status code"),

  -- Context
  route_pattern STRING OPTIONS(description="Route pattern"),
  policy_version STRING OPTIONS(description="Policy version"),
  worker_pool STRING OPTIONS(description="Worker pool"),
  worker_instance STRING OPTIONS(description="Worker instance ID"),

  -- Storage
  gcs_path STRING NOT NULL OPTIONS(description="GCS path to snapshot JSON"),
  snapshot_size_bytes INT64 OPTIONS(description="Snapshot file size"),

  -- Replay Status
  replayed BOOL DEFAULT FALSE OPTIONS(description="Whether snapshot has been replayed"),
  replay_succeeded BOOL OPTIONS(description="Whether replay succeeded"),
  replayed_at TIMESTAMP OPTIONS(description="When snapshot was replayed"),

  -- Metadata
  labels ARRAY<STRUCT<key STRING, value STRING>> OPTIONS(description="Custom labels")
)
PARTITION BY DATE(occurred_at)
CLUSTER BY tenant_id, error_type
OPTIONS(
  description="Metadata for error snapshots stored in GCS",
  partition_expiration_days=90,
  require_partition_filter=true,
  labels=[("env", "production"), ("cost_center", "observability")]
);

-- ============================================================================
-- MATERIALIZED VIEWS (for common queries)
-- ============================================================================

-- Real-time error rate by tenant tier (last 24 hours)
CREATE MATERIALIZED VIEW IF NOT EXISTS `PROJECT_ID.analytics.current_error_rate_by_tier`
PARTITION BY DATE(hour)
CLUSTER BY tenant_tier
OPTIONS(
  enable_refresh=true,
  refresh_interval_minutes=15,
  description="Error rate by tenant tier (refreshed every 15 min)"
)
AS
SELECT
  hour,
  tenant_tier,
  SUM(request_count) as total_requests,
  SUM(error_count) as total_errors,
  SAFE_DIVIDE(SUM(error_count), SUM(request_count)) * 100 as error_rate_percentage,
  AVG(duration_p99) as avg_p99_latency
FROM `PROJECT_ID.analytics.requests_hourly_agg`
WHERE hour >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
GROUP BY hour, tenant_tier;

-- Top slow routes (last 7 days)
CREATE MATERIALIZED VIEW IF NOT EXISTS `PROJECT_ID.analytics.slow_routes`
PARTITION BY DATE(day)
CLUSTER BY route_pattern
OPTIONS(
  enable_refresh=true,
  refresh_interval_minutes=60,
  description="Slowest routes by p99 latency (last 7 days)"
)
AS
SELECT
  day,
  route_pattern,
  tenant_tier,
  request_count,
  duration_p99,
  duration_p95,
  RANK() OVER (PARTITION BY day ORDER BY duration_p99 DESC) as slowness_rank
FROM `PROJECT_ID.analytics.requests_daily_agg`
WHERE day >= DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)
  AND request_count > 100  -- Only routes with significant traffic
ORDER BY day DESC, duration_p99 DESC;

-- ============================================================================
-- EXAMPLE QUERIES
-- ============================================================================

-- Query 1: Error rate by tenant tier (last 24 hours)
/*
SELECT
  tenant_tier,
  COUNT(*) as total_requests,
  COUNTIF(is_error) as errors,
  SAFE_DIVIDE(COUNTIF(is_error), COUNT(*)) * 100 as error_rate_pct,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(99)] as p99_latency
FROM `PROJECT_ID.analytics.requests`
WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
  AND DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY)  -- Partition filter
GROUP BY tenant_tier
ORDER BY error_rate_pct DESC;
*/

-- Query 2: Top 10 slowest routes (last 7 days)
/*
SELECT
  route_pattern,
  COUNT(*) as request_count,
  AVG(duration_ms) as avg_duration,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(95)] as p95,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(99)] as p99,
  MAX(duration_ms) as max_duration
FROM `PROJECT_ID.analytics.requests`
WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
  AND DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)  -- Partition filter
GROUP BY route_pattern
ORDER BY p99 DESC
LIMIT 10;
*/

-- Query 3: Cost per million requests by tenant tier (last 30 days)
/*
SELECT
  tenant_tier,
  SUM(request_count) as total_requests,
  SUM(estimated_cost_usd) as total_cost,
  SAFE_DIVIDE(SUM(estimated_cost_usd), SUM(request_count) / 1000000) as cost_per_million
FROM `PROJECT_ID.analytics.requests_daily_agg`
WHERE day >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
GROUP BY tenant_tier
ORDER BY cost_per_million DESC;
*/

-- Query 4: Error trends over time (last 30 days)
/*
SELECT
  day,
  SUM(request_count) as requests,
  SUM(error_count) as errors,
  SAFE_DIVIDE(SUM(error_count), SUM(request_count)) * 100 as error_rate,
  AVG(duration_p99) as p99_latency
FROM `PROJECT_ID.analytics.requests_daily_agg`
WHERE day >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
GROUP BY day
ORDER BY day DESC;
*/

-- ============================================================================
-- SCHEDULED QUERIES (to be created via Cloud Scheduler + BigQuery)
-- ============================================================================

-- Hourly Aggregation Job
-- Runs every hour at :05 to aggregate previous hour's data
/*
INSERT INTO `PROJECT_ID.analytics.requests_hourly_agg` (
  hour,
  tenant_tier,
  route_pattern,
  edge_region,
  status_code_bucket,
  request_count,
  error_count,
  success_count,
  duration_p50,
  duration_p95,
  duration_p99,
  duration_max,
  duration_avg,
  edge_latency_avg,
  router_latency_avg,
  queue_wait_avg,
  worker_latency_avg,
  policy_duration_avg,
  total_bytes,
  avg_bytes,
  auth_errors,
  rate_limit_errors,
  policy_errors,
  upstream_errors
)
SELECT
  TIMESTAMP_TRUNC(timestamp, HOUR) as hour,
  tenant_tier,
  route_pattern,
  edge_region,
  CASE
    WHEN status_code < 300 THEN '2xx'
    WHEN status_code < 400 THEN '3xx'
    WHEN status_code < 500 THEN '4xx'
    ELSE '5xx'
  END as status_code_bucket,
  COUNT(*) as request_count,
  COUNTIF(is_error) as error_count,
  COUNTIF(NOT is_error) as success_count,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(50)] as duration_p50,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(95)] as duration_p95,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(99)] as duration_p99,
  MAX(duration_ms) as duration_max,
  AVG(duration_ms) as duration_avg,
  AVG(edge_latency_ms) as edge_latency_avg,
  AVG(router_latency_ms) as router_latency_avg,
  AVG(queue_wait_ms) as queue_wait_avg,
  AVG(worker_latency_ms) as worker_latency_avg,
  AVG(policy_duration_ms) as policy_duration_avg,
  SUM(response_bytes) as total_bytes,
  AVG(response_bytes) as avg_bytes,
  COUNTIF(error_type = 'auth') as auth_errors,
  COUNTIF(error_type = 'rate_limit') as rate_limit_errors,
  COUNTIF(error_type = 'policy') as policy_errors,
  COUNTIF(error_type = 'upstream') as upstream_errors
FROM `PROJECT_ID.analytics.requests`
WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 2 HOUR)
  AND timestamp < TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
  AND DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 2 DAY)
GROUP BY hour, tenant_tier, route_pattern, edge_region, status_code_bucket;
*/

-- Daily Aggregation Job
-- Runs daily at 02:00 UTC to aggregate previous day's data
/*
INSERT INTO `PROJECT_ID.analytics.requests_daily_agg` (
  day,
  tenant_tier,
  route_pattern,
  edge_region,
  request_count,
  error_count,
  success_count,
  unique_tenants,
  unique_routes,
  duration_p50,
  duration_p95,
  duration_p99,
  duration_avg,
  slo_availability,
  slo_p99_under_50ms
)
SELECT
  DATE(timestamp) as day,
  tenant_tier,
  route_pattern,
  edge_region,
  COUNT(*) as request_count,
  COUNTIF(is_error) as error_count,
  COUNTIF(NOT is_error) as success_count,
  COUNT(DISTINCT tenant_id) as unique_tenants,
  COUNT(DISTINCT route_pattern) as unique_routes,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(50)] as duration_p50,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(95)] as duration_p95,
  APPROX_QUANTILES(duration_ms, 100)[OFFSET(99)] as duration_p99,
  AVG(duration_ms) as duration_avg,
  SAFE_DIVIDE(COUNTIF(NOT is_error), COUNT(*)) as slo_availability,
  SAFE_DIVIDE(
    COUNTIF(duration_ms < 50),
    COUNT(*)
  ) as slo_p99_under_50ms
FROM `PROJECT_ID.analytics.requests`
WHERE DATE(timestamp) = DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY)
GROUP BY day, tenant_tier, route_pattern, edge_region;
*/

-- ============================================================================
-- INDEXES (if needed for faster queries)
-- ============================================================================

-- Note: BigQuery doesn't support traditional indexes, but clustering and
-- partitioning serve the same purpose. Ensure all queries include partition
-- filters to minimize costs.

-- ============================================================================
-- COST ESTIMATES
-- ============================================================================

/*
Assumptions:
- 10 million requests/day
- 1% sampling = 100k rows/day in raw table
- Raw data: 100k rows/day * 2 KB/row * 30 days = 6 GB active storage
- Hourly aggs: 24 hours * 1k unique dimension combos * 2 KB/row * 730 days = 35 GB
- Daily aggs: 365 days * 100 dimension combos * 1 KB/row * 2555 days = 91 GB

Total storage: ~132 GB at $0.02/GB/month = $2.64/month

Query costs (assuming 10 GB scanned/day):
- 10 GB/day * 30 days * $0.005/GB = $1.50/month

Total BigQuery cost: ~$4-5/month at 10M requests/day

At 100M requests/day (10x):
- Storage: ~$26/month
- Queries: ~$15/month
- Total: ~$40-50/month (well within $300 budget)
*/
