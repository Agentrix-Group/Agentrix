package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteErrorResponse(t *testing.T) {
	r := require.New(t)

	testCases := []struct {
		name      string
		errorCode int
	}{
		{
			name:      "InternalError",
			errorCode: INTERNAL_ERROR,
		},
		{
			name:      "DatabaseError",
			errorCode: DATABASE_ERROR,
		},
		{
			name:      "NotFoundError",
			errorCode: NOT_FOUND_ERROR,
		},
		{
			name:      "InvalidCredentials",
			errorCode: INVALID_CREDENTIALS_ERROR,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteErrorResponse(w, tc.errorCode)

			r.Equal(ContentTypeJSON, w.Header().Get("Content-Type"))
			r.Equal(ErrorCodes[tc.errorCode], w.Code)

			var response Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			r.NoError(err)
			r.Equal(ErrorCodes[tc.errorCode], response.HttpStatusCode)
			r.Equal(tc.errorCode, response.ErrorCode)
			r.Equal(ErrorMessages[tc.errorCode], response.Message)
		})
	}
}

func TestWriteSuccessResponse(t *testing.T) {
	r := require.New(t)
	w := httptest.NewRecorder()

	WriteSuccessResponse(w, http.StatusOK, "Created")

	r.Equal(ContentTypeJSON, w.Header().Get("Content-Type"))
	r.Equal(http.StatusOK, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	r.NoError(err)
	r.Equal(http.StatusOK, response.HttpStatusCode)
	r.Equal("Created", response.Message)
}
