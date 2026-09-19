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

type mockGamesService struct {
	service.Service
	listGamesFn func(ctx context.Context) ([]model.Game, error)
	getGameFn   func(ctx context.Context, id string) (*model.Game, error)
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

func TestServerGamesHandlers(t *testing.T) {
	r := require.New(t)

	mockSvc := &mockGamesService{
		listGamesFn: func(ctx context.Context) ([]model.Game, error) {
			return []model.Game{{Id: "starfighter", Name: "Starfighter Arena"}}, nil
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
	req = httptest.NewRequest(http.MethodGet, "/games/starfighter", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "starfighter"})
	rec = httptest.NewRecorder()
	server.getGame(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 3. getGame (not found)
	req = httptest.NewRequest(http.MethodGet, "/games/not-found", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-found"})
	rec = httptest.NewRecorder()
	server.getGame(rec, req)
	r.Equal(http.StatusNotFound, rec.Code)

}
