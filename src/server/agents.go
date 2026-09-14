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

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	participantId := r.URL.Query().Get("participant_id")

	var agents []model.Agent
	var err error
	if participantId != "" {
		agents, err = s.Service.ListAgentsByParticipant(ctx, participantId)
	} else {
		agents, err = s.Service.ListAgents(ctx)
	}

	if err != nil {
		tracer.Errorf(ctx, "Failed to list agents: %s", err)
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, agents)
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	agent, err := s.Service.GetAgent(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Agent not found")
		} else {
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, agent)
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var agent model.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	// Associate with authenticated participant if not explicitly set
	if agent.ParticipantId == "" {
		if claims, ok := ctx.Value(common.UserContextKey).(*model.Claims); ok {
			agent.ParticipantId = claims.ParticipantId
		}
	}

	if err := validateAgent(&agent); err != nil {
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	if err := s.Service.CreateAgent(ctx, &agent); err != nil {
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusCreated, fmt.Sprintf("Agent %s created successfully", agent.Id))
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var agent model.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}
	agent.Id = id

	if err := s.Service.UpdateAgent(ctx, &agent); err != nil {
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Agent %s updated successfully", id))
}

func (s *Server) activateAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	isActive, _ := strconv.ParseBool(r.URL.Query().Get("status"))

	if err := s.Service.ActivateAgent(ctx, id, isActive); err != nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Agent not found")
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Agent %s status updated", id))
}
