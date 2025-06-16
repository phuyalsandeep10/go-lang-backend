package middleware

import (
	"strconv"
	"time"

	"homeinsight-properties/pkg/metrics"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Observe(duration)

		// Track cache hits/misses (based on context values set by handlers)
		if cacheHit, exists := c.Get("cache_hit"); exists && cacheHit.(bool) {
			metrics.CacheHitsTotal.Inc()
		} else if exists {
			metrics.CacheMissesTotal.Inc()
		}
	}
}
