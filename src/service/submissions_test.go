package service

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockAdmissionValidator struct {
	err error
}

func (m mockAdmissionValidator) ValidateBot(ctx context.Context, codePath string) error {
	return m.err
}

func makeBotBundle(t *testing.T, manifest, bot string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range map[string]string{"agentrix.json": manifest, "bot.py": bot} {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return output.Bytes()
}

type mockArtifactStore struct {
	savedPath string
	saveErr   error
}

func (m *mockArtifactStore) Save(ctx context.Context, subpath string, content []byte) (string, error) {
	if m.saveErr != nil {
		return "", m.saveErr
	}
	m.savedPath = "/artifacts/" + subpath
	return m.savedPath, nil
}

func (m *mockArtifactStore) Read(ctx context.Context, subpath string) ([]byte, error) {
	return nil, nil
}

func (m *mockArtifactStore) Exists(subpath string) bool {
	return true
}

func (m *mockArtifactStore) GetPath(subpath string) string {
	return "/artifacts/" + subpath
}

func (m *mockArtifactStore) Delete(ctx context.Context, subpath string) error {
	return nil
}

func (m *mockArtifactStore) OpenWriter(ctx context.Context, subpath string) (io.WriteCloser, string, error) {
	return nil, "", errors.New("not implemented by submission tests")
}

func (m *mockArtifactStore) Move(ctx context.Context, sourceSubpath, targetSubpath string) error {
	return nil
}

type mockSubmissionRepo struct {
	repository.Repository
	getAgentFn               func(ctx context.Context, id string) (*model.Agent, error)
	listSubmissionsByAgentFn func(ctx context.Context, agentId string) ([]model.Submission, error)
	createSubmissionFn       func(ctx context.Context, s *model.Submission) error
}

func (m *mockSubmissionRepo) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	if m.getAgentFn != nil {
		return m.getAgentFn(ctx, id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubmissionRepo) ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error) {
	if m.listSubmissionsByAgentFn != nil {
		return m.listSubmissionsByAgentFn(ctx, agentId)
	}
	return nil, nil
}

func (m *mockSubmissionRepo) CreateSubmission(ctx context.Context, s *model.Submission) error {
	if m.createSubmissionFn != nil {
		return m.createSubmissionFn(ctx, s)
	}
	return nil
}

func TestCreateSubmissionBundle(t *testing.T) {
	ctx := context.Background()
	ownedAgent := &model.Agent{Id: "agent-1", ParticipantId: "part-1", GameId: "starfighter"}
	var created *model.Submission
	repo := &mockSubmissionRepo{
		getAgentFn: func(context.Context, string) (*model.Agent, error) { return ownedAgent, nil },
		createSubmissionFn: func(ctx context.Context, submission *model.Submission) error {
			created = submission
			return nil
		},
	}
	bundle := makeBotBundle(t,
		`{"name":"Candidate","entrypoint":"bot.py","protocol_version":"1.0"}`,
		"import json, sys\n",
	)
	svc := NewService(repo, &mockArtifactStore{}, nil, mockAdmissionValidator{})
	submission, err := svc.CreateSubmissionBundle(ctx, "part-1", common.RoleParticipant, "agent-1", bundle)
	require.NoError(t, err)
	require.Same(t, created, submission)
	require.Equal(t, "python", submission.Language)
	require.Contains(t, submission.CodePath, "/submissions/agent-1/v1/bot.py")

	badBundle := makeBotBundle(t,
		`{"name":"Candidate","entrypoint":"main.js","protocol_version":"1.0"}`,
		"print('bad manifest')\n",
	)
	_, err = svc.CreateSubmissionBundle(ctx, "part-1", common.RoleParticipant, "agent-1", badBundle)
	require.ErrorIs(t, err, ErrInvalidBotBundle)

	unknownFieldBundle := makeBotBundle(t,
		`{"name":"Candidate","entrypoint":"bot.py","protocol_version":"1.0","runtime":"python"}`,
		"print('unexpected manifest field')\n",
	)
	_, err = svc.CreateSubmissionBundle(ctx, "part-1", common.RoleParticipant, "agent-1", unknownFieldBundle)
	require.ErrorIs(t, err, ErrInvalidBotBundle)

	rejecting := NewService(repo, &mockArtifactStore{}, nil, mockAdmissionValidator{err: errors.New("tick timeout")})
	_, err = rejecting.CreateSubmissionBundle(ctx, "part-1", common.RoleParticipant, "agent-1", bundle)
	require.ErrorIs(t, err, ErrAdmissionFailed)

	_, err = svc.CreateSubmissionBundle(ctx, "another-participant", common.RoleParticipant, "agent-1", bundle)
	require.ErrorIs(t, err, ErrAgentNotOwned)
}
