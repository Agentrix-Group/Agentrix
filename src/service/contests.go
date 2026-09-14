package service

import (
	"context"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
)

func (s *service) ListContests(ctx context.Context) ([]model.Contest, error) {
	return s.repo.ListContests(ctx)
}

func (s *service) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	return s.repo.GetContest(ctx, id)
}

func (s *service) CreateContest(ctx context.Context, contest *model.Contest) error {
	tracer.Debugf(ctx, "Creating contest '%s'", contest.Name)
	if contest.Id == "" {
		contest.Id = uuid.New().String()
	}
	if contest.Status == "" {
		contest.Status = "upcoming"
	}
	contest.Active = true
	contest.CreatedAt = time.Now().UTC()

	return s.repo.CreateContest(ctx, contest)
}

func (s *service) UpdateContest(ctx context.Context, contest *model.Contest) error {
	return s.repo.UpdateContest(ctx, contest)
}

func (s *service) ActivateContest(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateContest(ctx, id, isActive)
}

func (s *service) ListCategories(ctx context.Context) ([]model.Category, error) {
	return s.repo.ListCategories(ctx)
}

func (s *service) GetCategory(ctx context.Context, id string) (*model.Category, error) {
	return s.repo.GetCategory(ctx, id)
}

func (s *service) CreateCategory(ctx context.Context, category *model.Category) error {
	if category.Id == "" {
		category.Id = uuid.New().String()
	}
	category.Active = true
	return s.repo.CreateCategory(ctx, category)
}

func (s *service) UpdateCategory(ctx context.Context, category *model.Category) error {
	return s.repo.UpdateCategory(ctx, category)
}

func (s *service) ActivateCategory(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateCategory(ctx, id, isActive)
}
