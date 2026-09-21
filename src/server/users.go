package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

type LoginRequest = model.LoginRequest
type LoginResponse = model.LoginResponse

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	if err := validateLoginRequest(req.Username, req.Password); err != nil {
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	user, err := s.Service.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.INVALID_CREDENTIALS_ERROR, "Invalid username or password")
		} else if errors.Is(err, service.ErrAccountInactive) {
			common.WriteErrorMessage(w, common.MISSING_PERMISSION_ERROR, "User account is inactive")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "auth.login.failed", "No se pudo completar el ingreso", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}

	// Resolve dynamic capabilities from database
	var caps []string
	if s.Service != nil {
		func() {
			defer func() { _ = recover() }()
			caps, _ = s.Service.GetUserCapabilities(ctx, user.Id)
		}()
	}
	if len(caps) > 0 {
		user.Capabilities = caps
	} else {
		user.Capabilities = resolveCapabilities(user.RoleId)
	}

	var token *model.Token
	if s.SessionManager != nil {
		token, err = s.SessionManager.CreateSession(ctx, user, r.UserAgent(), extractIP(r))
	} else {
		token, err = s.Auth.GenerateAuthToken(user.Id, user.RoleId)
	}
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeAuth, "auth.token.failed", "No se pudo crear la sesión", tracer.Err(err))
		common.WriteErrorResponse(w, common.INTERNAL_ERROR)
		return
	}

	// Set HttpOnly refresh cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token.RefreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})

	user.Password = "" // Omit password hash in response
	common.WriteObjectResponse(w, http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

func resolveCapabilities(roleId string) []string {
	caps := []string{"contests:view", "matches:view", "rankings:view", "replays:view"}
	switch roleId {
	case "admin":
		return append(caps, "matches:run", "matches:schedule", "agents:create", "submissions:upload", "contests:enroll", "admin:access")
	case "organizer":
		return append(caps, "matches:run", "matches:schedule", "contests:create", "admin:access")
	case "participant", "player":
		return append(caps, "agents:create", "submissions:upload", "contests:enroll")
	default:
		return caps
	}
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	user := model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	err := s.Service.Register(ctx, &user)
	if err != nil {
		if errors.Is(err, service.ErrUsernameAlreadyExists) || errors.Is(err, service.ErrEmailAlreadyExists) {
			common.WriteErrorMessage(w, common.ALREADY_EXISTS_ERROR, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidUsername) || errors.Is(err, service.ErrInvalidEmail) || errors.Is(err, service.ErrInvalidPassword) {
			common.WriteErrorMessage(w, common.INVALID_REQUEST_ERROR, err.Error())
			return
		}
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "account.register.failed", "No se pudo registrar la cuenta", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusCreated, fmt.Sprintf("User %s registered successfully", user.Username))
}

func (s *Server) refreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	rawRefreshToken := req.RefreshToken
	if rawRefreshToken == "" {
		if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
			rawRefreshToken = cookie.Value
		}
	}

	if rawRefreshToken == "" {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	var token *model.Token
	var err error

	if s.SessionManager != nil {
		token, err = s.SessionManager.RefreshSession(ctx, rawRefreshToken, r.UserAgent(), extractIP(r))
		if err != nil {
			if errors.Is(err, auth.ErrSessionReuseDetected) {
				tracer.FailRequest(ctx, tracer.ScopeAuth, "auth.session.reuse_detected", "Reutilización de token detectada")
				common.WriteErrorMessage(w, common.ACCESS_DENIED_ERROR, "Session revoked due to token reuse")
				return
			}
			if errors.Is(err, auth.ErrAccountDisabled) {
				common.WriteErrorMessage(w, common.ACCESS_DENIED_ERROR, "Account is inactive")
				return
			}
			if errors.Is(err, auth.ErrSessionExpired) {
				common.WriteErrorMessage(w, common.INVALID_CREDENTIALS_ERROR, "Session expired")
				return
			}
			common.WriteErrorResponse(w, common.INVALID_CREDENTIALS_ERROR)
			return
		}
	} else {
		claims, err := s.Auth.ValidateRefreshToken(rawRefreshToken)
		if err != nil {
			common.WriteErrorResponse(w, common.INVALID_CREDENTIALS_ERROR)
			return
		}
		token, err = s.Auth.GenerateAuthToken(claims.UserID, claims.RoleId)
		if err != nil {
			common.WriteErrorResponse(w, common.INTERNAL_ERROR)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token.RefreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})

	common.WriteObjectResponse(w, http.StatusOK, token)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	rawRefreshToken := req.RefreshToken
	if rawRefreshToken == "" {
		if cookie, err := r.Cookie("refresh_token"); err == nil {
			rawRefreshToken = cookie.Value
		}
	}

	if s.SessionManager != nil && rawRefreshToken != "" {
		_ = s.SessionManager.RevokeSession(ctx, rawRefreshToken)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	common.WriteSuccessResponse(w, http.StatusOK, "Logged out successfully")
}

func (s *Server) logoutAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if !ok || claims == nil {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, fmt.Sprintf("%s ", auth.TokenType)) {
			tokenString := strings.TrimPrefix(authHeader, fmt.Sprintf("%s ", auth.TokenType))
			var err error
			claims, err = s.Auth.ValidateAccessToken(tokenString)
			if err != nil {
				common.WriteErrorResponse(w, common.INVALID_CREDENTIALS_ERROR)
				return
			}
		}
	}
	if claims == nil {
		common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
		return
	}

	if s.SessionManager != nil {
		_ = s.SessionManager.RevokeAllUserSessions(ctx, claims.UserID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	common.WriteSuccessResponse(w, http.StatusOK, "All sessions revoked successfully")
}

func (s *Server) checkSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if !ok {
		common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
		return
	}

	userID := claims.UserID

	user, err := s.Service.GetUser(ctx, userID)
	if err != nil {
		common.WriteErrorResponse(w, common.MISSING_PERMISSION_ERROR)
		return
	}
	if !user.Active {
		common.WriteErrorMessage(w, common.MISSING_PERMISSION_ERROR, "User account is inactive")
		return
	}

	user.Password = ""
	var caps []string
	if s.Service != nil {
		func() {
			defer func() { _ = recover() }()
			caps, _ = s.Service.GetUserCapabilities(ctx, userID)
		}()
	}
	if len(caps) > 0 {
		user.Capabilities = caps
	} else {
		user.Capabilities = resolveCapabilities(user.RoleId)
	}
	common.WriteObjectResponse(w, http.StatusOK, user)
}

func (s *Server) getMyAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if !ok {
		common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
		return
	}

	userID := claims.UserID

	agents, err := s.Service.ListAgentsByOwner(ctx, userID)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "agents.my_list.failed", "No se pudieron consultar los agentes propios", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteObjectResponse(w, http.StatusOK, agents)
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if ok {
		hasPerm, err := s.Service.HasPermission(ctx, claims.UserID, "users:read:any")
		if err != nil || !hasPerm {
			hasAdmin, _ := s.Service.HasPermission(ctx, claims.UserID, common.AdminPermission)
			if !hasAdmin {
				common.WriteErrorResponse(w, common.MISSING_PERMISSION_ERROR)
				return
			}
		}
	}

	users, err := s.Service.ListUsers(ctx)
	if err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "accounts.list.failed", "No se pudieron consultar las cuentas", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	for i := range users {
		users[i].Password = ""
	}
	common.WriteObjectResponse(w, http.StatusOK, users)
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if ok && claims.UserID != id {
		hasPerm, err := s.Service.HasPermission(ctx, claims.UserID, "users:read:any")
		if err != nil || !hasPerm {
			hasAdmin, _ := s.Service.HasPermission(ctx, claims.UserID, common.AdminPermission)
			if !hasAdmin {
				common.WriteErrorResponse(w, common.MISSING_PERMISSION_ERROR)
				return
			}
		}
	}

	user, err := s.Service.GetUser(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "User not found")
		} else {
			tracer.FailRequest(ctx, tracer.ScopeDatabase, "account.get.failed", "No se pudo consultar la cuenta", tracer.Err(err))
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}

	user.Password = ""
	common.WriteObjectResponse(w, http.StatusOK, user)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var u model.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}
	u.Id = id

	if err := s.Service.UpdateUser(ctx, &u); err != nil {
		tracer.FailRequest(ctx, tracer.ScopeDatabase, "account.update.failed", "No se pudo actualizar la cuenta", tracer.Err(err))
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("User %s updated successfully", id))
}

func (s *Server) activateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	isActive, _ := strconv.ParseBool(r.URL.Query().Get("status"))

	if err := s.Service.ActivateUser(ctx, id, isActive); err != nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "User not found")
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("User %s status updated", id))
}

func (s *Server) adminOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if !ok {
		common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
		return
	}
	hasAdmin, _ := s.Service.HasPermission(ctx, claims.UserID, common.AdminPermission)
	if !hasAdmin {
		common.WriteErrorResponse(w, common.MISSING_PERMISSION_ERROR)
		return
	}
	common.WriteSuccessResponse(w, http.StatusOK, "Admin access authorized")
}
