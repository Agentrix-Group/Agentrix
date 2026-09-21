package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

type mockUserServerService struct {
	service.Service
	loginFn             func(ctx context.Context, username, password string) (*model.User, error)
	registerFn          func(ctx context.Context, u *model.User) error
	getUserFn           func(ctx context.Context, id string) (*model.User, error)
	hasPermissionFn     func(ctx context.Context, userId, permission string) (bool, error)
	getPublicContestFn  func(ctx context.Context, id string) (*model.Contest, error)
	enrollAgentFn       func(ctx context.Context, userId, contestId, agentId string, submissionId ...string) (*model.ContestEntry, *model.Ranking, error)
	listContestAgentsFn func(ctx context.Context, contestId string) ([]model.Ranking, error)
}

func (m *mockUserServerService) Login(ctx context.Context, username, password string) (*model.User, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, username, password)
	}
	return nil, service.ErrInvalidCredentials
}

func (m *mockUserServerService) Register(ctx context.Context, u *model.User) error {
	if m.registerFn != nil {
		return m.registerFn(ctx, u)
	}
	return nil
}

func (m *mockUserServerService) GetUser(ctx context.Context, id string) (*model.User, error) {
	if m.getUserFn != nil {
		return m.getUserFn(ctx, id)
	}
	return nil, nil
}

func (m *mockUserServerService) HasPermission(ctx context.Context, userId, permission string) (bool, error) {
	if m.hasPermissionFn != nil {
		return m.hasPermissionFn(ctx, userId, permission)
	}
	return true, nil
}

func (m *mockUserServerService) GetUserCapabilities(ctx context.Context, userId string) ([]string, error) {
	return nil, nil
}

func (m *mockUserServerService) GetRepository() repository.Repository {
	return nil
}

func (m *mockUserServerService) GetPublicContest(ctx context.Context, id string) (*model.Contest, error) {
	if m.getPublicContestFn != nil {
		return m.getPublicContestFn(ctx, id)
	}
	return nil, service.ErrContestNotFound
}

func (m *mockUserServerService) EnrollAgent(ctx context.Context, userId, contestId, agentId string, submissionId ...string) (*model.ContestEntry, *model.Ranking, error) {
	if m.enrollAgentFn != nil {
		return m.enrollAgentFn(ctx, userId, contestId, agentId, submissionId...)
	}
	return nil, nil, nil
}

func (m *mockUserServerService) ListContestAgents(ctx context.Context, contestId string) ([]model.Ranking, error) {
	if m.listContestAgentsFn != nil {
		return m.listContestAgentsFn(ctx, contestId)
	}
	return nil, nil
}

func TestAuthEndpoints(t *testing.T) {
	r := require.New(t)

	mockUser := &model.User{
		Id:       "u-100",
		Username: "probot",
		Email:    "probot@example.com",
		RoleId:   "participant",
		Active:   true,
	}

	mockSvc := &mockUserServerService{
		loginFn: func(ctx context.Context, username, password string) (*model.User, error) {
			if username == "probot" && password == "secret123" {
				return mockUser, nil
			}
			if username == "inactive" {
				return nil, service.ErrAccountInactive
			}
			return nil, service.ErrInvalidCredentials
		},
		registerFn: func(ctx context.Context, u *model.User) error {
			if u.Username == "duplicate" {
				return service.ErrUsernameAlreadyExists
			}
			if u.Email == "duplicate@example.com" {
				return service.ErrEmailAlreadyExists
			}
			if len(u.Password) < 6 {
				return service.ErrInvalidPassword
			}
			return nil
		},
		getUserFn: func(ctx context.Context, id string) (*model.User, error) {
			if id == "u-100" {
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
		r.NotNil(resp.User)
		r.Equal("probot", resp.User.Username)
		r.Empty(resp.User.Password) // Password omitted
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
		token, err := srv.Auth.GenerateAuthToken("u-100", "participant")
		r.NoError(err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusOK, rec.Code)
		var user model.User
		err = json.Unmarshal(rec.Body.Bytes(), &user)
		r.NoError(err)
		r.Equal("probot", user.Username)
		r.Empty(user.Password)
	})
}
