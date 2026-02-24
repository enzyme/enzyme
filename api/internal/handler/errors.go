package handler

import "github.com/enzyme/api/internal/openapi"

// newError creates an ApiError with the given code and message
func newError(code, message string) openapi.ApiError {
	return openapi.ApiError{
		Code:    code,
		Message: message,
	}
}

// newErrorResponse creates an ApiErrorResponse with the given code and message
func newErrorResponse(code, message string) openapi.ApiErrorResponse {
	return openapi.ApiErrorResponse{
		Error: newError(code, message),
	}
}

// Common error codes
const (
	ErrCodeInvalidJSON      = "INVALID_JSON"
	ErrCodeInternalError    = "INTERNAL_ERROR"
	ErrCodeNotAuthenticated = "NOT_AUTHENTICATED"
	ErrCodeNotAMember       = "NOT_A_MEMBER"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
	ErrCodeValidationError  = "VALIDATION_ERROR"
	ErrCodeConflict         = "CONFLICT"
)

// Error response helpers that return typed shared response components.
// Usage: return openapi.SendMessage401JSONResponse{unauthorizedResponse()}, nil

func unauthorizedResponse() openapi.UnauthorizedJSONResponse {
	return openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated"))
}

func forbiddenResponse(msg string) openapi.ForbiddenJSONResponse {
	return openapi.ForbiddenJSONResponse(newErrorResponse(ErrCodePermissionDenied, msg))
}

func notFoundResponse(msg string) openapi.NotFoundJSONResponse {
	return openapi.NotFoundJSONResponse(newErrorResponse(ErrCodeNotFound, msg))
}

func badRequestResponse(code, msg string) openapi.BadRequestJSONResponse {
	return openapi.BadRequestJSONResponse(newErrorResponse(code, msg))
}

func notAMemberResponse(msg string) openapi.ForbiddenJSONResponse {
	return openapi.ForbiddenJSONResponse(newErrorResponse(ErrCodeNotAMember, msg))
}

func conflictResponse(msg string) openapi.ConflictJSONResponse {
	return openapi.ConflictJSONResponse(newErrorResponse(ErrCodeConflict, msg))
}
