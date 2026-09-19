package server

import (
	"bytes"
	"context"
	"mime/multipart"
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
	createBundleFn  func(ctx context.Context, participantId, roleId, agentId string, archive []byte) (*model.Submission, error)
	getSubmissionFn func(ctx context.Context, id string) (*model.Submission, error)
	hasPermissionFn func(ctx context.Context, participantId, permission string) (bool, error)
}

func (m *mockSubmissionsService) CreateSubmissionBundle(ctx context.Context, participantId, roleId, agentId string, archive []byte) (*model.Submission, error) {
	if m.createBundleFn != nil {
		return m.createBundleFn(ctx, participantId, roleId, agentId, archive)
	}
	return nil, service.ErrInvalidBotBundle
}

func TestServerUploadSubmissionBundle(t *testing.T) {
	authMgr := auth.NewAuth("secret_test_access_key_32chars!", "secret_test_refresh_key_32chars!")
	token, err := authMgr.GenerateAuthToken("user-1", "participant")
	require.NoError(t, err)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("agent_id", "agent-1"))
	file, err := writer.CreateFormFile("bundle", "candidate.zip")
	require.NoError(t, err)
	_, err = file.Write([]byte("PK fake bundle"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	mockSvc := &mockSubmissionsService{
		createBundleFn: func(ctx context.Context, participantId, roleId, agentId string, archive []byte) (*model.Submission, error) {
			require.Equal(t, "user-1", participantId)
			require.Equal(t, "agent-1", agentId)
			require.NotEmpty(t, archive)
			return &model.Submission{Id: "sub-zip", AgentId: agentId, Language: "python", Status: "ready"}, nil
		},
	}
	server := NewServer(mockSvc)
	server.Auth = authMgr
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/upload", &body)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusCreated, response.Code)
	require.Contains(t, response.Body.String(), "sub-zip")
}

func TestServerUploadSubmissionBundleRequiresSubmitPermission(t *testing.T) {
	authMgr := auth.NewAuth("secret_test_access_key_32chars!", "secret_test_refresh_key_32chars!")
	token, err := authMgr.GenerateAuthToken("user-1", "participant")
	require.NoError(t, err)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("agent_id", "agent-1"))
	file, err := writer.CreateFormFile("bundle", "candidate.zip")
	require.NoError(t, err)
	_, err = file.Write([]byte("PK fake bundle"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	server := NewServer(&mockSubmissionsService{
		hasPermissionFn: func(context.Context, string, string) (bool, error) { return false, nil },
	})
	server.Auth = authMgr
	request := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/upload", &body)
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusForbidden, response.Code)
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
