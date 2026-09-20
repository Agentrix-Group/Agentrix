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
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerId := r.URL.Query().Get("owner_user_id")
	if ownerId == "" {
		ownerId = r.URL.Query().Get("participant_id")
	}

	var agents []model.Agent
	var err error
	if ownerId != "" {
		agents, err = s.Service.ListAgentsByOwner(ctx, ownerId)
	} else {
		agents, err = s.Service.ListAgents(ctx)
	}

	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "agents.list.failed", "No se pudieron consultar los agentes", tracer.Err(err))
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
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "agent.get.failed", "No se pudo consultar el agente", tracer.Err(err))
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
		common.WriteDiagnosticError(w, ctx, common.INVALID_REQUEST_ERROR, "Invalid request format", "Malformed JSON body")
		return
	}

	// Derive authenticated user
	var authUserID string
	var authRole string
	if claims, ok := ctx.Value(common.UserContextKey).(*model.Claims); ok {
		authUserID = claims.UserID
		if authUserID == "" {
			authUserID = claims.ParticipantId
		}
		authRole = claims.RoleId
	}

	// Normalize owner_user_id
	if agent.OwnerUserId == "" {
		agent.OwnerUserId = agent.ParticipantId
	}
	if agent.OwnerUserId == "" {
		agent.OwnerUserId = authUserID
	} else if agent.OwnerUserId != authUserID && authRole != common.RoleAdmin && authUserID != "" {
		tracer.FailRequest(ctx, tracer.ScopeAuth, "agent.create.denied", "Usuario no autorizado para crear agentes de otros")
		common.WriteDiagnosticError(w, ctx, common.MISSING_PERMISSION_ERROR, "Forbidden", "Cannot create agent for another user")
		return
	}
	agent.ParticipantId = agent.OwnerUserId

	if err := validateAgent(&agent); err != nil {
		if agent.Name == "" || agent.GameId == "" {
			common.WriteDiagnosticError(w, ctx, common.MISSING_FIELDS_ERROR, "Required fields are missing", err.Error())
		} else {
			common.WriteDiagnosticError(w, ctx, common.INVALID_REQUEST_ERROR, err.Error(), err.Error())
		}
		return
	}

	if err := s.Service.CreateAgent(ctx, &agent); err != nil {
		var dbErr *repository.DBError
		sqlState := ""
		constraint := ""
		if errors.As(err, &dbErr) {
			sqlState = dbErr.SQLState
			constraint = dbErr.ConstraintName
		}

		fields := []tracer.Field{
			tracer.Operation("agent.create"),
			tracer.String("agent_name", agent.Name),
			tracer.String("game_id", agent.GameId),
		}
		if sqlState != "" {
			fields = append(fields, tracer.SQLState(sqlState))
		}
		if constraint != "" {
			fields = append(fields, tracer.Constraint(constraint))
		}

		switch {
		case errors.Is(err, service.ErrUnsupportedGame):
			fields = append(fields, tracer.Origin(tracer.OriginGame))
			tracer.FailRequest(ctx, tracer.ScopeAgent, "agent.create.rejected", "Juego no soportado", fields...)
			common.WriteDiagnosticError(w, ctx, common.GAME_NOT_FOUND_ERROR, "Game is not supported", err.Error())
			return

		case errors.Is(err, service.ErrGameNotFound):
			fields = append(fields, tracer.Origin(tracer.OriginInfrastructure))
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "agent.create.failed", "Juego no encontrado en la base de datos", fields...)
			common.WriteDiagnosticError(w, ctx, common.GAME_NOT_FOUND_ERROR, "Game not found in database", fmt.Sprintf("Constraint %s violated: game not found in database", constraint))
			return

		case errors.Is(err, service.ErrUserNotFound), errors.Is(err, service.ErrParticipantNotFound):
			fields = append(fields, tracer.Origin(tracer.OriginPlatform))
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "agent.create.failed", "Propietario del agente no existe", fields...)
			common.WriteDiagnosticError(w, ctx, common.NOT_FOUND_ERROR, "Agent owner user not found", fmt.Sprintf("Constraint %s violated: user does not exist", constraint))
			return

		case errors.Is(err, service.ErrAgentAlreadyExists):
			fields = append(fields, tracer.Origin(tracer.OriginPlatform))
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "agent.create.failed", "Agente ya existe", fields...)
			common.WriteDiagnosticError(w, ctx, common.ALREADY_EXISTS_ERROR, "Agent already exists", fmt.Sprintf("Constraint %s violated: duplicate entry", constraint))
			return

		case errors.Is(err, service.ErrSchemaIncompatible):
			fields = append(fields, tracer.Origin(tracer.OriginInfrastructure))
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "agent.create.failed", "Esquema de base de datos desactualizado", fields...)
			common.WriteDiagnosticError(w, ctx, common.DATABASE_ERROR, "Database schema is incompatible", "Database tables or columns missing; pending migrations required")
			return

		default:
			fields = append(fields, tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "agent.create.failed", "No se pudo crear el agente", fields...)
			common.WriteDiagnosticError(w, ctx, common.DATABASE_ERROR, "Database operation failed", err.Error())
			return
		}
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
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "agent.update.failed", "No se pudo actualizar el agente", tracer.Err(err))
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
