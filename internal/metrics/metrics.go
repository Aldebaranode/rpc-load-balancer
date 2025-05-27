package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HttpRequestDuration measures incoming HTTP request duration.
	HttpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rpc_gateway_http_request_duration_seconds",
		Help:    "Duration of HTTP requests.",
		Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
	}, []string{"method", "status_code", "endpoint"}) // Added endpoint label

	// HttpRequestTotal counts total incoming HTTP requests.
	HttpRequestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_gateway_http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "status_code", "endpoint"}) // Added endpoint label

	// RpcCheckDuration measures RPC health check duration.
	RpcCheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rpc_gateway_rpc_check_duration_seconds",
		Help:    "Duration of RPC health checks.",
		Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"endpoint"})

	// RpcCheckErrorsTotal counts failed RPC health checks.
	RpcCheckErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_gateway_rpc_check_errors_total",
		Help: "Total number of failed RPC health checks.",
	}, []string{"endpoint", "reason"})

	// RpcRateLimitsTotal counts detected rate limits.
	RpcRateLimitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_gateway_rpc_rate_limits_total",
		Help: "Total number of rate limits detected.",
	}, []string{"endpoint", "source"}) // Source: 'check' or 'proxy'

	// RpcEndpointBlockNumber shows the current block number per endpoint.
	RpcEndpointBlockNumber = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rpc_gateway_rpc_endpoint_block_number",
		Help: "Current block number for each RPC endpoint.",
	}, []string{"endpoint"})

	// RpcEndpointLatency shows the current latency per endpoint.
	RpcEndpointLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rpc_gateway_rpc_endpoint_latency_seconds",
		Help: "Current latency for each RPC endpoint.",
	}, []string{"endpoint"})

	// RpcEndpointIsActive shows if an endpoint is considered active (1) or not (0).
	RpcEndpointIsActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rpc_gateway_rpc_endpoint_is_active",
		Help: "Whether an endpoint is currently considered active (1) or inactive (0).",
	}, []string{"endpoint"})

	// RpcEndpointIsCurrentBest shows if an endpoint is the current best (1) or not (0).
	RpcEndpointIsCurrentBest = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rpc_gateway_rpc_endpoint_is_current_best",
		Help: "Whether an endpoint is the current best choice (1) or not (0).",
	}, []string{"endpoint"})
)

var RpcEndpointCurrentBestActive float64 = 1
var RpcEndpointCurrentBestNotActive float64 = 0

// InitMetrics - We don't strictly need an Init function when using promauto,
// as metrics are registered on creation. This is kept for conceptual clarity
// or if we switch from promauto later.
func InitMetrics() {
	// promauto handles registration, so this can be empty
	// or used for more complex setup if needed.
}
