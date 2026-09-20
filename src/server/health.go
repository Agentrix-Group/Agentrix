package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

var serverStartTime = time.Now()

func (s *Server) livenessProbe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":         "UP",
		"uptime_seconds": int64(time.Since(serverStartTime).Seconds()),
	})
}

func (s *Server) readinessProbe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	checks, err := s.Service.CheckReadiness(ctx)
	if err != nil {
		tracer.WarnEvent(ctx, tracer.ScopeSystem, "readiness.failed", "Readiness probe failed",
			tracer.String("error", err.Error()),
			tracer.Origin(tracer.OriginInfrastructure),
		)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "NOT_READY",
			"error":  err.Error(),
			"checks": checks,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "READY",
		"checks": checks,
	})
}
