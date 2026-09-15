package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCorrelationMiddlewareCreatesSafeRequestID(t *testing.T) {
	server := &Server{}
	handler := server.correlationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-Id", "invalid\nlog-entry")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	generated := response.Header().Get("X-Request-Id")
	require.NotEmpty(t, generated)
	_, err := uuid.Parse(generated)
	require.NoError(t, err)
}

func TestCorrelationMiddlewarePreservesValidRequestID(t *testing.T) {
	server := &Server{}
	requestID := uuid.New().String()
	handler := server.correlationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-Id", requestID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, requestID, response.Header().Get("X-Request-Id"))
}

func TestCorrelationMiddlewarePreservesSafeExternalTraceID(t *testing.T) {
	server := &Server{}
	requestID := "gateway:trace-123"
	handler := server.correlationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-Id", requestID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, requestID, response.Header().Get("X-Request-Id"))
}

func TestStatusLoggingResponseWriterKeepsFirstStatus(t *testing.T) {
	response := httptest.NewRecorder()
	writer := &statusLoggingResponseWriter{ResponseWriter: response, statusCode: http.StatusOK}

	_, err := writer.Write([]byte("ok"))
	require.NoError(t, err)
	writer.WriteHeader(http.StatusInternalServerError)

	require.Equal(t, http.StatusOK, writer.statusCode)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "ok", response.Body.String())
}
