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

func (s *Server) listContests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contests, err := s.Service.ListContests(ctx)
	if err != nil {
		tracer.Errorf(ctx, "Failed to list contests: %s", err)
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
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, contest)
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
