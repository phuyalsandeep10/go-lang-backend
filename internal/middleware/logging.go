package middleware

import (
	"encoding/json"
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

		// Core log fields
		logFields := map[string]interface{}{
			"ts":   time.Now().UTC().Format(time.RFC3339),
			"m":    method,
			"p":    path,
			"s":    c.Writer.Status(),
			"l_ms": time.Since(start).Milliseconds(),
			"ip":   clientIP,
		}

		// Conditionally add route-specific fields
		if ds, exists := c.Get("data_source"); exists && ds != "" {
			logFields["ds"] = ds
		}
		if ch, exists := c.Get("cache_hit"); exists {
			logFields["ch"] = ch
		}
		if q, exists := c.Get("query"); exists && q != "" {
			logFields["q"] = q
		}
		if pid, exists := c.Get("property_id"); exists && pid != "" {
			logFields["pid"] = pid
		}

		logJSON, err := json.Marshal(logFields)
		if err != nil {
			logger.Logger.Printf("Failed to marshal log: %v", err)
			return
		}

		logger.Logger.Println(string(logJSON))
	}
}
