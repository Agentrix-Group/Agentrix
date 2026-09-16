package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type mockGamesService struct {
	service.Service
	listGamesFn    func(ctx context.Context) ([]model.Game, error)
	getGameFn      func(ctx context.Context, id string) (*model.Game, error)
	createGameFn   func(ctx context.Context, game *model.Game) error
	updateGameFn   func(ctx context.Context, game *model.Game) error
	activateGameFn func(ctx context.Context, id string, isActive bool) error
}

func (m *mockGamesService) ListGames(ctx context.Context) ([]model.Game, error) {
	if m.listGamesFn != nil {
		return m.listGamesFn(ctx)
	}
	return nil, nil
}

func (m *mockGamesService) GetGame(ctx context.Context, id string) (*model.Game, error) {
	if m.getGameFn != nil {
		return m.getGameFn(ctx, id)
	}
	return &model.Game{Id: id, Name: "TestGame"}, nil
}

func (m *mockGamesService) CreateGame(ctx context.Context, game *model.Game) error {
	if m.createGameFn != nil {
		return m.createGameFn(ctx, game)
	}
	return nil
}

func (m *mockGamesService) UpdateGame(ctx context.Context, game *model.Game) error {
	if m.updateGameFn != nil {
		return m.updateGameFn(ctx, game)
	}
	return nil
}

func (m *mockGamesService) ActivateGame(ctx context.Context, id string, isActive bool) error {
	if m.activateGameFn != nil {
		return m.activateGameFn(ctx, id, isActive)
	}
	return nil
}

func TestServerGamesHandlers(t *testing.T) {
	r := require.New(t)

	mockSvc := &mockGamesService{
		listGamesFn: func(ctx context.Context) ([]model.Game, error) {
			return []model.Game{{Id: "g1", Name: "Game1"}}, nil
		},
		getGameFn: func(ctx context.Context, id string) (*model.Game, error) {
			if id == "not-found" {
				return nil, sql.ErrNoRows
			}
			return &model.Game{Id: id, Name: "Game1"}, nil
		},
	}

	server := &Server{Service: mockSvc}

	// 1. listGames
	req := httptest.NewRequest(http.MethodGet, "/games", nil)
	rec := httptest.NewRecorder()
	server.listGames(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 2. getGame (found)
	req = httptest.NewRequest(http.MethodGet, "/games/g1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "g1"})
	rec = httptest.NewRecorder()
	server.getGame(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 3. getGame (not found)
	req = httptest.NewRequest(http.MethodGet, "/games/not-found", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-found"})
	rec = httptest.NewRecorder()
	server.getGame(rec, req)
	r.Equal(http.StatusNotFound, rec.Code)

	// 4. createGame (valid)
	body, _ := json.Marshal(model.Game{Name: "NewGame", MinPlayers: 2, MaxPlayers: 4})
	req = httptest.NewRequest(http.MethodPost, "/games", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.createGame(rec, req)
	r.Equal(http.StatusCreated, rec.Code)

	// 5. createGame (missing fields)
	body, _ = json.Marshal(model.Game{Name: ""})
	req = httptest.NewRequest(http.MethodPost, "/games", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.createGame(rec, req)
	r.Equal(http.StatusBadRequest, rec.Code)

	// 6. updateGame
	body, _ = json.Marshal(model.Game{Name: "UpdatedGame"})
	req = httptest.NewRequest(http.MethodPut, "/games/g1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "g1"})
	rec = httptest.NewRecorder()
	server.updateGame(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 7. activateGame
	req = httptest.NewRequest(http.MethodPatch, "/games/g1?status=true", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "g1"})
	rec = httptest.NewRecorder()
	server.activateGame(rec, req)
	r.Equal(http.StatusOK, rec.Code)
}
