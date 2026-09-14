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

type CreateSubmissionRequest struct {
	AgentId  string `json:"agent_id"`
	Version  int    `json:"version"`
	Language string `json:"language"`
	Code     string `json:"code"`
}

func (s *Server) listSubmissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agentId := r.URL.Query().Get("agent_id")

	var submissions []model.Submission
	var err error
	if agentId != "" {
		submissions, err = s.Service.ListSubmissionsByAgent(ctx, agentId)
	} else {
		submissions, err = s.Service.ListSubmissions(ctx)
	}

	if err != nil {
		tracer.Errorf(ctx, "Failed to list submissions: %s", err)
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, submissions)
}

func (s *Server) getSubmission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	submission, err := s.Service.GetSubmission(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Submission not found")
		} else {
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, submission)
}

func (s *Server) createSubmission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreateSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	submission := model.Submission{
		AgentId:  req.AgentId,
		Version:  req.Version,
		Language: req.Language,
		Status:   common.SubmissionStatusReady,
	}

	if err := validateSubmission(&submission); err != nil {
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	if err := s.Service.CreateSubmission(ctx, &submission, []byte(req.Code)); err != nil {
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusCreated, fmt.Sprintf("Submission %s created successfully", submission.Id))
}

func (s *Server) updateSubmission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var submission model.Submission
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}
	submission.Id = id

	if err := s.Service.UpdateSubmission(ctx, &submission); err != nil {
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Submission %s updated successfully", id))
}

func (s *Server) activateSubmission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	isActive, _ := strconv.ParseBool(r.URL.Query().Get("status"))

	if err := s.Service.ActivateSubmission(ctx, id, isActive); err != nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Submission not found")
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Submission %s status updated", id))
}
