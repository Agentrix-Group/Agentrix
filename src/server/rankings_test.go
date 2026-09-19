package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type mockRankingsService struct {
	service.Service
	listRankingsFn          func(ctx context.Context) ([]model.Ranking, error)
	listRankingsByContestFn func(ctx context.Context, contestId string) ([]model.Ranking, error)
	getRankingFn            func(ctx context.Context, id string) (*model.Ranking, error)
}

func (m *mockRankingsService) ListRankings(ctx context.Context) ([]model.Ranking, error) {
	if m.listRankingsFn != nil {
		return m.listRankingsFn(ctx)
	}
	return nil, nil
}

func (m *mockRankingsService) ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error) {
	if m.listRankingsByContestFn != nil {
		return m.listRankingsByContestFn(ctx, contestId)
	}
	return nil, nil
}

func (m *mockRankingsService) GetRanking(ctx context.Context, id string) (*model.Ranking, error) {
	if m.getRankingFn != nil {
		return m.getRankingFn(ctx, id)
	}
	return &model.Ranking{Id: id, Score: 100}, nil
}

func TestServerRankingsHandlers(t *testing.T) {
	r := require.New(t)

	mockSvc := &mockRankingsService{
		listRankingsFn: func(ctx context.Context) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r1", Score: 100}}, nil
		},
		listRankingsByContestFn: func(ctx context.Context, contestId string) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r1", ContestId: contestId, Score: 100}}, nil
		},
		getRankingFn: func(ctx context.Context, id string) (*model.Ranking, error) {
			if id == "not-found" {
				return nil, sql.ErrNoRows
			}
			return &model.Ranking{Id: id, Score: 100}, nil
		},
	}

	server := &Server{Service: mockSvc}

	// 1. listRankings (all)
	req := httptest.NewRequest(http.MethodGet, "/rankings", nil)
	rec := httptest.NewRecorder()
	server.listRankings(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 2. listRankings (by contest)
	req = httptest.NewRequest(http.MethodGet, "/rankings?contest_id=c1", nil)
	rec = httptest.NewRecorder()
	server.listRankings(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 3. getRanking (found)
	req = httptest.NewRequest(http.MethodGet, "/rankings/r1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "r1"})
	rec = httptest.NewRecorder()
	server.getRanking(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 4. getRanking (not found)
	req = httptest.NewRequest(http.MethodGet, "/rankings/not-found", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-found"})
	rec = httptest.NewRecorder()
	server.getRanking(rec, req)
	r.Equal(http.StatusNotFound, rec.Code)
}

func TestRankingRoutesArePublic(t *testing.T) {
	server := NewServer(&mockRankingsService{
		listRankingsFn: func(context.Context) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r-public", Score: 10}}, nil
		},
		getRankingFn: func(_ context.Context, id string) (*model.Ranking, error) {
			return &model.Ranking{Id: id, Score: 10}, nil
		},
	})

	for _, path := range []string{"/api/v1/rankings", "/api/v1/rankings/r-public"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, path)
	}
}
