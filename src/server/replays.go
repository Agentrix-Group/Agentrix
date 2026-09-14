package server

import (
	"net/http"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

func (s *Server) getReplay(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	replay, err := s.Service.GetReplay(ctx, id)
	if err != nil {
		tracer.Warnf(ctx, "Replay %s not found: %s", id, err)
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Replay not found")
		return
	}

	common.WriteObjectResponse(w, http.StatusOK, replay)
}

func (s *Server) streamReplay(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	data, err := s.Service.StreamReplay(ctx, id)
	if err != nil {
		tracer.Warnf(ctx, "Failed to stream replay %s: %s", id, err)
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Replay not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
