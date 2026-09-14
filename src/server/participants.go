package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/gorilla/mux"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token       *model.Token       `json:"token"`
	Participant *model.Participant `json:"participant"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		tracer.Warnf(ctx, "Failed to decode login request: %s", err)
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	if err := validateLoginRequest(req.Username, req.Password); err != nil {
		tracer.Warnf(ctx, "Validation failed for login: %s", err)
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	participant, err := s.Service.Login(ctx, req.Username, req.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Login credentials incorrect for '%s'", req.Username)
			common.WriteErrorMessage(w, common.INVALID_CREDENTIALS_ERROR, "Invalid username or password")
		} else {
			tracer.Errorf(ctx, "Database error during login for '%s': %s", req.Username, err)
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}

	token, err := s.Auth.GenerateAuthToken(participant.Id, participant.RoleId)
	if err != nil {
		tracer.Errorf(ctx, "Failed to generate auth token: %s", err)
		common.WriteErrorResponse(w, common.INTERNAL_ERROR)
		return
	}

	participant.Password = "" // Omit password hash in response
	common.WriteObjectResponse(w, http.StatusOK, LoginResponse{
		Token:       token,
		Participant: participant,
	})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var participant model.Participant
	if err := json.NewDecoder(r.Body).Decode(&participant); err != nil {
		tracer.Warnf(ctx, "Failed to decode register request: %s", err)
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	if err := validateParticipant(&participant); err != nil {
		tracer.Warnf(ctx, "Validation failed for registration: %s", err)
		common.WriteErrorResponse(w, common.MISSING_FIELDS_ERROR)
		return
	}

	if participant.Password == "" {
		common.WriteErrorMessage(w, common.MISSING_FIELDS_ERROR, "Password is required")
		return
	}

	err := s.Service.Register(ctx, &participant)
	if err != nil {
		tracer.Errorf(ctx, "Failed to register participant: %s", err)
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusCreated, fmt.Sprintf("Participant %s registered successfully", participant.Username))
}

func (s *Server) refreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}

	claims, err := s.Auth.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		tracer.Warnf(ctx, "Invalid refresh token: %s", err)
		common.WriteErrorResponse(w, common.INVALID_CREDENTIALS_ERROR)
		return
	}

	token, err := s.Auth.GenerateAuthToken(claims.ParticipantId, claims.RoleId)
	if err != nil {
		common.WriteErrorResponse(w, common.INTERNAL_ERROR)
		return
	}

	common.WriteObjectResponse(w, http.StatusOK, token)
}

func (s *Server) checkSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(common.UserContextKey).(*model.Claims)
	if !ok {
		common.WriteErrorResponse(w, common.ACCESS_DENIED_ERROR)
		return
	}

	participant, err := s.Service.GetParticipant(ctx, claims.ParticipantId)
	if err != nil {
		common.WriteErrorResponse(w, common.NOT_FOUND_ERROR)
		return
	}

	participant.Password = ""
	common.WriteObjectResponse(w, http.StatusOK, participant)
}

func (s *Server) listParticipants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	participants, err := s.Service.ListParticipants(ctx)
	if err != nil {
		tracer.Errorf(ctx, "Failed to list participants: %s", err)
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	for i := range participants {
		participants[i].Password = ""
	}
	common.WriteObjectResponse(w, http.StatusOK, participants)
}

func (s *Server) getParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	participant, err := s.Service.GetParticipant(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Participant not found")
		} else {
			common.WriteErrorResponse(w, common.DATABASE_ERROR)
		}
		return
	}

	participant.Password = ""
	common.WriteObjectResponse(w, http.StatusOK, participant)
}

func (s *Server) updateParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var p model.Participant
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		common.WriteErrorResponse(w, common.INVALID_REQUEST_ERROR)
		return
	}
	p.Id = id

	if err := s.Service.UpdateParticipant(ctx, &p); err != nil {
		common.WriteErrorResponse(w, common.DATABASE_ERROR)
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Participant %s updated successfully", id))
}

func (s *Server) activateParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	isActive, _ := strconv.ParseBool(r.URL.Query().Get("status"))

	if err := s.Service.ActivateParticipant(ctx, id, isActive); err != nil {
		common.WriteErrorMessage(w, common.NOT_FOUND_ERROR, "Participant not found")
		return
	}

	common.WriteSuccessResponse(w, http.StatusOK, fmt.Sprintf("Participant %s status updated", id))
}
