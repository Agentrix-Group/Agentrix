package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

type CreateMatchRequest struct {
	ContestId     string   `json:"contest_id"`
	GameId        string   `json:"game_id"`
	SubmissionIds []string `json:"submission_ids"`
	Seed          int64    `json:"seed,omitempty"`
}

func (s *Server) listMatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := r.URL.Query().Get("contest_id")

	var matches []model.Match
	var err error
	if contestId != "" {
		matches, err = s.Service.ListMatchesByContest(ctx, contestId)
	} else {
		matches, err = s.Service.ListMatches(ctx)
	}

	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "matches.list.failed", "No se pudieron consultar las partidas", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, matches)
}

func (s *Server) getMatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	match, err := s.Service.GetMatch(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Match not found")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "match.get.failed", "No se pudo consultar la partida", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, match)
}

func (s *Server) createMatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreateMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	match := model.Match{
		ContestId: req.ContestId,
		GameId:    req.GameId,
		Seed:      req.Seed,
	}

	if err := validateMatch(&match); err != nil {
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	resp, err := s.Service.CreateMatch(ctx, &match, req.SubmissionIds)
	if errors.Is(err, service.ErrInvalidParticipants) {
		common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, err.Error())
		return
	}
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "match.create.failed", "No se pudo crear la partida", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteObjectResponse(w, http.StatusCreated, resp)
}

func (s *Server) runMatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	idempotencyKey := r.Header.Get("Idempotency-Key")

	resp, err := s.Service.RunMatch(ctx, id, idempotencyKey)
	if err != nil {
		if errors.Is(err, service.ErrMatchNotFound) {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Match not found")
			return
		}
		if errors.Is(err, service.ErrInvalidMatchState) || errors.Is(err, service.ErrActiveMatchCannotRerun) {
			common.WriteErrorMessage(w, common.ALREADY_EXISTS_ERROR, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidSubmissions) {
			common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, err.Error())
			return
		}
		tracer.FailRequest(ctx, tracer.ScopeQueue, "match.enqueue.failed", "No se pudo encolar la partida", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	statusCode := http.StatusAccepted
	if resp.HttpStatusCode != 0 {
		statusCode = resp.HttpStatusCode
	}
	common.WriteObjectResponse(w, statusCode, resp)
}

func (s *Server) updateMatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var match model.Match
	if err := json.NewDecoder(r.Body).Decode(&match); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}
	match.Id = id

	if err := s.Service.UpdateMatch(ctx, &match); err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "match.update.failed", "No se pudo actualizar la partida", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Match %s updated successfully", id))
}

func (s *Server) activateMatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	isActive, _ := strconv.ParseBool(r.URL.Query().Get("status"))

	if err := s.Service.ActivateMatch(ctx, id, isActive); err != nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Match not found")
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Match %s status updated", id))
}
