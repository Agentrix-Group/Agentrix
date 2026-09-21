package server

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
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

func (s *Server) listContestRankings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := mux.Vars(r)["id"]

	rankings, err := s.Service.ListRankingsByContest(ctx, contestId)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "rankings.list.failed", "No se pudo consultar la clasificación del concurso",
			tracer.String("contest_id", contestId), tracer.Err(err))
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

func (s *Server) recalculateRankings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := mux.Vars(r)["id"]

	rankings, err := s.Service.RecalculateContestRankings(ctx, contestId)
	if err != nil {
		if errors.Is(err, service.ErrContestNotFound) || errors.Is(err, sql.ErrNoRows) {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Contest not found")
			return
		}
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "rankings.recalculate.failed", "Error recalculando clasificación",
			tracer.String("contest_id", contestId), tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteObjectResponse(w, http.StatusOK, rankings)
}

func (s *Server) publishRankingSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := mux.Vars(r)["id"]

	var authUserID string
	if claims, ok := ctx.Value(common.UserContextKey).(*model.Claims); ok {
		authUserID = claims.UserID
	}

	snapshot, err := s.Service.PublishRankingSnapshot(ctx, contestId, authUserID)
	if err != nil {
		if errors.Is(err, service.ErrContestNotFound) || errors.Is(err, sql.ErrNoRows) {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Contest not found")
			return
		}
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "rankings.publish.failed", "Error publicando instantánea de clasificación",
			tracer.String("contest_id", contestId), tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteObjectResponse(w, http.StatusOK, snapshot)
}

func (s *Server) listRankingSnapshots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := mux.Vars(r)["id"]

	snapshots, err := s.Service.ListRankingSnapshots(ctx, contestId)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "rankings.snapshots.failed", "Error consultando instantáneas de clasificación",
			tracer.String("contest_id", contestId), tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteObjectResponse(w, http.StatusOK, snapshots)
}

func (s *Server) getRankingSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestId := mux.Vars(r)["id"]
	versionStr := mux.Vars(r)["version"]

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, "invalid version parameter")
		return
	}

	snapshot, err := s.Service.GetPublishedRankings(ctx, contestId, version)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "rankings.snapshot.get.failed", "Error consultando instantánea de versión",
			tracer.String("contest_id", contestId), tracer.Int("version", version), tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	if snapshot == nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "snapshot not found")
		return
	}

	common.WriteObjectResponse(w, http.StatusOK, snapshot)
}
