package server

import (
	"net/http"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
)

func (s *Server) live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// ready answers load balancers with the aggregate status only; component
// details require admin:access.
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	readiness := s.svc.Readiness(r.Context())
	status := http.StatusOK
	if readiness.Status == service.Down {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, map[string]any{"status": readiness.Status, "checked_at": readiness.CheckedAt})
}

func (s *Server) adminReadiness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.svc.Readiness(r.Context()))
}

func (s *Server) setRefreshCookie(w http.ResponseWriter, token string, expires time.Time) {
	sameSite := http.SameSiteStrictMode
	switch s.cfg.Auth.CookieSameSite {
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}
	cookie := &http.Cookie{Name: RefreshCookieName, Value: token, Path: s.cfg.Auth.CookiePath, HttpOnly: true,
		Secure: s.cfg.Auth.CookieSecure, SameSite: sameSite}
	if token == "" {
		cookie.MaxAge = -1
	} else {
		cookie.Expires = expires
		cookie.MaxAge = int(time.Until(expires).Seconds())
	}
	http.SetCookie(w, cookie)
}

func refreshCookie(r *http.Request) string {
	c, err := r.Cookie(RefreshCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

type credentialsRequest struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	user, err := s.svc.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, userDTO(user, true))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	session, err := s.svc.Login(r.Context(), req.Username, req.Password, r.UserAgent(), s.clientIP(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	s.setRefreshCookie(w, session.RefreshToken, session.RefreshExpiresAt)
	writeJSON(w, http.StatusOK, sessionDTO(session, time.Now()))
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	session, err := s.svc.Refresh(r.Context(), refreshCookie(r))
	if err != nil {
		s.setRefreshCookie(w, "", time.Time{})
		writeError(w, r, err)
		return
	}
	s.setRefreshCookie(w, session.RefreshToken, session.RefreshExpiresAt)
	writeJSON(w, http.StatusOK, sessionDTO(session, time.Now()))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Logout(r.Context(), refreshCookie(r), principalFrom(r)); err != nil {
		writeError(w, r, err)
		return
	}
	s.setRefreshCookie(w, "", time.Time{})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) logoutAll(w http.ResponseWriter, r *http.Request) {
	n, err := s.svc.LogoutAll(r.Context(), principalFrom(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	s.setRefreshCookie(w, "", time.Time{})
	writeJSON(w, http.StatusOK, map[string]int64{"revoked_sessions": n})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	p := principalFrom(r)
	user, err := s.svc.Me(r.Context(), p)
	if err != nil {
		writeError(w, r, err)
		return
	}
	dto := userDTO(user, true)
	for _, c := range p.Capabilities {
		dto.Capabilities = append(dto.Capabilities, string(c))
	}
	writeJSON(w, http.StatusOK, dto)
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.svc.ListUsers(r.Context(), principalFrom(r), model.UserStatus(r.URL.Query().Get("status")))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list(users, func(u *model.User) UserDTO { return userDTO(u, true) }))
}

type createUserRequest struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	user, err := s.svc.CreateUser(r.Context(), principalFrom(r), req.Username, req.Email, req.Password, req.Roles)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, userDTO(user, true))
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	p := principalFrom(r)
	user, err := s.svc.GetUser(r.Context(), p, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userDTO(user, true))
}

type updateUserRequest struct {
	Email           *string `json:"email,omitempty"`
	CurrentPassword string  `json:"current_password,omitempty"`
	NewPassword     string  `json:"new_password,omitempty"`
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	user, err := s.svc.UpdateAccount(r.Context(), principalFrom(r), r.PathValue("id"), req.Email, req.CurrentPassword, req.NewPassword)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userDTO(user, true))
}

type rolesRequest struct {
	Roles []string `json:"roles"`
}

func (s *Server) replaceRoles(w http.ResponseWriter, r *http.Request) {
	var req rolesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	user, err := s.svc.ReplaceUserRoles(r.Context(), principalFrom(r), r.PathValue("id"), req.Roles)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userDTO(user, true))
}

type userStatusRequest struct {
	Status model.UserStatus `json:"status"`
}

// setUserStatus requires the target status explicitly; a missing field is a
// validation error, never an implicit "disabled".
func (s *Server) setUserStatus(w http.ResponseWriter, r *http.Request) {
	var req userStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if req.Status == "" {
		writeError(w, r, model.Validation("status_required", "status is required"))
		return
	}
	user, err := s.svc.SetUserStatus(r.Context(), principalFrom(r), r.PathValue("id"), req.Status)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userDTO(user, true))
}
