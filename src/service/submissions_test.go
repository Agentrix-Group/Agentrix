package service

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"
	"time"

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
	ownedAgent := &model.Agent{Id: "agent-1", OwnerUserId: "user-1", GameId: "starfighter"}
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
	// La subida no ejecuta el bot (ADR-0007): ni siquiera necesita validador.
	svc := NewService(repo, &mockArtifactStore{}, nil)
	submission, err := svc.CreateSubmissionBundle(ctx, "user-1", common.RoleParticipant, "agent-1", bundle)
	require.NoError(t, err)
	require.Same(t, created, submission)
	require.Equal(t, "python", submission.Language)
	require.Equal(t, common.SubmissionStatusValidating, submission.Status, "the worker runs the admission dry run")
	require.Contains(t, submission.CodePath, "/submissions/agent-1/v1/bundle/bot.py")

	badBundle := makeBotBundle(t,
		`{"name":"Candidate","entrypoint":"main.js","protocol_version":"1.0"}`,
		"print('bad manifest')\n",
	)
	_, err = svc.CreateSubmissionBundle(ctx, "user-1", common.RoleParticipant, "agent-1", badBundle)
	require.ErrorIs(t, err, ErrInvalidBotBundle)

	unknownFieldBundle := makeBotBundle(t,
		`{"name":"Candidate","entrypoint":"bot.py","protocol_version":"1.0","runtime":"python"}`,
		"print('unexpected manifest field')\n",
	)
	_, err = svc.CreateSubmissionBundle(ctx, "user-1", common.RoleParticipant, "agent-1", unknownFieldBundle)
	require.ErrorIs(t, err, ErrInvalidBotBundle)

	_, err = svc.CreateSubmissionBundle(ctx, "another-user", common.RoleParticipant, "agent-1", bundle)
	require.ErrorIs(t, err, ErrAgentNotOwned)
}

// mockAdmissionRepo simula la cola de admisión de submissions.
type mockAdmissionRepo struct {
	mockSubmissionRepo
	pending  []*model.Submission
	finished map[string][2]string
}

func (m *mockAdmissionRepo) ClaimNextAdmission(ctx context.Context, staleAfter time.Duration) (*model.Submission, error) {
	if len(m.pending) == 0 {
		return nil, nil
	}
	next := m.pending[0]
	m.pending = m.pending[1:]
	return next, nil
}

func (m *mockAdmissionRepo) FinishAdmission(ctx context.Context, id, status, detail string) error {
	m.finished[id] = [2]string{status, detail}
	return nil
}

// ADR-0014 (N4): el worker prueba cada submission pendiente y guarda
// 'ready' o 'rejected' con el motivo.
func TestProcessNextAdmission(t *testing.T) {
	ctx := context.Background()
	repo := &mockAdmissionRepo{
		pending:  []*model.Submission{{Id: "sub-ok", CodePath: "/ok/bot.py"}, {Id: "sub-bad", CodePath: "/bad/bot.py"}},
		finished: map[string][2]string{},
	}
	validator := mockPathValidator{failing: map[string]error{"/bad/bot.py": errors.New("bot failed admission: exceeded the 1024 MB memory limit")}}
	svc := NewService(repo, &mockArtifactStore{}, nil, validator)

	for i := 0; i < 2; i++ {
		processed, err := svc.ProcessNextAdmission(ctx)
		require.NoError(t, err)
		require.True(t, processed)
	}
	processed, err := svc.ProcessNextAdmission(ctx)
	require.NoError(t, err)
	require.False(t, processed, "no pending admissions left")

	require.Equal(t, [2]string{common.SubmissionStatusReady, ""}, repo.finished["sub-ok"])
	require.Equal(t, [2]string{common.SubmissionStatusRejected, "bot failed admission: exceeded the 1024 MB memory limit"}, repo.finished["sub-bad"])

	// Sin validador (proceso de la API) no se ejecuta ninguna admisión.
	_, err = NewService(repo, &mockArtifactStore{}, nil).ProcessNextAdmission(ctx)
	require.ErrorIs(t, err, ErrAdmissionFailed)
}

type mockPathValidator struct{ failing map[string]error }

func (m mockPathValidator) ValidateBot(ctx context.Context, codePath string) error {
	return m.failing[codePath]
}
