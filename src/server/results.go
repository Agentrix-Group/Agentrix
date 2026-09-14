package server

import (
	"database/sql"
	"net/http"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

func (s *Server) listResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	matchId := r.URL.Query().Get("match_id")

	if matchId != "" {
		results, err := s.Service.ListResultsByMatch(ctx, matchId)
		if err != nil {
			tracer.Errorf(ctx, "Failed to list results for match %s: %s", matchId, err)
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
			return
		}
		common.WriteObjectResponse(w, http.StatusOK, results)
		return
	}

	results, err := s.Service.ListResults(ctx)
	if err != nil {
		tracer.Errorf(ctx, "Failed to list results: %s", err)
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, results)
}

func (s *Server) getResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	result, err := s.Service.GetResult(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Result not found")
		} else {
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, result)
}
