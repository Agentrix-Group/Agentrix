package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type mockResultsService struct {
	service.Service
	listResultsFn        func(ctx context.Context) ([]model.Result, error)
	listResultsByMatchFn func(ctx context.Context, matchId string) ([]model.Result, error)
	getResultFn          func(ctx context.Context, id string) (*model.Result, error)
}

func (m *mockResultsService) ListResults(ctx context.Context) ([]model.Result, error) {
	if m.listResultsFn != nil {
		return m.listResultsFn(ctx)
	}
	return nil, nil
}

func (m *mockResultsService) ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error) {
	if m.listResultsByMatchFn != nil {
		return m.listResultsByMatchFn(ctx, matchId)
	}
	return nil, nil
}

func (m *mockResultsService) GetResult(ctx context.Context, id string) (*model.Result, error) {
	if m.getResultFn != nil {
		return m.getResultFn(ctx, id)
	}
	return &model.Result{Id: id, Score: 100}, nil
}

func TestServerResultsHandlers(t *testing.T) {
	r := require.New(t)

	mockSvc := &mockResultsService{
		listResultsFn: func(ctx context.Context) ([]model.Result, error) {
			return []model.Result{{Id: "res-1", Score: 100}}, nil
		},
		listResultsByMatchFn: func(ctx context.Context, matchId string) ([]model.Result, error) {
			return []model.Result{{Id: "res-1", MatchId: matchId, Score: 100}}, nil
		},
		getResultFn: func(ctx context.Context, id string) (*model.Result, error) {
			if id == "not-found" {
				return nil, sql.ErrNoRows
			}
			return &model.Result{Id: id, Score: 100}, nil
		},
	}

	server := &Server{Service: mockSvc}

	// 1. listResults (all)
	req := httptest.NewRequest(http.MethodGet, "/results", nil)
	rec := httptest.NewRecorder()
	server.listResults(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 2. listResults (by match)
	req = httptest.NewRequest(http.MethodGet, "/results?match_id=m1", nil)
	rec = httptest.NewRecorder()
	server.listResults(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 3. getResult (found)
	req = httptest.NewRequest(http.MethodGet, "/results/res-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "res-1"})
	rec = httptest.NewRecorder()
	server.getResult(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 4. getResult (not found)
	req = httptest.NewRequest(http.MethodGet, "/results/not-found", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-found"})
	rec = httptest.NewRecorder()
	server.getResult(rec, req)
	r.Equal(http.StatusNotFound, rec.Code)
}
