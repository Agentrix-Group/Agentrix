package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type mockRankingsService struct {
	service.Service
	listRankingsFn            func(ctx context.Context) ([]model.Ranking, error)
	listRankingsByContestFn   func(ctx context.Context, contestId string) ([]model.Ranking, error)
	getRankingFn              func(ctx context.Context, id string) (*model.Ranking, error)
	recalculateContestRanksFn func(ctx context.Context, contestId string) ([]model.Ranking, error)
	publishRankingSnapshotFn  func(ctx context.Context, contestId, publisherUserId string) (*model.RankingSnapshot, error)
	listRankingSnapshotsFn    func(ctx context.Context, contestId string) ([]model.RankingSnapshot, error)
	getPublishedRankingsFn    func(ctx context.Context, contestId string, version ...int) (*model.RankingSnapshot, error)
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

func (m *mockRankingsService) RecalculateContestRankings(ctx context.Context, contestId string) ([]model.Ranking, error) {
	if m.recalculateContestRanksFn != nil {
		return m.recalculateContestRanksFn(ctx, contestId)
	}
	return []model.Ranking{{Id: "r1", ContestId: contestId, Rank: 1, Points: 3}}, nil
}

func (m *mockRankingsService) PublishRankingSnapshot(ctx context.Context, contestId, publisherUserId string) (*model.RankingSnapshot, error) {
	if m.publishRankingSnapshotFn != nil {
		return m.publishRankingSnapshotFn(ctx, contestId, publisherUserId)
	}
	return &model.RankingSnapshot{Id: "snap-1", ContestId: contestId, Version: 1}, nil
}

func (m *mockRankingsService) ListRankingSnapshots(ctx context.Context, contestId string) ([]model.RankingSnapshot, error) {
	if m.listRankingSnapshotsFn != nil {
		return m.listRankingSnapshotsFn(ctx, contestId)
	}
	return []model.RankingSnapshot{{Id: "snap-1", ContestId: contestId, Version: 1}}, nil
}

func (m *mockRankingsService) GetPublishedRankings(ctx context.Context, contestId string, version ...int) (*model.RankingSnapshot, error) {
	if m.getPublishedRankingsFn != nil {
		return m.getPublishedRankingsFn(ctx, contestId, version...)
	}
	ver := 1
	if len(version) > 0 {
		ver = version[0]
	}
	return &model.RankingSnapshot{Id: "snap-1", ContestId: contestId, Version: ver}, nil
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

	// 5. recalculateRankings
	req = httptest.NewRequest(http.MethodPost, "/contests/c1/rankings/recalculate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "c1"})
	rec = httptest.NewRecorder()
	server.recalculateRankings(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 6. publishRankingSnapshot
	req = httptest.NewRequest(http.MethodPost, "/contests/c1/rankings/publish", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "c1"})
	rec = httptest.NewRecorder()
	server.publishRankingSnapshot(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 7. listRankingSnapshots
	req = httptest.NewRequest(http.MethodGet, "/contests/c1/rankings/snapshots", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "c1"})
	rec = httptest.NewRecorder()
	server.listRankingSnapshots(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 8. getRankingSnapshot
	req = httptest.NewRequest(http.MethodGet, "/contests/c1/rankings/snapshots/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "c1", "version": "1"})
	rec = httptest.NewRecorder()
	server.getRankingSnapshot(rec, req)
	r.Equal(http.StatusOK, rec.Code)
}

func TestRankingRoutesArePublic(t *testing.T) {
	server := NewServer(&mockRankingsService{
		listRankingsFn: func(context.Context) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r-public", Score: 10}}, nil
		},
		listRankingsByContestFn: func(context.Context, string) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r-public", Score: 10}}, nil
		},
		getRankingFn: func(_ context.Context, id string) (*model.Ranking, error) {
			return &model.Ranking{Id: id, Score: 10}, nil
		},
		listRankingSnapshotsFn: func(context.Context, string) ([]model.RankingSnapshot, error) {
			return []model.RankingSnapshot{{Id: "snap-1", Version: 1, PublishedAt: time.Now()}}, nil
		},
		getPublishedRankingsFn: func(context.Context, string, ...int) (*model.RankingSnapshot, error) {
			return &model.RankingSnapshot{Id: "snap-1", Version: 1, PublishedAt: time.Now()}, nil
		},
	})

	for _, path := range []string{
		"/api/v1/rankings",
		"/api/v1/rankings/r-public",
		"/api/v1/contests/c1/rankings",
		"/api/v1/contests/c1/rankings/snapshots",
		"/api/v1/contests/c1/rankings/snapshots/1",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, path)
	}
}
