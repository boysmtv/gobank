package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gobank_http_requests_total",
			Help: "Total number of HTTP requests by method, path and status.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gobank_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	TransferTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gobank_transfer_total",
			Help: "Total transfer transactions by status.",
		},
		[]string{"status", "currency"},
	)

	TransferAmount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gobank_transfer_amount",
			Help:    "Transfer amounts in minor currency units.",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000},
		},
		[]string{"currency"},
	)

	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gobank_active_connections",
			Help: "Number of active HTTP connections.",
		},
	)
)
