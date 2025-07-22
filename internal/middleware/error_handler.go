package middleware

import (

	"homeinsight-properties/internal/errors"
	"homeinsight-properties/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ErrorHandler catches errors and returns standardized responses.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			appErr := errors.MapError(err)

			// Log technical details
			logger.GlobalLogger.Errorf("Request failed: path=%s, method=%s, client_ip=%s, error=%s",
				c.Request.URL.Path,
				c.Request.Method,
				c.ClientIP(),
				appErr.TechnicalMessage)

			c.JSON(appErr.HTTPStatus, gin.H{
				"error": gin.H{
					"message": appErr.UserMessage,
					"code":    appErr.Code,
				},
			})
			return
		}
	}
}
