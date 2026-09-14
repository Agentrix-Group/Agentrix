package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

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

func TestCreateSubmission(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	ownedAgent := &model.Agent{
		Id:            "agent-1",
		ParticipantId: "part-1",
		Name:          "HunterBot",
		GameId:        "arena-basica",
	}

	t.Run("Valid submission succeeds and saves code artifact", func(t *testing.T) {
		var created *model.Submission
		artifacts := &mockArtifactStore{}
		repo := &mockSubmissionRepo{
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				if id == "agent-1" {
					return ownedAgent, nil
				}
				return nil, sql.ErrNoRows
			},
			createSubmissionFn: func(ctx context.Context, s *model.Submission) error {
				created = s
				return nil
			},
		}

		svc := NewService(repo, artifacts, nil)

		sub := &model.Submission{
			AgentId:  "agent-1",
			Language: "python",
		}
		code := []byte("print('hello bot')")

		err := svc.CreateSubmission(ctx, "part-1", common.RoleParticipant, sub, code)
		r.NoError(err)
		r.NotNil(created)
		r.Equal("agent-1", created.AgentId)
		r.Equal(1, created.Version)
		r.Equal(common.SubmissionStatusReady, created.Status)
		r.NotEmpty(created.CodePath)
		r.Equal("/artifacts/submissions/agent-1/agent_v1.py", created.CodePath)
	})

	t.Run("Rejects empty submission code", func(t *testing.T) {
		repo := &mockSubmissionRepo{
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return ownedAgent, nil
			},
		}
		svc := NewService(repo, &mockArtifactStore{}, nil)

		sub := &model.Submission{AgentId: "agent-1"}
		err := svc.CreateSubmission(ctx, "part-1", common.RoleParticipant, sub, []byte(""))
		r.Error(err)
		r.True(errors.Is(err, ErrEmptySubmissionCode))
	})

	t.Run("Rejects submission if agent belongs to another participant", func(t *testing.T) {
		repo := &mockSubmissionRepo{
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return ownedAgent, nil
			},
		}
		svc := NewService(repo, &mockArtifactStore{}, nil)

		sub := &model.Submission{AgentId: "agent-1"}
		err := svc.CreateSubmission(ctx, "another-part", common.RoleParticipant, sub, []byte("code"))
		r.Error(err)
		r.True(errors.Is(err, ErrAgentNotOwned))
	})

	t.Run("Admin can submit for any participant agent", func(t *testing.T) {
		var created *model.Submission
		repo := &mockSubmissionRepo{
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return ownedAgent, nil
			},
			createSubmissionFn: func(ctx context.Context, s *model.Submission) error {
				created = s
				return nil
			},
		}
		svc := NewService(repo, &mockArtifactStore{}, nil)

		sub := &model.Submission{AgentId: "agent-1"}
		err := svc.CreateSubmission(ctx, "admin-user", common.RoleAdmin, sub, []byte("admin-override-code"))
		r.NoError(err)
		r.NotNil(created)
	})

	t.Run("Auto-increments version based on existing submissions", func(t *testing.T) {
		var created *model.Submission
		repo := &mockSubmissionRepo{
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return ownedAgent, nil
			},
			listSubmissionsByAgentFn: func(ctx context.Context, agentId string) ([]model.Submission, error) {
				return []model.Submission{
					{Id: "sub-1", Version: 1},
					{Id: "sub-2", Version: 2},
				}, nil
			},
			createSubmissionFn: func(ctx context.Context, s *model.Submission) error {
				created = s
				return nil
			},
		}
		svc := NewService(repo, &mockArtifactStore{}, nil)

		sub := &model.Submission{AgentId: "agent-1"}
		err := svc.CreateSubmission(ctx, "part-1", common.RoleParticipant, sub, []byte("version-3-code"))
		r.NoError(err)
		r.NotNil(created)
		r.Equal(3, created.Version)
	})
}
