package repository

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestParticipantsRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListParticipants
	participants, err := repo.ListParticipants(ctx)
	r.Error(err)
	r.Nil(participants)

	// GetParticipant
	p, err := repo.GetParticipant(ctx, "p1")
	r.Error(err)
	r.Nil(p)

	// GetParticipantByUsername
	pUser, err := repo.GetParticipantByUsername(ctx, "user1")
	r.Error(err)
	r.Nil(pUser)

	// GetParticipantByEmail
	pEmail, err := repo.GetParticipantByEmail(ctx, "user@example.com")
	r.Error(err)
	r.Nil(pEmail)

	// CreateParticipant
	err = repo.CreateParticipant(ctx, &model.Participant{Username: "user1"})
	r.Error(err)

	// UpdateParticipant
	err = repo.UpdateParticipant(ctx, &model.Participant{Id: "p1", Username: "user1"})
	r.Error(err)

	// ActivateParticipant
	err = repo.ActivateParticipant(ctx, "p1", false)
	r.Error(err)

	// HasPermission
	hasPerm, err := repo.HasPermission(ctx, "p1", "admin")
	r.Error(err)
	r.False(hasPerm)
}
