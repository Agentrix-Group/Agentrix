package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

func (s *Server) listGames(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	games, err := s.Service.ListGames(ctx)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "games.list.failed", "No se pudieron consultar los juegos", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, games)
}

func (s *Server) getGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	game, err := s.Service.GetGame(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Game not found")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "game.get.failed", "No se pudo consultar el juego", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, game)
}

func (s *Server) createGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var game model.Game
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	if err := validateGame(&game); err != nil {
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	if err := s.Service.CreateGame(ctx, &game); err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "game.create.failed", "No se pudo crear el juego", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusCreated, fmt.Sprintf("Game %s created successfully", game.Id))
}

func (s *Server) updateGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var game model.Game
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}
	game.Id = id

	if err := s.Service.UpdateGame(ctx, &game); err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "game.update.failed", "No se pudo actualizar el juego", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Game %s updated successfully", id))
}

func (s *Server) activateGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	isActive, _ := strconv.ParseBool(r.URL.Query().Get("status"))

	if err := s.Service.ActivateGame(ctx, id, isActive); err != nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Game not found")
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Game %s status updated", id))
}
