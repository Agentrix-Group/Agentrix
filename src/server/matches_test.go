package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type mockMatchesService struct {
	service.Service
	listMatchesFn          func(ctx context.Context) ([]model.Match, error)
	listMatchesByContestFn func(ctx context.Context, contestId string) ([]model.Match, error)
	getMatchFn             func(ctx context.Context, id string) (*model.Match, error)
	createMatchFn          func(ctx context.Context, match *model.Match, submissionIds []string) error
	runMatchFn             func(ctx context.Context, matchId string) error
	updateMatchFn          func(ctx context.Context, match *model.Match) error
	activateMatchFn        func(ctx context.Context, id string, isActive bool) error
}

func (m *mockMatchesService) ListMatches(ctx context.Context) ([]model.Match, error) {
	if m.listMatchesFn != nil {
		return m.listMatchesFn(ctx)
	}
	return nil, nil
}

func (m *mockMatchesService) ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error) {
	if m.listMatchesByContestFn != nil {
		return m.listMatchesByContestFn(ctx, contestId)
	}
	return nil, nil
}

func (m *mockMatchesService) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	if m.getMatchFn != nil {
		return m.getMatchFn(ctx, id)
	}
	return &model.Match{Id: id, GameId: "starfighter"}, nil
}

func (m *mockMatchesService) CreateMatch(ctx context.Context, match *model.Match, submissionIds []string) error {
	if m.createMatchFn != nil {
		return m.createMatchFn(ctx, match, submissionIds)
	}
	return nil
}

func (m *mockMatchesService) RunMatch(ctx context.Context, matchId string) error {
	if m.runMatchFn != nil {
		return m.runMatchFn(ctx, matchId)
	}
	return nil
}

func (m *mockMatchesService) UpdateMatch(ctx context.Context, match *model.Match) error {
	if m.updateMatchFn != nil {
		return m.updateMatchFn(ctx, match)
	}
	return nil
}

func (m *mockMatchesService) ActivateMatch(ctx context.Context, id string, isActive bool) error {
	if m.activateMatchFn != nil {
		return m.activateMatchFn(ctx, id, isActive)
	}
	return nil
}

func TestServerMatchesHandlers(t *testing.T) {
	r := require.New(t)

	mockSvc := &mockMatchesService{
		listMatchesFn: func(ctx context.Context) ([]model.Match, error) {
			return []model.Match{{Id: "m1", GameId: "starfighter"}}, nil
		},
		listMatchesByContestFn: func(ctx context.Context, contestId string) ([]model.Match, error) {
			return []model.Match{{Id: "m1", ContestId: contestId}}, nil
		},
		getMatchFn: func(ctx context.Context, id string) (*model.Match, error) {
			if id == "not-found" {
				return nil, sql.ErrNoRows
			}
			return &model.Match{Id: id, GameId: "starfighter"}, nil
		},
	}

	server := &Server{Service: mockSvc}

	// 1. listMatches (all)
	req := httptest.NewRequest(http.MethodGet, "/matches", nil)
	rec := httptest.NewRecorder()
	server.listMatches(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 2. listMatches (by contest)
	req = httptest.NewRequest(http.MethodGet, "/matches?contest_id=c1", nil)
	rec = httptest.NewRecorder()
	server.listMatches(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 3. getMatch (found)
	req = httptest.NewRequest(http.MethodGet, "/matches/m1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "m1"})
	rec = httptest.NewRecorder()
	server.getMatch(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 4. getMatch (not found)
	req = httptest.NewRequest(http.MethodGet, "/matches/not-found", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-found"})
	rec = httptest.NewRecorder()
	server.getMatch(rec, req)
	r.Equal(http.StatusNotFound, rec.Code)

	// 5. createMatch (valid)
	body, _ := json.Marshal(CreateMatchRequest{ContestId: "c1", GameId: "starfighter", SubmissionIds: []string{"sub-1"}})
	req = httptest.NewRequest(http.MethodPost, "/matches", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.createMatch(rec, req)
	r.Equal(http.StatusCreated, rec.Code)

	// 6. createMatch (missing fields)
	body, _ = json.Marshal(CreateMatchRequest{GameId: ""})
	req = httptest.NewRequest(http.MethodPost, "/matches", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.createMatch(rec, req)
	r.Equal(http.StatusBadRequest, rec.Code)

	// 7. runMatch
	req = httptest.NewRequest(http.MethodPost, "/matches/m1/run", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "m1"})
	rec = httptest.NewRecorder()
	server.runMatch(rec, req)
	r.Equal(http.StatusAccepted, rec.Code)

	// 8. updateMatch
	body, _ = json.Marshal(model.Match{GameId: "starfighter"})
	req = httptest.NewRequest(http.MethodPut, "/matches/m1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "m1"})
	rec = httptest.NewRecorder()
	server.updateMatch(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 9. activateMatch
	req = httptest.NewRequest(http.MethodPatch, "/matches/m1?status=true", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "m1"})
	rec = httptest.NewRecorder()
	server.activateMatch(rec, req)
	r.Equal(http.StatusOK, rec.Code)
}

func TestMatchReadRoutesArePublic(t *testing.T) {
	server := NewServer(&mockMatchesService{
		listMatchesFn: func(context.Context) ([]model.Match, error) {
			return []model.Match{{Id: "m-public", GameId: "starfighter"}}, nil
		},
		getMatchFn: func(_ context.Context, id string) (*model.Match, error) {
			return &model.Match{Id: id, GameId: "starfighter"}, nil
		},
	})

	for _, path := range []string{"/api/v1/matches", "/api/v1/matches/m-public"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, path)
	}
}
