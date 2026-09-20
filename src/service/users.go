package service

import (
	"context"
	"crypto/sha512"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials    = errors.New("invalid username or password")
	ErrAccountInactive       = errors.New("user account is inactive")
	ErrUsernameAlreadyExists = errors.New("username is already taken")
	ErrEmailAlreadyExists    = errors.New("email is already registered")
	ErrInvalidUsername       = errors.New("username must be between 3 and 50 characters and contain only letters, numbers, underscores, or hyphens")
	ErrInvalidEmail          = errors.New("invalid email address format")
	ErrInvalidPassword       = errors.New("password must be at least 6 characters")
	ErrUserNotFound          = repository.ErrUserNotFound
)

// UserService defines user and authentication use cases (ATD-015).
type UserService interface {
	Login(ctx context.Context, username, password string) (*model.User, error)
	Register(ctx context.Context, user *model.User) error
	ListUsers(ctx context.Context) ([]model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	ActivateUser(ctx context.Context, id string, isActive bool) error
	HasPermission(ctx context.Context, userId, permission string) (bool, error)
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

func (s *service) Login(ctx context.Context, username, password string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	u, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !u.Active {
		return nil, ErrAccountInactive
	}

	hashedPassword := hashPassword(password)
	if u.Password != hashedPassword {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}

func (s *service) Register(ctx context.Context, user *model.User) error {
	if !isValidUsername(user.Username) {
		return ErrInvalidUsername
	}

	if !isValidEmail(user.Email) {
		return ErrInvalidEmail
	}

	if len(user.Password) < 6 {
		return ErrInvalidPassword
	}

	// Verify username uniqueness
	existingUser, err := s.repo.GetUserByUsername(ctx, user.Username)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingUser != nil {
		return ErrUsernameAlreadyExists
	}

	// Verify email uniqueness
	existingEmail, err := s.repo.GetUserByEmail(ctx, user.Email)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingEmail != nil {
		return ErrEmailAlreadyExists
	}

	if user.Id == "" {
		user.Id = uuid.New().String()
	}
	// Force default participant role on self-registration to prevent privilege escalation
	user.RoleId = common.RoleParticipant

	user.Password = hashPassword(user.Password)
	user.Active = true
	user.CreatedAt = time.Now().UTC()

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.repo.ListUsers(ctx)
}

func (s *service) GetUser(ctx context.Context, id string) (*model.User, error) {
	return s.repo.GetUser(ctx, id)
}

func (s *service) UpdateUser(ctx context.Context, user *model.User) error {
	return s.repo.UpdateUser(ctx, user)
}

func (s *service) ActivateUser(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateUser(ctx, id, isActive)
}

type rolePermCache struct {
	mu        sync.RWMutex
	rolePerms map[string]map[string]bool
	updatedAt map[string]time.Time
	ttl       time.Duration
}

var globalPermCache = &rolePermCache{
	rolePerms: make(map[string]map[string]bool),
	updatedAt: make(map[string]time.Time),
	ttl:       5 * time.Minute,
}

func (c *rolePermCache) get(roleId string) (map[string]bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	updated, ok := c.updatedAt[roleId]
	if !ok || time.Since(updated) > c.ttl {
		return nil, false
	}
	perms := c.rolePerms[roleId]
	return perms, true
}

func (c *rolePermCache) set(roleId string, perms map[string]bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rolePerms[roleId] = perms
	c.updatedAt[roleId] = time.Now()
}

func InvalidatePermissionCache() {
	globalPermCache.mu.Lock()
	defer globalPermCache.mu.Unlock()
	globalPermCache.rolePerms = make(map[string]map[string]bool)
	globalPermCache.updatedAt = make(map[string]time.Time)
}

func (s *service) HasPermission(ctx context.Context, userId, permission string) (bool, error) {
	if userId == "" || permission == "" {
		return false, nil
	}

	user, err := s.repo.GetUser(ctx, userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if !user.Active {
		return false, nil
	}

	perms, ok := globalPermCache.get(user.RoleId)
	if !ok {
		permList, err := s.repo.GetRolePermissions(ctx, user.RoleId)
		if err != nil {
			// Fallback directly to repo check
			return s.repo.HasPermission(ctx, userId, permission)
		}
		perms = make(map[string]bool, len(permList))
		for _, p := range permList {
			perms[p] = true
		}
		globalPermCache.set(user.RoleId, perms)
	}

	return perms[permission] || perms[common.AdminPermission], nil
}

func hashPassword(password string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(password)))
}

