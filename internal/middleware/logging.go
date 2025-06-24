package middleware

import (
	"fmt"
	"time"

	"homeinsight-properties/pkg/logger"

	"github.com/gin-gonic/gin"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		// Process request
		c.Next()

		// Log after request
		latency := time.Since(start)
		status := c.Writer.Status()

		// Get data source information if available
		dataSource := c.GetString("data_source")
		cacheHit := c.GetBool("cache_hit")

		// Build log message with data source info
		logMsg := ""
		if dataSource != "" {
			if cacheHit {
				logMsg = fmt.Sprintf("%s %s %d %v [%s] - DATA_SOURCE: %s (CACHE_HIT)",
					method, path, status, latency, clientIP, dataSource)
			} else {
				logMsg = fmt.Sprintf("%s %s %d %v [%s] - DATA_SOURCE: %s (CACHE_MISS)",
					method, path, status, latency, clientIP, dataSource)
			}
		} else {
			logMsg = fmt.Sprintf("%s %s %d %v [%s]",
				method, path, status, latency, clientIP)
		}

		logger.Logger.Println(logMsg)
	}
}
