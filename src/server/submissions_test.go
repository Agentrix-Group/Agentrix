package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/F4nk1/Agentrix/src/auth"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

type mockSubmissionsService struct {
	service.Service
	createSubmissionFn func(ctx context.Context, participantId, roleId string, submission *model.Submission, codeContent []byte) error
	getSubmissionFn    func(ctx context.Context, id string) (*model.Submission, error)
	hasPermissionFn    func(ctx context.Context, participantId, permission string) (bool, error)
}

func (m *mockSubmissionsService) CreateSubmission(ctx context.Context, participantId, roleId string, submission *model.Submission, codeContent []byte) error {
	if m.createSubmissionFn != nil {
		return m.createSubmissionFn(ctx, participantId, roleId, submission, codeContent)
	}
	return nil
}

func (m *mockSubmissionsService) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	if m.getSubmissionFn != nil {
		return m.getSubmissionFn(ctx, id)
	}
	return nil, service.ErrSubmissionNotFound
}

func (m *mockSubmissionsService) HasPermission(ctx context.Context, participantId, permission string) (bool, error) {
	if m.hasPermissionFn != nil {
		return m.hasPermissionFn(ctx, participantId, permission)
	}
	return true, nil
}

func TestServer_CreateSubmission(t *testing.T) {
	r := require.New(t)
	authMgr := auth.NewAuth("secret_test_access_key_32chars!", "secret_test_refresh_key_32chars!")

	validToken, err := authMgr.GenerateAuthToken("user-1", "participant")
	r.NoError(err)

	t.Run("Valid authenticated submission returns 201 Created", func(t *testing.T) {
		mockSvc := &mockSubmissionsService{
			createSubmissionFn: func(ctx context.Context, participantId, roleId string, s *model.Submission, code []byte) error {
				r.Equal("user-1", participantId)
				r.Equal("agent-1", s.AgentId)
				r.Equal("print('ok')", string(code))
				s.Id = "sub-100"
				s.Version = 1
				return nil
			},
		}

		server := NewServer(mockSvc)
		server.Auth = authMgr

		body, _ := json.Marshal(map[string]interface{}{
			"agent_id": "agent-1",
			"code":     "print('ok')",
			"language": "python",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)

		r.Equal(http.StatusCreated, rr.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		r.NoError(err)
		r.Equal("sub-100", resp["id"])
	})

	t.Run("Missing auth token returns 401 Unauthorized", func(t *testing.T) {
		server := NewServer(&mockSubmissionsService{})
		server.Auth = authMgr

		body, _ := json.Marshal(map[string]interface{}{
			"agent_id": "agent-1",
			"code":     "code",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)

		r.Equal(http.StatusUnauthorized, rr.Code)
	})

	t.Run("Submission for unowned agent returns 403 Forbidden", func(t *testing.T) {
		mockSvc := &mockSubmissionsService{
			createSubmissionFn: func(ctx context.Context, participantId, roleId string, s *model.Submission, code []byte) error {
				return service.ErrAgentNotOwned
			},
		}

		server := NewServer(mockSvc)
		server.Auth = authMgr

		body, _ := json.Marshal(map[string]interface{}{
			"agent_id": "other-agent",
			"code":     "code",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)

		r.Equal(http.StatusForbidden, rr.Code)
	})

	t.Run("Submission with empty code returns 400 Bad Request", func(t *testing.T) {
		mockSvc := &mockSubmissionsService{
			createSubmissionFn: func(ctx context.Context, participantId, roleId string, s *model.Submission, code []byte) error {
				return service.ErrEmptySubmissionCode
			},
		}

		server := NewServer(mockSvc)
		server.Auth = authMgr

		body, _ := json.Marshal(map[string]interface{}{
			"agent_id": "agent-1",
			"code":     "",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)

		r.Equal(http.StatusBadRequest, rr.Code)
	})

	t.Run("Submission for missing agent returns 404 Not Found", func(t *testing.T) {
		mockSvc := &mockSubmissionsService{
			createSubmissionFn: func(ctx context.Context, participantId, roleId string, s *model.Submission, code []byte) error {
				return service.ErrAgentNotFound
			},
		}

		server := NewServer(mockSvc)
		server.Auth = authMgr

		body, _ := json.Marshal(map[string]interface{}{
			"agent_id": "nonexistent-agent",
			"code":     "code",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+validToken.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)

		r.Equal(http.StatusNotFound, rr.Code)
	})
}
