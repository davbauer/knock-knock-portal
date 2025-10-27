package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorCode represents a structured error code
type ErrorCode string

const (
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrCodeRateLimit      ErrorCode = "RATE_LIMIT"
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeBadGateway     ErrorCode = "BAD_GATEWAY"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeCircuitOpen    ErrorCode = "CIRCUIT_OPEN"
	ErrCodeResourceLimit  ErrorCode = "RESOURCE_LIMIT"
)

// AppError represents a structured application error
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Underlying error                  `json:"-"`
	StackTrace []string               `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Underlying)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap implements the unwrap interface for error chains
func (e *AppError) Unwrap() error {
	return e.Underlying
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) []string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	
	frames := runtime.CallersFrames(pcs[:n])
	trace := make([]string, 0, n)
	
	for {
		frame, more := frames.Next()
		trace = append(trace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	
	return trace
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StackTrace: captureStackTrace(1),
	}
}

// Newf creates a new AppError with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		StackTrace: captureStackTrace(1),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}
	
	return &AppError{
		Code:       code,
		Message:    message,
		Underlying: err,
		StackTrace: captureStackTrace(1),
	}
}

// Wrapf wraps an existing error with formatted message
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	if err == nil {
		return nil
	}
	
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		Underlying: err,
		StackTrace: captureStackTrace(1),
	}
}

// WithDetails adds details to an error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// WithDetail adds a single detail to an error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// GetStackTrace returns a formatted stack trace
func (e *AppError) GetStackTrace() string {
	return strings.Join(e.StackTrace, "\n")
}

// Common error constructors

// NewValidationError creates a validation error
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       ErrCodeValidation,
		Message:    message,
		Details:    details,
		StackTrace: captureStackTrace(1),
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		StackTrace: captureStackTrace(1),
	}
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		StackTrace: captureStackTrace(1),
	}
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeForbidden,
		Message:    message,
		StackTrace: captureStackTrace(1),
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError() *AppError {
	return &AppError{
		Code:       ErrCodeRateLimit,
		Message:    "Rate limit exceeded",
		StackTrace: captureStackTrace(1),
	}
}

// NewInternalError creates an internal error
func NewInternalError(message string, underlying error) *AppError {
	return &AppError{
		Code:       ErrCodeInternal,
		Message:    message,
		Underlying: underlying,
		StackTrace: captureStackTrace(1),
	}
}

// NewCircuitOpenError creates a circuit open error
func NewCircuitOpenError(service string) *AppError {
	return &AppError{
		Code:    ErrCodeCircuitOpen,
		Message: fmt.Sprintf("Circuit breaker open for %s", service),
		Details: map[string]interface{}{
			"service": service,
		},
		StackTrace: captureStackTrace(1),
	}
}

// NewResourceLimitError creates a resource limit error
func NewResourceLimitError(resource string, limit int) *AppError {
	return &AppError{
		Code:    ErrCodeResourceLimit,
		Message: fmt.Sprintf("Resource limit exceeded for %s", resource),
		Details: map[string]interface{}{
			"resource": resource,
			"limit":    limit,
		},
		StackTrace: captureStackTrace(1),
	}
}

// IsErrorCode checks if an error has a specific error code
func IsErrorCode(err error, code ErrorCode) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrCodeInternal
}
