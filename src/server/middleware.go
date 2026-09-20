package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var RoutePermissions = map[string]map[string]string{
	"/api/v1/me/agents": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/users": {
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.AdminPermission,
	},
	"/api/v1/users/{id}": {
		http.MethodGet:   common.ReadPermission,
		http.MethodPut:   common.AdminPermission,
		http.MethodPatch: common.AdminPermission,
	},
	"/api/v1/participants": {
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.AdminPermission,
	},
	"/api/v1/participants/{id}": {
		http.MethodGet:   common.ReadPermission,
		http.MethodPut:   common.AdminPermission,
		http.MethodPatch: common.AdminPermission,
	},
	"/api/v1/contests": {
		http.MethodPost: common.AdminPermission,
	},
	"/api/v1/contests/{id}": {
		http.MethodGet:   common.ReadPermission,
		http.MethodPut:   common.AdminPermission,
		http.MethodPatch: common.AdminPermission,
	},
	"/api/v1/contests/{id}/agents": {
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.SubmitAgentPermission,
	},
	"/api/v1/contests/{id}/entries": {
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.SubmitAgentPermission,
	},
	"/api/v1/categories": {
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.AdminPermission,
	},
	"/api/v1/categories/{id}": {
		http.MethodGet:   common.ReadPermission,
		http.MethodPut:   common.AdminPermission,
		http.MethodPatch: common.AdminPermission,
	},
	"/api/v1/games": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/games/{id}": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/agents": {
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.SubmitAgentPermission,
	},
	"/api/v1/agents/{id}": {
		http.MethodGet:   common.ReadPermission,
		http.MethodPut:   common.SubmitAgentPermission,
		http.MethodPatch: common.AdminPermission,
	},
	"/api/v1/submissions": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/submissions/upload": {
		http.MethodPost: common.SubmitAgentPermission,
	},
	"/api/v1/submissions/{id}": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/matches": {
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.ExecuteMatchPermission,
	},
	"/api/v1/matches/{id}": {
		http.MethodGet:   common.ReadPermission,
		http.MethodPut:   common.AdminPermission,
		http.MethodPatch: common.AdminPermission,
	},
	"/api/v1/matches/{id}/run": {
		http.MethodPost: common.ExecuteMatchPermission,
	},
	"/api/v1/results": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/results/{id}": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/rankings": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/rankings/{id}": {
		http.MethodGet: common.ReadPermission,
	},
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "X-Requested-With, Content-Type, Authorization, X-Request-Id")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			tracer.DebugEvent(ctx, tracer.ScopeAuth, "auth.credentials.missing", "Solicitud sin credenciales")
			common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, fmt.Sprintf("%s ", auth.TokenType))
		if tokenString == authHeader {
			tracer.DebugEvent(ctx, tracer.ScopeAuth, "auth.credentials.invalid", "Formato de credencial inválido")
			common.WriteErrorResponse(w, common.INVALID_CREDENTIALS_ERROR)
			return
		}

		claims, err := s.Auth.ValidateAccessToken(tokenString)
		if err != nil {
			tracer.DebugEvent(ctx, tracer.ScopeAuth, "auth.credentials.invalid", "Credencial rechazada", tracer.Err(err))
			common.WriteErrorResponse(w, common.INVALID_CREDENTIALS_ERROR)
			return
		}

		actorID := claims.UserID
		if actorID == "" {
			actorID = claims.ParticipantId
		}
		ctx = context.WithValue(ctx, common.UserContextKey, claims)
		ctx = tracer.WithActorID(ctx, actorID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) permissionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requiredPermission := getRequiredPermission(r)

		if requiredPermission == common.InvalidPermission {
			next.ServeHTTP(w, r)
			return
		}

		claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
		if !ok {
			tracer.FailRequest(ctx, tracer.ScopeAuth, "auth.context.missing", "No se pudo comprobar la autorización")
			common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
			return
		}

		actorID := claims.UserID
		if actorID == "" {
			actorID = claims.ParticipantId
		}

		hasPermission, err := s.Service.HasPermission(ctx, actorID, requiredPermission)
		if err != nil {
			tracer.FailRequest(ctx, tracer.ScopeAuth, "auth.permission.failed", "No se pudo comprobar el permiso",
				tracer.String("permission", requiredPermission), tracer.Err(err))
			common.WriteErrorResponse(w, common.MISSING_PERMISSION_ERROR)
			return
		}
		if !hasPermission {
			tracer.DebugEvent(ctx, tracer.ScopeAuth, "auth.permission.denied", "Permiso denegado",
				tracer.String("permission", requiredPermission))
			common.WriteErrorResponse(w, common.MISSING_PERMISSION_ERROR)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) correlationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId := r.Header.Get("X-Request-Id")
		if !isSafeRequestID(requestId) {
			requestId = uuid.New().String()
		}
		ctx := tracer.BeginRequest(r.Context(), requestId)
		w.Header().Set("X-Request-Id", requestId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isSafeRequestID(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}

type statusLoggingResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *statusLoggingResponseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *statusLoggingResponseWriter) Write(data []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

func (s *Server) requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusLoggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)

		ctx := r.Context()
		path := r.URL.Path
		if route := mux.CurrentRoute(r); route != nil {
			if template, err := route.GetPathTemplate(); err == nil {
				path = template
			}
		}
		tracer.CompleteRequest(ctx, r.Method, path, wrapped.statusCode, duration)
	})
}

func getRequiredPermission(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return common.InvalidPermission
	}

	template, err := route.GetPathTemplate()
	if err != nil {
		return common.InvalidPermission
	}

	if methods, exists := RoutePermissions[template]; exists {
		if perm, ok := methods[r.Method]; ok {
			return perm
		}
	}

	return common.InvalidPermission
}
