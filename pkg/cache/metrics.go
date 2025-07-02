package cache

import (
	"homeinsight-properties/pkg/metrics"
)

//record the duration of a Redis operation with the given label.
func RecordOperationDuration(label string, duration float64) {
	metrics.RedisOperationDuration.WithLabelValues(label).Observe(duration)
}

// increment the error counter for a Redis operation with the given label.
func IncrementError(label string) {
	metrics.RedisErrorsTotal.WithLabelValues(label).Inc()
}
