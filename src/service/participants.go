package service

import (
	"context"
	"crypto/sha512"
	"database/sql"
	"fmt"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
)

func (s *service) Login(ctx context.Context, username, password string) (*model.Participant, error) {
	tracer.Debugf(ctx, "Executing login authentication for username '%s'", username)

	p, err := s.repo.GetParticipantByUsername(ctx, username)
	if err != nil {
		tracer.Warnf(ctx, "Authentication lookup failed for username '%s'", username)
		return nil, err
	}

	hashedPassword := hashPassword(password)
	if p.Password != hashedPassword {
		tracer.Warnf(ctx, "Password mismatch for participant '%s'", username)
		return nil, sql.ErrNoRows
	}

	tracer.Debugf(ctx, "Participant '%s' authenticated successfully", username)
	return p, nil
}

func (s *service) Register(ctx context.Context, participant *model.Participant) error {
	tracer.Debugf(ctx, "Registering new participant '%s'", participant.Username)

	if participant.Id == "" {
		participant.Id = uuid.New().String()
	}
	if participant.RoleId == "" {
		participant.RoleId = common.RoleParticipant
	}

	participant.Password = hashPassword(participant.Password)
	participant.Active = true
	participant.CreatedAt = time.Now().UTC()

	err := s.repo.CreateParticipant(ctx, participant)
	if err != nil {
		tracer.Errorf(ctx, "Failed to register participant '%s': %s", participant.Username, err)
		return err
	}

	tracer.Debugf(ctx, "Participant '%s' registered with ID '%s'", participant.Username, participant.Id)
	return nil
}

func (s *service) ListParticipants(ctx context.Context) ([]model.Participant, error) {
	return s.repo.ListParticipants(ctx)
}

func (s *service) GetParticipant(ctx context.Context, id string) (*model.Participant, error) {
	return s.repo.GetParticipant(ctx, id)
}

func (s *service) UpdateParticipant(ctx context.Context, participant *model.Participant) error {
	return s.repo.UpdateParticipant(ctx, participant)
}

func (s *service) ActivateParticipant(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateParticipant(ctx, id, isActive)
}

func (s *service) HasPermission(ctx context.Context, participantId, permission string) (bool, error) {
	return s.repo.HasPermission(ctx, participantId, permission)
}

func hashPassword(password string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(password)))
}
