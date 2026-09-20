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

func (s *Server) listPublicContests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stateParam := r.URL.Query().Get("state")
	includeArchivedParam := r.URL.Query().Get("include_archived")

	var includeArchived bool
	if includeArchivedParam != "" {
		parsed, err := strconv.ParseBool(includeArchivedParam)
		if err != nil {
			common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, "Parameter 'include_archived' must be a boolean (true or false)")
			return
		}
		includeArchived = parsed
	}

	filter := model.PublicContestsFilter{
		State:           model.ContestState(stateParam),
		IncludeArchived: includeArchived,
	}

	contests, err := s.Service.ListPublicContests(ctx, filter)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStateFilter) || errors.Is(err, service.ErrPrivateStateFilter) {
			common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, err.Error())
			return
		}
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "contests.public_list.failed", "No se pudieron consultar los concursos", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	if contests == nil {
		contests = make([]model.PublicContestSummary, 0)
	}

	common.WriteObjectResponse(w, http.StatusOK, contests)
}

func (s *Server) listContests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contests, err := s.Service.ListContests(ctx)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "contests.list.failed", "No se pudieron consultar los concursos", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, contests)
}

func (s *Server) getContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	contest, err := s.Service.GetContest(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Contest not found")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "contest.get.failed", "No se pudo consultar el concurso", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, contest)
}

func (s *Server) getPublicContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	contest, err := s.Service.GetPublicContest(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrContestNotFound) || err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Contest not found")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "contest.public_get.failed", "No se pudo consultar el concurso",
				tracer.String("contest_id", id), tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}

	common.WriteObjectResponse(w, http.StatusOK, contest)
}

func (s *Server) enrollAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := mux.Vars(r)["id"]

	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if !ok || claims == nil {
		common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
		return
	}

	var req model.EnrollAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AgentId == "" {
		common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, "Field 'agent_id' is required")
		return
	}

	userID := claims.UserID

	entry, ranking, err := s.Service.EnrollAgent(ctx, userID, contestId, req.AgentId)
	if err != nil {
		if errors.Is(err, service.ErrContestNotFound) || errors.Is(err, service.ErrAgentNotFound) {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, err.Error())
			return
		}
		if errors.Is(err, service.ErrUnauthorizedAgent) {
			common.WriteErrorMessage(w, common.MISSING_PERMISSION_ERROR, err.Error())
			return
		}
		if errors.Is(err, service.ErrRegistrationClosed) || errors.Is(err, service.ErrGameMismatch) {
			common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, err.Error())
			return
		}
		if errors.Is(err, service.ErrAgentAlreadyEnrolled) {
			common.WriteErrorMessage(w, common.ALREADY_EXISTS_ERROR, err.Error())
			return
		}
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "contest.enrollment.failed", "No se pudo registrar el agente en el concurso",
			tracer.String("agent_id", req.AgentId), tracer.String("contest_id", contestId), tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteObjectResponse(w, http.StatusCreated, model.EnrollAgentResponse{
		HttpStatusCode: http.StatusCreated,
		Message:        "Agent enrolled in contest successfully",
		Entry:          entry,
		Ranking:        ranking,
	})
}

func (s *Server) listContestEntries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := mux.Vars(r)["id"]

	entries, err := s.Service.ListContestEntries(ctx, contestId)
	if err != nil {
		if errors.Is(err, service.ErrContestNotFound) || err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Contest not found")
			return
		}
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "contest.entries.failed", "No se pudieron consultar las inscripciones del concurso",
			tracer.String("contest_id", contestId), tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	if entries == nil {
		entries = make([]model.ContestEntry, 0)
	}
	common.WriteObjectResponse(w, http.StatusOK, entries)
}

func (s *Server) listContestAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := mux.Vars(r)["id"]

	rankings, err := s.Service.ListContestAgents(ctx, contestId)
	if err != nil {
		if errors.Is(err, service.ErrContestNotFound) || err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Contest not found")
			return
		}
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "contest.agents.failed", "No se pudieron consultar los agentes del concurso",
			tracer.String("contest_id", contestId), tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	if rankings == nil {
		rankings = make([]model.Ranking, 0)
	}
	common.WriteObjectResponse(w, http.StatusOK, rankings)
}

func (s *Server) createContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var contest model.Contest
	if err := json.NewDecoder(r.Body).Decode(&contest); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	if err := validateContest(&contest); err != nil {
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	if err := s.Service.CreateContest(ctx, &contest); err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "contest.create.failed", "No se pudo crear el concurso", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusCreated, fmt.Sprintf("Contest %s created successfully", contest.Id))
}

func (s *Server) updateContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var contest model.Contest
	if err := json.NewDecoder(r.Body).Decode(&contest); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}
	contest.Id = id

	if err := s.Service.UpdateContest(ctx, &contest); err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "contest.update.failed", "No se pudo actualizar el concurso", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Contest %s updated successfully", id))
}

func (s *Server) activateContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	isActive, _ := strconv.ParseBool(r.URL.Query().Get("status"))

	if err := s.Service.ActivateContest(ctx, id, isActive); err != nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Contest not found")
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Contest %s status updated", id))
}

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categories, err := s.Service.ListCategories(ctx)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "categories.list.failed", "No se pudieron consultar las categorías", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, categories)
}

func (s *Server) getCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	category, err := s.Service.GetCategory(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Category not found")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "category.get.failed", "No se pudo consultar la categoría", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, category)
}

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var category model.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	if err := validateCategory(&category); err != nil {
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	if err := s.Service.CreateCategory(ctx, &category); err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "category.create.failed", "No se pudo crear la categoría", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusCreated, fmt.Sprintf("Category %s created successfully", category.Id))
}

func (s *Server) updateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var category model.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}
	category.Id = id

	if err := s.Service.UpdateCategory(ctx, &category); err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "category.update.failed", "No se pudo actualizar la categoría", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Category %s updated successfully", id))
}

func (s *Server) activateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	isActive, _ := strconv.ParseBool(r.URL.Query().Get("status"))

	if err := s.Service.ActivateCategory(ctx, id, isActive); err != nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Category not found")
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Category %s status updated", id))
}
