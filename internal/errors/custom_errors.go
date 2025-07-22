package errors

import (
	"fmt"
)

// AppError represents a structured application error with user-friendly and technical details.
type AppError struct {
	TechnicalMessage string
	UserMessage     string
	Code            string
	HTTPStatus      int
	OriginalError   error  
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %v", e.UserMessage, e.OriginalError)
}

// Unwrap returns the original error for error chaining.
func (e *AppError) Unwrap() error {
	return e.OriginalError
}

// NewAppError creates a new AppError instance.
func NewAppError(technicalMessage, userMessage, code string, status int, originalErr error) *AppError {
	return &AppError{
		TechnicalMessage: technicalMessage,
		UserMessage:      userMessage,
		Code:             code,
		HTTPStatus:       status,
		OriginalError:    originalErr,
	}
}

// Common error codes
const (
	ErrCodeInvalidAddress      = "INVALID_ADDRESS"
	ErrCodePropertyNotFound    = "PROPERTY_NOT_FOUND"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimited         = "RATE_LIMITED"
	ErrCodeInvalidParameters   = "INVALID_PARAMETERS"
)
