package cache

import (
	"fmt"
)

type CacheError struct {
	Operation string
	Err       error
	Retryable bool
}

func NewCacheError(operation string, err error, retryable bool) *CacheError {
	return &CacheError{
		Operation: operation,
		Err:       err,
		Retryable: retryable,
	}
}

func (e *CacheError) Error() string {
	return fmt.Sprintf("cache operation %s failed: %v", e.Operation, e.Err)
}

func (e *CacheError) Unwrap() error {
	return e.Err
}
