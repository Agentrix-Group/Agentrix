package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorCodesMapping(t *testing.T) {
	r := require.New(t)

	expectedCodes := []int{
		INVALID_REQUEST_ERROR,
		MISSING_FIELDS_ERROR,
		NOT_FOUND_ERROR,
		ALREADY_EXISTS_ERROR,
		RATE_LIMIT_EXCEEDED_ERROR,
		INVALID_CREDENTIALS_ERROR,
		ACCESS_DENIED_ERROR,
		MISSING_PERMISSION_ERROR,
		INVALID_GAME_ACTION_ERROR,
		EXECUTION_TIMEOUT_ERROR,
		GAME_NOT_FOUND_ERROR,
		DATABASE_ERROR,
		INTERNAL_ERROR,
	}

	for _, code := range expectedCodes {
		httpStatus, hasStatus := ErrorCodes[code]
		r.True(hasStatus, "code %d should have an HTTP status in ErrorCodes", code)
		r.True(httpStatus >= 400 && httpStatus < 600, "code %d should map to a 4xx or 5xx HTTP status, got %d", code, httpStatus)

		msg, hasMsg := ErrorMessages[code]
		r.True(hasMsg, "code %d should have a message in ErrorMessages", code)
		r.NotEmpty(msg, "message for code %d should not be empty", code)
	}

	r.Equal(http.StatusBadRequest, ErrorCodes[INVALID_REQUEST_ERROR])
	r.Equal(http.StatusUnauthorized, ErrorCodes[INVALID_CREDENTIALS_ERROR])
	r.Equal(http.StatusForbidden, ErrorCodes[MISSING_PERMISSION_ERROR])
	r.Equal(http.StatusNotFound, ErrorCodes[NOT_FOUND_ERROR])
	r.Equal(http.StatusConflict, ErrorCodes[ALREADY_EXISTS_ERROR])
	r.Equal(http.StatusInternalServerError, ErrorCodes[DATABASE_ERROR])
	r.Equal(http.StatusInternalServerError, ErrorCodes[INTERNAL_ERROR])
	r.Equal(http.StatusGatewayTimeout, ErrorCodes[EXECUTION_TIMEOUT_ERROR])
}
