package service

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockResultsRepo struct {
	repository.Repository
	listResultsFn        func(ctx context.Context) ([]model.Result, error)
	listResultsByMatchFn func(ctx context.Context, matchId string) ([]model.Result, error)
	getResultFn          func(ctx context.Context, id string) (*model.Result, error)
	createResultFn       func(ctx context.Context, result *model.Result) error
}

func (m *mockResultsRepo) ListResults(ctx context.Context) ([]model.Result, error) {
	if m.listResultsFn != nil {
		return m.listResultsFn(ctx)
	}
	return nil, nil
}

func (m *mockResultsRepo) ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error) {
	if m.listResultsByMatchFn != nil {
		return m.listResultsByMatchFn(ctx, matchId)
	}
	return nil, nil
}

func (m *mockResultsRepo) GetResult(ctx context.Context, id string) (*model.Result, error) {
	if m.getResultFn != nil {
		return m.getResultFn(ctx, id)
	}
	return &model.Result{Id: id, Score: 50}, nil
}

func (m *mockResultsRepo) CreateResult(ctx context.Context, result *model.Result) error {
	if m.createResultFn != nil {
		return m.createResultFn(ctx, result)
	}
	return nil
}

func TestResultsService(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	createdResults := make([]*model.Result, 0)
	mockRepo := &mockResultsRepo{
		listResultsFn: func(ctx context.Context) ([]model.Result, error) {
			return []model.Result{{Id: "res-1", Score: 100}}, nil
		},
		listResultsByMatchFn: func(ctx context.Context, matchId string) ([]model.Result, error) {
			return []model.Result{{Id: "res-1", MatchId: matchId, Score: 100}}, nil
		},
		createResultFn: func(ctx context.Context, result *model.Result) error {
			createdResults = append(createdResults, result)
			return nil
		},
	}

	svc := NewService(mockRepo, nil, nil)

	// ListResults
	results, err := svc.ListResults(ctx)
	r.NoError(err)
	r.Len(results, 1)

	// ListResultsByMatch
	matchResults, err := svc.ListResultsByMatch(ctx, "m1")
	r.NoError(err)
	r.Len(matchResults, 1)

	// GetResult
	res, err := svc.GetResult(ctx, "res-1")
	r.NoError(err)
	r.Equal("res-1", res.Id)

	// CreateResult
	newRes := &model.Result{MatchId: "m1", SubmissionId: "sub-1", Score: 80}
	err = svc.CreateResult(ctx, newRes)
	r.NoError(err)
	r.NotEmpty(newRes.Id)
	r.Len(createdResults, 1)
}
