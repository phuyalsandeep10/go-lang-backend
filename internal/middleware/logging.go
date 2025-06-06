package middleware

import (
	"time"

	"homeinsight-properties/pkg/logger"

	"github.com/gin-gonic/gin"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Log after request
		latency := time.Since(start)
		status := c.Writer.Status()
		logger.Logger.Printf("%s %s %d %v", method, path, status, latency)
	}
}
