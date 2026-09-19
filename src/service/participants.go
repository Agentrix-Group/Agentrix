package service

import (
	"context"
	"crypto/sha512"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials    = errors.New("invalid username or password")
	ErrAccountInactive       = errors.New("participant account is inactive")
	ErrUsernameAlreadyExists = errors.New("username is already taken")
	ErrEmailAlreadyExists    = errors.New("email is already registered")
	ErrInvalidUsername       = errors.New("username must be between 3 and 50 characters and contain only letters, numbers, underscores, or hyphens")
	ErrInvalidEmail          = errors.New("invalid email address format")
	ErrInvalidPassword       = errors.New("password must be at least 6 characters")
)

// ParticipantService defines participant and authentication use cases (ATD-015).
type ParticipantService interface {
	Login(ctx context.Context, username, password string) (*model.Participant, error)
	Register(ctx context.Context, participant *model.Participant) error
	ListParticipants(ctx context.Context) ([]model.Participant, error)
	GetParticipant(ctx context.Context, id string) (*model.Participant, error)
	UpdateParticipant(ctx context.Context, participant *model.Participant) error
	ActivateParticipant(ctx context.Context, id string, isActive bool) error
	HasPermission(ctx context.Context, participantId, permission string) (bool, error)
}

func isValidUsername(u string) bool {
	if len(u) < 3 || len(u) > 50 {
		return false
	}
	for _, ch := range u {
		if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') && ch != '_' && ch != '-' {
			return false
		}
	}
	return true
}

func isValidEmail(e string) bool {
	if len(e) < 5 || len(e) > 255 {
		return false
	}
	atIdx := strings.Index(e, "@")
	dotIdx := strings.LastIndex(e, ".")
	return atIdx > 0 && dotIdx > atIdx+1 && dotIdx < len(e)-1
}

func (s *service) Login(ctx context.Context, username, password string) (*model.Participant, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	p, err := s.repo.GetParticipantByUsername(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !p.Active {
		return nil, ErrAccountInactive
	}

	hashedPassword := hashPassword(password)
	if p.Password != hashedPassword {
		return nil, ErrInvalidCredentials
	}

	return p, nil
}

func (s *service) Register(ctx context.Context, participant *model.Participant) error {
	if !isValidUsername(participant.Username) {
		return ErrInvalidUsername
	}

	if !isValidEmail(participant.Email) {
		return ErrInvalidEmail
	}

	if len(participant.Password) < 6 {
		return ErrInvalidPassword
	}

	// Verify username uniqueness
	existingUser, err := s.repo.GetParticipantByUsername(ctx, participant.Username)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingUser != nil {
		return ErrUsernameAlreadyExists
	}

	// Verify email uniqueness
	existingEmail, err := s.repo.GetParticipantByEmail(ctx, participant.Email)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingEmail != nil {
		return ErrEmailAlreadyExists
	}

	if participant.Id == "" {
		participant.Id = uuid.New().String()
	}
	// Always force participant role on self-registration regardless of input
	participant.RoleId = common.RoleParticipant

	participant.Password = hashPassword(participant.Password)
	participant.Active = true
	participant.CreatedAt = time.Now().UTC()

	err = s.repo.CreateParticipant(ctx, participant)
	if err != nil {
		return err
	}

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
