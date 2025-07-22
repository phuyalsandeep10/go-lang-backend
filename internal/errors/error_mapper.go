
package errors

import (
	"net/http"
	"strings"
)

// MapError converts a technical error into a user-friendly AppError.
func MapError(err error) *AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	technicalMessage := err.Error()

	// Map specific error patterns to user-friendly errors
	switch {
	case strings.Contains(technicalMessage, "CoreLogic") && (strings.Contains(technicalMessage, "404 Not Found") || strings.Contains(technicalMessage, "Clip not found")):
		return &AppError{
			TechnicalMessage: technicalMessage,
			UserMessage:      MsgPropertyNotFound,
			Code:             ErrCodePropertyNotFound,
			HTTPStatus:       http.StatusNotFound,
			OriginalError:    err,
		}
	case strings.Contains(technicalMessage, "CoreLogic"):
		return &AppError{
			TechnicalMessage: technicalMessage,
			UserMessage:      MsgServiceUnavailable,
			Code:             ErrCodeServiceUnavailable,
			HTTPStatus:       http.StatusServiceUnavailable,
			OriginalError:    err,
		}
	case strings.Contains(technicalMessage, "street address and city are required"):
		return &AppError{
			TechnicalMessage: technicalMessage,
			UserMessage:      MsgInvalidAddress,
			Code:             ErrCodeInvalidAddress,
			HTTPStatus:       http.StatusBadRequest,
			OriginalError:    err,
		}
	case strings.Contains(technicalMessage, "database query failed"):
		return &AppError{
			TechnicalMessage: technicalMessage,
			UserMessage:      MsgServiceUnavailable,
			Code:             ErrCodeServiceUnavailable,
			HTTPStatus:       http.StatusServiceUnavailable,
			OriginalError:    err,
		}
	case strings.Contains(technicalMessage, "property not found"):
		return &AppError{
			TechnicalMessage: technicalMessage,
			UserMessage:      MsgPropertyNotFound,
			Code:             ErrCodePropertyNotFound,
			HTTPStatus:       http.StatusNotFound,
			OriginalError:    err,
		}
	default:
		return &AppError{
			TechnicalMessage: technicalMessage,
			UserMessage:      MsgInternalError,
			Code:             "INTERNAL_ERROR",
			HTTPStatus:       http.StatusInternalServerError,
			OriginalError:    err,
		}
	}
}
