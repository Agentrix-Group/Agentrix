package common

import "net/http"

const (
	// Client errors (1xxx)
	INVALID_REQUEST_ERROR     = 1001
	MISSING_FIELDS_ERROR      = 1002
	NOT_FOUND_ERROR           = 1003
	ALREADY_EXISTS_ERROR      = 1004
	RATE_LIMIT_EXCEEDED_ERROR = 1005

	// Auth errors (2xxx)
	INVALID_CREDENTIALS_ERROR = 2001
	ACCESS_DENIED_ERROR       = 2002
	MISSING_PERMISSION_ERROR  = 2003

	// Game & Execution errors (3xxx)
	INVALID_GAME_ACTION_ERROR = 3001
	EXECUTION_TIMEOUT_ERROR   = 3002
	GAME_NOT_FOUND_ERROR      = 3003

	// Server errors (5xxx)
	DATABASE_ERROR = 5001
	INTERNAL_ERROR = 5002
)

var ErrorCodes = map[int]int{
	INVALID_REQUEST_ERROR:     http.StatusBadRequest,
	MISSING_FIELDS_ERROR:      http.StatusBadRequest,
	NOT_FOUND_ERROR:           http.StatusNotFound,
	ALREADY_EXISTS_ERROR:      http.StatusConflict,
	RATE_LIMIT_EXCEEDED_ERROR: http.StatusTooManyRequests,
	INVALID_CREDENTIALS_ERROR: http.StatusUnauthorized,
	ACCESS_DENIED_ERROR:       http.StatusUnauthorized,
	MISSING_PERMISSION_ERROR:  http.StatusForbidden,
	INVALID_GAME_ACTION_ERROR: http.StatusBadRequest,
	EXECUTION_TIMEOUT_ERROR:   http.StatusGatewayTimeout,
	GAME_NOT_FOUND_ERROR:      http.StatusNotFound,
	DATABASE_ERROR:            http.StatusInternalServerError,
	INTERNAL_ERROR:            http.StatusInternalServerError,
}

var ErrorMessages = map[int]string{
	INVALID_REQUEST_ERROR:     "Invalid request format",
	MISSING_FIELDS_ERROR:      "Required fields are missing",
	NOT_FOUND_ERROR:           "Resource not found",
	ALREADY_EXISTS_ERROR:      "Resource already exists",
	RATE_LIMIT_EXCEEDED_ERROR: "Too many requests. Please try again later.",
	INVALID_CREDENTIALS_ERROR: "Invalid credentials provided",
	ACCESS_DENIED_ERROR:       "Authentication required",
	MISSING_PERMISSION_ERROR:  "Insufficient permissions",
	INVALID_GAME_ACTION_ERROR: "Invalid game action submitted",
	EXECUTION_TIMEOUT_ERROR:   "Agent execution timed out",
	GAME_NOT_FOUND_ERROR:      "Game engine not found",
	DATABASE_ERROR:            "Database operation failed",
	INTERNAL_ERROR:            "Internal server error",
}
