package server

import (
	"database/sql"
	"net/http"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

func (s *Server) listGames(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	games, err := s.Service.ListGames(ctx)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "games.list.failed", "No se pudo consultar Starfighter", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, games)
}

func (s *Server) getGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	if id != "starfighter" {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Game not found")
		return
	}
	game, err := s.Service.GetGame(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Game not found")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "game.get.failed", "No se pudo consultar Starfighter", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, game)
}
