package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// File upload metrics
	FileUploadsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "file_uploads_total",
			Help: "Total number of file uploads",
		},
	)

	FileUploadBytes = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "file_upload_bytes",
			Help:    "Size of uploaded files in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 104857600, 1073741824}, // 1KB to 1GB
		},
	)

	// Deduplication metrics
	DeduplicationHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "deduplication_hits_total",
			Help: "Total number of deduplication hits",
		},
	)

	DeduplicationSavedBytes = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "deduplication_saved_bytes_total",
			Help: "Total bytes saved through deduplication",
		},
	)

	// Storage metrics
	StorageUsedBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "storage_used_bytes",
			Help: "Current storage usage in bytes",
		},
	)

	StorageQuotaBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "storage_quota_bytes",
			Help: "Storage quota in bytes",
		},
	)

	// Database metrics
	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)

	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation"},
	)

	// Rate limiting metrics
	RateLimitExceededTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_exceeded_total",
			Help: "Total number of rate limit violations",
		},
		[]string{"user_id"},
	)
)