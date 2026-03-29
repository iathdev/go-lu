package sharederror

import (
	"errors"
)

// Code represents a machine-readable error code (SCREAMING_SNAKE_CASE).
// Used as the "code" field in API error responses.
type Code string

const (
	CodeBadRequest          Code = "BAD_REQUEST"           // 400
	CodeUnauthorized        Code = "UNAUTHORIZED"          // 401
	CodeForbidden           Code = "FORBIDDEN"             // 403
	CodeNotFound            Code = "NOT_FOUND"             // 404
	CodeConflict            Code = "CONFLICT"              // 409
	CodeValidationFailed    Code = "VALIDATION_FAILED"     // 422
	CodeTooManyRequests     Code = "TOO_MANY_REQUESTS"     // 429
	CodeInternalServerError Code = "INTERNAL_SERVER_ERROR" // 500
	CodeServiceUnavailable  Code = "SERVICE_UNAVAILABLE"   // 503
)

// AppError is the unified error type used across use cases and handlers.
// 4xx errors carry no cause; 5xx errors carry the underlying cause for logging.
type AppError struct {
	code    Code
	message string
	cause   error
	data    map[string]any
}

func (appErr *AppError) Error() string {
	return string(appErr.code) + ": " + appErr.message
}

func (appErr *AppError) Code() Code           { return appErr.code }
func (appErr *AppError) Message() string      { return appErr.message }
func (appErr *AppError) Unwrap() error        { return appErr.cause }
func (appErr *AppError) Data() map[string]any { return appErr.data }

// WithData attaches arbitrary key-value details to the error.
// Rendered as "details" in the API error response.
func (appErr *AppError) WithData(data map[string]any) *AppError {
	appErr.data = data
	return appErr
}

// WithCode overrides the default error code with a granular business code.
// Example: Forbidden("entitlement.not_entitled").WithCode("FEATURE_NOT_ENTITLED")
func (appErr *AppError) WithCode(code Code) *AppError {
	appErr.code = code
	return appErr
}

func (appErr *AppError) Is(target error) bool {
	var t *AppError
	if errors.As(target, &t) {
		return appErr.code == t.code
	}
	return false
}

// --- Client errors (4xx): no cause, no logging needed ---

func BadRequest(message string) *AppError {
	return &AppError{code: CodeBadRequest, message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{code: CodeUnauthorized, message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{code: CodeForbidden, message: message}
}

func NotFound(message string) *AppError {
	return &AppError{code: CodeNotFound, message: message}
}

func Conflict(message string) *AppError {
	return &AppError{code: CodeConflict, message: message}
}

func ValidationFailed(message string) *AppError {
	return &AppError{code: CodeValidationFailed, message: message}
}

// UnprocessableEntity creates a 422 error. Alias kept for backward compatibility.
// Prefer ValidationFailed for new code.
func UnprocessableEntity(message string) *AppError {
	return &AppError{code: CodeValidationFailed, message: message}
}

func TooManyRequests(message string) *AppError {
	return &AppError{code: CodeTooManyRequests, message: message}
}

// --- Server errors (5xx): carry cause for handler-layer logging ---

func InternalServerError(message string, cause error) *AppError {
	return &AppError{code: CodeInternalServerError, message: message, cause: cause}
}

func ServiceUnavailable(message string, cause error) *AppError {
	return &AppError{code: CodeServiceUnavailable, message: message, cause: cause}
}

// IsAppError extracts an AppError from the error chain, if present.
func IsAppError(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
