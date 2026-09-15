package server

import (
	"database/sql"
	"net/http"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

func (s *Server) listRankings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := r.URL.Query().Get("contest_id")

	if contestId != "" {
		rankings, err := s.Service.ListRankingsByContest(ctx, contestId)
		if err != nil {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "rankings.list.failed", "No se pudo consultar la clasificación",
				tracer.String("contest_id", contestId), tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
			return
		}
		common.WriteObjectResponse(w, http.StatusOK, rankings)
		return
	}

	rankings, err := s.Service.ListRankings(ctx)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "rankings.list.failed", "No se pudo consultar la clasificación", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, rankings)
}

func (s *Server) getRanking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	ranking, err := s.Service.GetRanking(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Ranking not found")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "ranking.get.failed", "No se pudo consultar la posición", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}
	common.WriteObjectResponse(w, http.StatusOK, ranking)
}
