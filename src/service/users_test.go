package service

import (
	"context"
	"crypto/sha512"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockUserRepo struct {
	repository.Repository
	getByUsernameFn func(ctx context.Context, username string) (*model.User, error)
	getByEmailFn    func(ctx context.Context, email string) (*model.User, error)
	createFn        func(ctx context.Context, u *model.User) error
}

func (m *mockUserRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepo) CreateUser(ctx context.Context, u *model.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, u)
	}
	return nil
}

func TestUserService_Login(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	rawPassword := "validPass123"
	hashed := fmt.Sprintf("%x", sha512.Sum512([]byte(rawPassword)))

	activeUser := &model.User{
		Id:       "p1",
		Username: "botcoder",
		Email:    "botcoder@example.com",
		Password: hashed,
		RoleId:   "participant",
		Active:   true,
	}

	inactiveUser := &model.User{
		Id:       "p2",
		Username: "inactivebot",
		Email:    "inactive@example.com",
		Password: hashed,
		RoleId:   "participant",
		Active:   false,
	}

	t.Run("Valid credentials succeed", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				if username == "botcoder" {
					return activeUser, nil
				}
				return nil, sql.ErrNoRows
			},
		}
		svc := NewService(repo, nil, nil)

		user, err := svc.Login(ctx, "botcoder", rawPassword)
		r.NoError(err)
		r.NotNil(user)
		r.Equal("botcoder", user.Username)
	})

	t.Run("Invalid password fails", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return activeUser, nil
			},
		}
		svc := NewService(repo, nil, nil)

		user, err := svc.Login(ctx, "botcoder", "wrongPassword")
		r.Error(err)
		r.True(errors.Is(err, ErrInvalidCredentials))
		r.Nil(user)
	})

	t.Run("Unknown username fails", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return nil, sql.ErrNoRows
			},
		}
		svc := NewService(repo, nil, nil)

		user, err := svc.Login(ctx, "nonexistent", rawPassword)
		r.Error(err)
		r.True(errors.Is(err, ErrInvalidCredentials))
		r.Nil(user)
	})

	t.Run("Inactive account is rejected", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return inactiveUser, nil
			},
		}
		svc := NewService(repo, nil, nil)

		user, err := svc.Login(ctx, "inactivebot", rawPassword)
		r.Error(err)
		r.True(errors.Is(err, ErrAccountInactive))
		r.Nil(user)
	})

	t.Run("Empty fields fail", func(t *testing.T) {
		svc := NewService(&mockUserRepo{}, nil, nil)

		user, err := svc.Login(ctx, "", "")
		r.Error(err)
		r.True(errors.Is(err, ErrInvalidCredentials))
		r.Nil(user)
	})
}

func TestUserService_Register(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	t.Run("Valid registration succeeds and hashes password", func(t *testing.T) {
		var created *model.User
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return nil, sql.ErrNoRows
			},
			getByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
				return nil, sql.ErrNoRows
			},
			createFn: func(ctx context.Context, u *model.User) error {
				created = u
				return nil
			},
		}
		svc := NewService(repo, nil, nil)

		u := &model.User{
			Username: "newcoder",
			Email:    "newcoder@example.com",
			Password: "securePassword123",
		}

		err := svc.Register(ctx, u)
		r.NoError(err)
		r.NotNil(created)
		r.Equal("newcoder", created.Username)
		r.Equal("newcoder@example.com", created.Email)
		r.True(created.Active)
		r.NotEmpty(created.Id)
		r.Equal("participant", created.RoleId)
		// Password should be hashed, not plaintext
		r.NotEqual("securePassword123", created.Password)
		expectedHash := fmt.Sprintf("%x", sha512.Sum512([]byte("securePassword123")))
		r.Equal(expectedHash, created.Password)
	})

	t.Run("Supplied role_id is ignored and forced to participant", func(t *testing.T) {
		var created *model.User
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return nil, sql.ErrNoRows
			},
			getByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
				return nil, sql.ErrNoRows
			},
			createFn: func(ctx context.Context, u *model.User) error {
				created = u
				return nil
			},
		}
		svc := NewService(repo, nil, nil)

		u := &model.User{
			Username: "sneakyadmin",
			Email:    "sneaky@example.com",
			Password: "securePassword123",
			RoleId:   "admin", // Malicious attempt to self-assign admin role
		}

		err := svc.Register(ctx, u)
		r.NoError(err)
		r.NotNil(created)
		r.Equal("participant", created.RoleId) // Must be forced to participant
	})

	t.Run("Duplicate username is rejected", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return &model.User{Id: "existing-id", Username: username}, nil
			},
		}
		svc := NewService(repo, nil, nil)

		u := &model.User{
			Username: "existinguser",
			Email:    "diff@example.com",
			Password: "password123",
		}

		err := svc.Register(ctx, u)
		r.Error(err)
		r.True(errors.Is(err, ErrUsernameAlreadyExists))
	})

	t.Run("Duplicate email is rejected", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return nil, sql.ErrNoRows
			},
			getByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
				return &model.User{Id: "existing-email-id", Email: email}, nil
			},
		}
		svc := NewService(repo, nil, nil)

		u := &model.User{
			Username: "brandnew",
			Email:    "existing@example.com",
			Password: "password123",
		}

		err := svc.Register(ctx, u)
		r.Error(err)
		r.True(errors.Is(err, ErrEmailAlreadyExists))
	})

	t.Run("Invalid username format is rejected", func(t *testing.T) {
		svc := NewService(&mockUserRepo{}, nil, nil)

		// Too short
		err := svc.Register(ctx, &model.User{
			Username: "ab",
			Email:    "valid@example.com",
			Password: "password123",
		})
		r.Error(err)
		r.True(errors.Is(err, ErrInvalidUsername))

		// Invalid characters
		err = svc.Register(ctx, &model.User{
			Username: "user!name#",
			Email:    "valid@example.com",
			Password: "password123",
		})
		r.Error(err)
		r.True(errors.Is(err, ErrInvalidUsername))
	})

	t.Run("Invalid email format is rejected", func(t *testing.T) {
		svc := NewService(&mockUserRepo{}, nil, nil)

		err := svc.Register(ctx, &model.User{
			Username: "validuser",
			Email:    "not-an-email",
			Password: "password123",
		})
		r.Error(err)
		r.True(errors.Is(err, ErrInvalidEmail))
	})

	t.Run("Short password is rejected", func(t *testing.T) {
		svc := NewService(&mockUserRepo{}, nil, nil)

		err := svc.Register(ctx, &model.User{
			Username: "validuser",
			Email:    "valid@example.com",
			Password: "12345",
		})
		r.Error(err)
		r.True(errors.Is(err, ErrInvalidPassword))
	})
}
