package utils

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"homeinsight-properties/internal/errors"

	"github.com/gin-gonic/gin"
)

// LogAndMapError logs technical details and returns a user-friendly AppError.
func LogAndMapError(ctx context.Context, err error, operation string, params ...interface{}) *errors.AppError {
	appErr := errors.MapError(err)
	if appErr == nil {
		return nil
	}

	ginCtx, _ := ctx.(*gin.Context)
	if ginCtx == nil {
		ginCtx = &gin.Context{}
	}

	// Log technical details
	logDetails := map[string]interface{}{
		"operation":       operation,
		"technical_error": appErr.TechnicalMessage,
	}
	for i := 0; i < len(params); i += 2 {
		if i+1 < len(params) {
			logDetails[fmt.Sprintf("%v", params[i])] = params[i+1]
		}
	}

	return appErr
}

// WrapError adds context to an error while preserving the original.
func WrapError(err error, message string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(message, args...), err)
}

// IsRetryableError determines if an error is transient and worth retrying.
func IsRetryableError(err error) bool {
	if appErr, ok := err.(*errors.AppError); ok {
		return appErr.HTTPStatus == http.StatusServiceUnavailable ||
			strings.Contains(appErr.TechnicalMessage, "timeout") ||
			strings.Contains(appErr.TechnicalMessage, "connection")
	}
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection")
}
