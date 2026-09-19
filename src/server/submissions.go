package server

import (
	"database/sql"
	"errors"
	"io"
	"net/http"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

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
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "submissions.list.failed", "No se pudieron consultar los envíos", tracer.Err(err))
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
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "submission.get.failed", "No se pudo consultar el envío", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, submission)
}

func (s *Server) uploadSubmissionBundle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if !ok || claims == nil {
		common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 3<<20)
	if err := r.ParseMultipartForm(3 << 20); err != nil {
		common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, "ZIP upload exceeds the 2 MiB limit or is malformed")
		return
	}
	agentID := r.FormValue("agent_id")
	if agentID == "" {
		common.WriteErrorMessage(w, common.MISSING_FIELDS_ERROR, "agent_id is required")
		return
	}
	file, header, err := r.FormFile("bundle")
	if err != nil {
		common.WriteErrorMessage(w, common.MISSING_FIELDS_ERROR, "bundle ZIP is required")
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > 2<<20 {
		common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, "bundle ZIP must not exceed 2 MiB")
		return
	}
	archive, err := io.ReadAll(io.LimitReader(file, (2<<20)+1))
	if err != nil || len(archive) > 2<<20 {
		common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, "could not read bundle ZIP")
		return
	}
	submission, err := s.Service.CreateSubmissionBundle(ctx, claims.ParticipantId, claims.RoleId, agentID, archive)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAgentNotFound):
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Agent not found")
		case errors.Is(err, service.ErrAgentNotOwned):
			common.WriteErrorMessage(w, common.MISSING_PERMISSION_ERROR, "You are not authorized to submit for this agent")
		case errors.Is(err, service.ErrInvalidBotBundle), errors.Is(err, service.ErrAdmissionFailed):
			common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, err.Error())
		default:
			tracer.FailRequest(ctx, tracer.ScopeArtifact, "submission.bundle.failed", "No se pudo admitir el ZIP del bot", tracer.Err(err))
			common.WriteErrorResponse(w, common.INTERNAL_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusCreated, submission)
}
