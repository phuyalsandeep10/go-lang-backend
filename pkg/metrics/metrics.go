package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// HTTP Metrics
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	// Redis Metrics
	CacheHitsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_hits_total",
			Help: "Total number of Redis cache hits",
		},
	)
	CacheMissesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_misses_total",
			Help: "Total number of Redis cache misses",
		},
	)
	RedisOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_seconds",
			Help:    "Duration of Redis operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
	RedisErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_errors_total",
			Help: "Total number of Redis operation errors",
		},
		[]string{"operation"},
	)

	// MongoDB Metrics
	MongoOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mongodb_operation_duration_seconds",
			Help:    "Duration of MongoDB operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "collection"},
	)
	MongoErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mongodb_errors_total",
			Help: "Total number of MongoDB operation errors",
		},
		[]string{"operation", "collection"},
	)
)

func Init() {
	prometheus.MustRegister(HTTPRequestsTotal)
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(CacheHitsTotal)
	prometheus.MustRegister(CacheMissesTotal)
	prometheus.MustRegister(RedisOperationDuration)
	prometheus.MustRegister(RedisErrorsTotal)
	prometheus.MustRegister(MongoOperationDuration)
	prometheus.MustRegister(MongoErrorsTotal)
}
