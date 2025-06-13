package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"homeinsight-properties/pkg/auth"
	"homeinsight-properties/pkg/config"
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

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := config.LoadConfig("configs/config.yaml")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load config"})
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := auth.ValidateJWT(parts[1], cfg.JWT.Secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("full_name", claims.FullName)
		c.Set("email", claims.Email)
		c.Set("phone", claims.Phone)
		c.Next()
	}
}
