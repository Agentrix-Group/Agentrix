package repository

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestUsersRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListUsers
	users, err := repo.ListUsers(ctx)
	r.Error(err)
	r.Nil(users)

	// GetUser
	u, err := repo.GetUser(ctx, "u1")
	r.Error(err)
	r.Nil(u)

	// GetUserByUsername
	uUser, err := repo.GetUserByUsername(ctx, "user1")
	r.Error(err)
	r.Nil(uUser)

	// GetUserByEmail
	uEmail, err := repo.GetUserByEmail(ctx, "user@example.com")
	r.Error(err)
	r.Nil(uEmail)

	// CreateUser
	err = repo.CreateUser(ctx, &model.User{Username: "user1"})
	r.Error(err)

	// UpdateUser
	err = repo.UpdateUser(ctx, &model.User{Id: "u1", Username: "user1"})
	r.Error(err)

	// ActivateUser
	err = repo.ActivateUser(ctx, "u1", false)
	r.Error(err)

	// HasPermission
	hasPerm, err := repo.HasPermission(ctx, "u1", "admin")
	r.Error(err)
	r.False(hasPerm)
}
