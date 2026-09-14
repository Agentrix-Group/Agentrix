package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/F4nk1/Agentrix/src/auth"
	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var RoutePermissions = map[string]map[string]string{
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
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.AdminPermission,
	},
	"/api/v1/games/{id}": {
		http.MethodGet:   common.ReadPermission,
		http.MethodPut:   common.AdminPermission,
		http.MethodPatch: common.AdminPermission,
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
		http.MethodGet:  common.ReadPermission,
		http.MethodPost: common.SubmitAgentPermission,
	},
	"/api/v1/submissions/{id}": {
		http.MethodGet:   common.ReadPermission,
		http.MethodPut:   common.SubmitAgentPermission,
		http.MethodPatch: common.AdminPermission,
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
	"/api/v1/replays/{id}": {
		http.MethodGet: common.ReadPermission,
	},
	"/api/v1/replays/{id}/stream": {
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
			tracer.Warnf(ctx, "Missing authorization header")
			common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, fmt.Sprintf("%s ", auth.TokenType))
		if tokenString == authHeader {
			tracer.Warnf(ctx, "Invalid token format")
			common.WriteErrorResponse(w, common.INVALID_CREDENTIALS_ERROR)
			return
		}

		claims, err := s.Auth.ValidateAccessToken(tokenString)
		if err != nil {
			tracer.Warnf(ctx, "Invalid access token: %s", err)
			common.WriteErrorResponse(w, common.INVALID_CREDENTIALS_ERROR)
			return
		}

		ctx = context.WithValue(ctx, common.UserContextKey, claims)
		ctx = context.WithValue(ctx, tracer.ParticipantIdKey, claims.ParticipantId)
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
			tracer.Warnf(ctx, "Missing claims in context")
			common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
			return
		}

		// Admins bypass permission check
		if claims.RoleId == common.RoleAdmin {
			next.ServeHTTP(w, r)
			return
		}

		hasPermission, err := s.Service.HasPermission(ctx, claims.ParticipantId, requiredPermission)
		if err != nil || !hasPermission {
			tracer.Warnf(ctx, "Permission '%s' denied for participant '%s'", requiredPermission, claims.ParticipantId)
			common.WriteErrorResponse(w, common.MISSING_PERMISSION_ERROR)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) correlationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId := r.Header.Get("X-Request-Id")
		if requestId == "" {
			requestId = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), tracer.RequestIdKey, requestId)
		w.Header().Set("X-Request-Id", requestId)
		next.ServeHTTP(w, r.WithContext(ctx))
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
