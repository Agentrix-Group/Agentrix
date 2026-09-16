package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	r := require.New(t)

	s := NewServer(&mockContestService{})
	r.NotNil(s)
	r.NotNil(s.Handler)
	r.NotNil(s.SessionStore)
	r.NotNil(s.Auth)

	// Test route routing on built handler
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contests", nil)
	rec := httptest.NewRecorder()
	s.Handler.ServeHTTP(rec, req)

	// Will execute correlation & cors middleware and route
	r.NotEmpty(rec.Header().Get("X-Request-Id"))
	r.Equal(http.StatusOK, rec.Code)
}
