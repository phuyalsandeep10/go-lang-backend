package utils

import (
	"homeinsight-properties/pkg/metrics"
	"time"
)

func RecordMongoOperationDuration(operation, collection string, start time.Time) {
	duration := time.Since(start).Seconds()
	metrics.MongoOperationDuration.WithLabelValues(operation, collection).Observe(duration)
}

func RecordMongoError(operation, collection string) {
	metrics.MongoErrorsTotal.WithLabelValues(operation, collection).Inc()
}

func RecordRedisOperationDuration(operation string, start time.Time) {
	duration := time.Since(start).Seconds()
	metrics.RedisOperationDuration.WithLabelValues(operation).Observe(duration)
}

func RecordRedisError(operation string) {
	metrics.RedisErrorsTotal.WithLabelValues(operation, "").Inc()
}
