package service

import (
	"context"
	"database/sql"
	"errors"
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

var (
	ErrSubmissionNotFound  = errors.New("submission not found")
	ErrEmptySubmissionCode = errors.New("submission code cannot be empty")
	ErrAgentNotOwned       = ErrUnauthorizedAgent
)

func (s *service) CreateSubmission(ctx context.Context, participantId, roleId string, submission *model.Submission, codeContent []byte) error {
	tracer.Debugf(ctx, "Creating submission for agent '%s' by participant '%s'", submission.AgentId, participantId)

	if len(codeContent) == 0 {
		tracer.Warnf(ctx, "Submission rejected: empty code payload for agent '%s'", submission.AgentId)
		return ErrEmptySubmissionCode
	}

	// Verify agent exists
	agent, err := s.repo.GetAgent(ctx, submission.AgentId)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Submission rejected: agent '%s' not found", submission.AgentId)
			return ErrAgentNotFound
		}
		tracer.Errorf(ctx, "Failed to retrieve agent '%s': %s", submission.AgentId, err)
		return err
	}

	// Verify ownership (unless admin)
	if roleId != common.RoleAdmin && agent.ParticipantId != participantId {
		tracer.Warnf(ctx, "Submission rejected: agent '%s' (owner '%s') not owned by caller '%s'", submission.AgentId, agent.ParticipantId, participantId)
		return ErrAgentNotOwned
	}

	if submission.Language == "" {
		submission.Language = "python"
	}

	// Auto-compute version if not explicitly supplied
	if submission.Version <= 0 {
		existing, err := s.repo.ListSubmissionsByAgent(ctx, submission.AgentId)
		if err == nil {
			submission.Version = len(existing) + 1
		} else {
			submission.Version = 1
		}
	}

	if submission.Id == "" {
		submission.Id = uuid.New().String()
	}

	submission.Status = common.SubmissionStatusReady
	submission.Active = true
	submission.CreatedAt = time.Now().UTC()

	// Save code to artifact store
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

	return s.repo.CreateSubmission(ctx, submission)
}

func (s *service) UpdateSubmission(ctx context.Context, submission *model.Submission) error {
	return s.repo.UpdateSubmission(ctx, submission)
}

func (s *service) ActivateSubmission(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateSubmission(ctx, id, isActive)
}
