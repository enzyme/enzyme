package handler

import "github.com/feather/api/internal/api"

// newError creates an ApiError with the given code and message
func newError(code, message string) api.ApiError {
	return api.ApiError{
		Code:    code,
		Message: message,
	}
}

// newErrorResponse creates an ApiErrorResponse with the given code and message
func newErrorResponse(code, message string) api.ApiErrorResponse {
	return api.ApiErrorResponse{
		Error: newError(code, message),
	}
}

// Common error codes
const (
	ErrCodeInvalidJSON       = "INVALID_JSON"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeNotAuthenticated  = "NOT_AUTHENTICATED"
	ErrCodeNotAMember        = "NOT_A_MEMBER"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodePermissionDenied  = "PERMISSION_DENIED"
	ErrCodeValidationError   = "VALIDATION_ERROR"
	ErrCodeConflict          = "CONFLICT"
)
