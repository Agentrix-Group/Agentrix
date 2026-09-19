package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type mockReplaysService struct {
	service.Service
	getReplayFn    func(ctx context.Context, id string) (*model.Replay, error)
	streamReplayFn func(ctx context.Context, id string) ([]byte, error)
}

func (m *mockReplaysService) GetReplay(ctx context.Context, id string) (*model.Replay, error) {
	if m.getReplayFn != nil {
		return m.getReplayFn(ctx, id)
	}
	return &model.Replay{Id: id, DurationTicks: 50}, nil
}

func (m *mockReplaysService) StreamReplay(ctx context.Context, id string) ([]byte, error) {
	if m.streamReplayFn != nil {
		return m.streamReplayFn(ctx, id)
	}
	return []byte(`{"id":"` + id + `"}`), nil
}

func TestServerReplaysHandlers(t *testing.T) {
	r := require.New(t)

	mockSvc := &mockReplaysService{
		getReplayFn: func(ctx context.Context, id string) (*model.Replay, error) {
			if id == "not-found" {
				return nil, errors.New("file not found")
			}
			return &model.Replay{Id: id, DurationTicks: 50}, nil
		},
		streamReplayFn: func(ctx context.Context, id string) ([]byte, error) {
			if id == "not-found" {
				return nil, errors.New("file not found")
			}
			return []byte(`{"id":"` + id + `"}`), nil
		},
	}

	server := &Server{Service: mockSvc}

	// 1. getReplay (found)
	req := httptest.NewRequest(http.MethodGet, "/replays/rep-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rep-1"})
	rec := httptest.NewRecorder()
	server.getReplay(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 2. getReplay (not found)
	req = httptest.NewRequest(http.MethodGet, "/replays/not-found", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-found"})
	rec = httptest.NewRecorder()
	server.getReplay(rec, req)
	r.Equal(http.StatusNotFound, rec.Code)

	// 3. streamReplay (found)
	req = httptest.NewRequest(http.MethodGet, "/replays/rep-1/stream", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rep-1"})
	rec = httptest.NewRecorder()
	server.streamReplay(rec, req)
	r.Equal(http.StatusOK, rec.Code)
	r.Equal("application/x-ndjson", rec.Header().Get("Content-Type"))

	// 4. streamReplay (not found)
	req = httptest.NewRequest(http.MethodGet, "/replays/not-found/stream", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-found"})
	rec = httptest.NewRecorder()
	server.streamReplay(rec, req)
	r.Equal(http.StatusNotFound, rec.Code)
}

func TestReplayStreamIsPublic(t *testing.T) {
	server := NewServer(&mockReplaysService{
		streamReplayFn: func(context.Context, string) ([]byte, error) {
			return []byte("{\"type\":\"metadata\"}\n"), nil
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/replays/rep-public/stream", nil)
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "application/x-ndjson", response.Header().Get("Content-Type"))
}
