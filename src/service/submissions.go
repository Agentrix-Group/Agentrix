package service

import (
	"context"
	"fmt"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
)

func (s *service) ListSubmissions(ctx context.Context) ([]model.Submission, error) {
	return s.repo.ListSubmissions(ctx)
}

func (s *service) ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error) {
	return s.repo.ListSubmissionsByAgent(ctx, agentId)
}

func (s *service) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	return s.repo.GetSubmission(ctx, id)
}

func (s *service) CreateSubmission(ctx context.Context, submission *model.Submission, codeContent []byte) error {
	tracer.Debugf(ctx, "Creating submission for agent '%s'", submission.AgentId)
	if submission.Id == "" {
		submission.Id = uuid.New().String()
	}

	if submission.Language == "" {
		submission.Language = "python"
	}

	if submission.Status == "" {
		submission.Status = common.SubmissionStatusReady
	}

	submission.Active = true
	submission.CreatedAt = time.Now().UTC()

	// Save code to artifact store if provided
	if len(codeContent) > 0 {
		ext := "py"
		if submission.Language == "go" {
			ext = "go"
		} else if submission.Language == "javascript" || submission.Language == "js" {
			ext = "js"
		}
		subpath := fmt.Sprintf("submissions/%s/agent_v%d.%s", submission.AgentId, submission.Version, ext)
		savedPath, err := s.artifacts.Save(ctx, subpath, codeContent)
		if err != nil {
			tracer.Errorf(ctx, "Failed to save submission artifact: %s", err)
			return err
		}
		submission.CodePath = savedPath
	}

	return s.repo.CreateSubmission(ctx, submission)
}

func (s *service) UpdateSubmission(ctx context.Context, submission *model.Submission) error {
	return s.repo.UpdateSubmission(ctx, submission)
}

func (s *service) ActivateSubmission(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateSubmission(ctx, id, isActive)
}
