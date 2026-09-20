package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

type mockServerService struct {
	service.Service
	loginFn             func(ctx context.Context, username, password string) (*model.Participant, error)
	registerFn          func(ctx context.Context, p *model.Participant) error
	getParticipantFn    func(ctx context.Context, id string) (*model.Participant, error)
	hasPermissionFn     func(ctx context.Context, participantId, permission string) (bool, error)
	getPublicContestFn  func(ctx context.Context, id string) (*model.Contest, error)
	enrollAgentFn       func(ctx context.Context, participantId, contestId, agentId string) (*model.ContestEntry, *model.Ranking, error)
	listContestAgentsFn func(ctx context.Context, contestId string) ([]model.Ranking, error)
}

func (m *mockServerService) Login(ctx context.Context, username, password string) (*model.Participant, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, username, password)
	}
	return nil, service.ErrInvalidCredentials
}

func (m *mockServerService) Register(ctx context.Context, p *model.Participant) error {
	if m.registerFn != nil {
		return m.registerFn(ctx, p)
	}
	return nil
}

func (m *mockServerService) GetParticipant(ctx context.Context, id string) (*model.Participant, error) {
	if m.getParticipantFn != nil {
		return m.getParticipantFn(ctx, id)
	}
	return nil, nil
}

func (m *mockServerService) GetUser(ctx context.Context, id string) (*model.User, error) {
	return m.GetParticipant(ctx, id)
}

func (m *mockServerService) HasPermission(ctx context.Context, participantId, permission string) (bool, error) {
	if m.hasPermissionFn != nil {
		return m.hasPermissionFn(ctx, participantId, permission)
	}
	return true, nil
}

func (m *mockServerService) GetPublicContest(ctx context.Context, id string) (*model.Contest, error) {
	if m.getPublicContestFn != nil {
		return m.getPublicContestFn(ctx, id)
	}
	return nil, service.ErrContestNotFound
}

func (m *mockServerService) EnrollAgent(ctx context.Context, participantId, contestId, agentId string) (*model.ContestEntry, *model.Ranking, error) {
	if m.enrollAgentFn != nil {
		return m.enrollAgentFn(ctx, participantId, contestId, agentId)
	}
	return nil, nil, nil
}

func (m *mockServerService) ListContestAgents(ctx context.Context, contestId string) ([]model.Ranking, error) {
	if m.listContestAgentsFn != nil {
		return m.listContestAgentsFn(ctx, contestId)
	}
	return nil, nil
}

func TestAuthEndpoints(t *testing.T) {
	r := require.New(t)

	mockUser := &model.Participant{
		Id:       "p-100",
		Username: "probot",
		Email:    "probot@example.com",
		RoleId:   "participant",
		Active:   true,
	}

	mockSvc := &mockServerService{
		loginFn: func(ctx context.Context, username, password string) (*model.Participant, error) {
			if username == "probot" && password == "secret123" {
				return mockUser, nil
			}
			if username == "inactive" {
				return nil, service.ErrAccountInactive
			}
			return nil, service.ErrInvalidCredentials
		},
		registerFn: func(ctx context.Context, p *model.Participant) error {
			if p.Username == "duplicate" {
				return service.ErrUsernameAlreadyExists
			}
			if p.Email == "duplicate@example.com" {
				return service.ErrEmailAlreadyExists
			}
			if len(p.Password) < 6 {
				return service.ErrInvalidPassword
			}
			return nil
		},
		getParticipantFn: func(ctx context.Context, id string) (*model.Participant, error) {
			if id == "p-100" {
				return mockUser, nil
			}
			return nil, nil
		},
	}

	srv := NewServer(mockSvc)

	t.Run("POST /api/v1/auth/login valid credentials returns 200 and token", func(t *testing.T) {
		body, _ := json.Marshal(model.LoginRequest{
			Username: "probot",
			Password: "secret123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusOK, rec.Code)
		var resp model.LoginResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		r.NoError(err)
		r.NotNil(resp.Token)
		r.NotEmpty(resp.Token.AccessToken)
		r.Equal("Bearer", resp.Token.TokenType)
		r.NotNil(resp.Participant)
		r.Equal("probot", resp.Participant.Username)
		r.Empty(resp.Participant.Password) // Password omitted
	})

	t.Run("POST /api/v1/auth/login wrong password returns 401", func(t *testing.T) {
		body, _ := json.Marshal(model.LoginRequest{
			Username: "probot",
			Password: "wrongpassword",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusUnauthorized, rec.Code)
	})

	t.Run("POST /api/v1/auth/login inactive account returns 403", func(t *testing.T) {
		body, _ := json.Marshal(model.LoginRequest{
			Username: "inactive",
			Password: "anypassword",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusForbidden, rec.Code)
	})

	t.Run("POST /api/v1/auth/register valid input returns 201 Created", func(t *testing.T) {
		body, _ := json.Marshal(model.RegisterRequest{
			Username: "newbot",
			Email:    "newbot@example.com",
			Password: "secretPassword",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusCreated, rec.Code)
	})

	t.Run("POST /api/v1/auth/register duplicate username returns 409 Conflict", func(t *testing.T) {
		body, _ := json.Marshal(model.RegisterRequest{
			Username: "duplicate",
			Email:    "valid@example.com",
			Password: "secretPassword",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusConflict, rec.Code)
	})

	t.Run("POST /api/v1/auth/register short password returns 400 Bad Request", func(t *testing.T) {
		body, _ := json.Marshal(model.RegisterRequest{
			Username: "anybot",
			Email:    "valid@example.com",
			Password: "123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusBadRequest, rec.Code)
	})

	t.Run("GET /api/v1/me without token returns 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusUnauthorized, rec.Code)
	})

	t.Run("GET /api/v1/me with valid Bearer token returns 200 OK and profile", func(t *testing.T) {
		token, err := srv.Auth.GenerateAuthToken("p-100", "participant")
		r.NoError(err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusOK, rec.Code)
		var user model.Participant
		err = json.Unmarshal(rec.Body.Bytes(), &user)
		r.NoError(err)
		r.Equal("probot", user.Username)
		r.Empty(user.Password)
	})
}
