package model

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

type UserStatus string

const (
	UserActive    UserStatus = "active"
	UserSuspended UserStatus = "suspended"
	UserDisabled  UserStatus = "disabled"
)

func (s UserStatus) Valid() bool {
	return s == UserActive || s == UserSuspended || s == UserDisabled
}

// Canonical roles. "participant" is not a platform role.
const (
	RoleAdmin     = "admin"
	RoleOrganizer = "organizer"
	RolePlayer    = "player"
	RoleReferee   = "referee"
	RoleSpectator = "spectator"
)

var Roles = []string{RoleAdmin, RoleOrganizer, RolePlayer, RoleReferee, RoleSpectator}

func ValidRole(role string) bool { return slices.Contains(Roles, role) }

// Capability is a namespaced permission. The catalog below mirrors the
// capabilities table seeded by the canonical migration; a test keeps both in
// sync.
type Capability string

const (
	CapUsersReadOwn       Capability = "users:read:own"
	CapUsersReadAny       Capability = "users:read:any"
	CapUsersUpdateOwn     Capability = "users:update:own"
	CapUsersUpdateAny     Capability = "users:update:any"
	CapUsersRolesManage   Capability = "users:roles:manage"
	CapAgentsCreate       Capability = "agents:create"
	CapAgentsReadOwn      Capability = "agents:read:own"
	CapAgentsReadAny      Capability = "agents:read:any"
	CapAgentsUpdateOwn    Capability = "agents:update:own"
	CapAgentsUpdateAny    Capability = "agents:update:any"
	CapSubmissionsCreate  Capability = "submissions:create:own"
	CapSubmissionsReadOwn Capability = "submissions:read:own"
	CapSubmissionsReadAny Capability = "submissions:read:any"
	CapContestsView       Capability = "contests:view"
	CapContestsManage     Capability = "contests:manage"
	CapEntriesCreateOwn   Capability = "entries:create:own"
	CapEntriesManageAny   Capability = "entries:manage:any"
	CapMatchesView        Capability = "matches:view"
	CapMatchesCreate      Capability = "matches:create"
	CapMatchesRun         Capability = "matches:run"
	CapMatchesCancel      Capability = "matches:cancel"
	CapRankingsView       Capability = "rankings:view"
	CapRankingsPublish    Capability = "rankings:publish"
	CapReplaysView        Capability = "replays:view"
	CapAdminAccess        Capability = "admin:access"
)

var Capabilities = []Capability{
	CapUsersReadOwn, CapUsersReadAny, CapUsersUpdateOwn, CapUsersUpdateAny, CapUsersRolesManage,
	CapAgentsCreate, CapAgentsReadOwn, CapAgentsReadAny, CapAgentsUpdateOwn, CapAgentsUpdateAny,
	CapSubmissionsCreate, CapSubmissionsReadOwn, CapSubmissionsReadAny,
	CapContestsView, CapContestsManage, CapEntriesCreateOwn, CapEntriesManageAny,
	CapMatchesView, CapMatchesCreate, CapMatchesRun, CapMatchesCancel,
	CapRankingsView, CapRankingsPublish, CapReplaysView, CapAdminAccess,
}

type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	Status       UserStatus
	Roles        []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Principal is the authenticated (or anonymous) actor of a request. Its
// capabilities are loaded from user_roles on every request, so role changes
// and revocations take effect immediately.
type Principal struct {
	UserID       string // empty for anonymous visitors
	SessionID    string
	Roles        []string
	Capabilities []Capability
}

func (p Principal) Anonymous() bool { return p.UserID == "" }

func (p Principal) Can(c Capability) bool { return slices.Contains(p.Capabilities, c) }

// CanFor reports whether the principal may act on a resource owned by ownerID
// given an "own" capability and its "any" counterpart.
func (p Principal) CanFor(ownerID string, own, any Capability) bool {
	if p.Can(any) {
		return true
	}
	return !p.Anonymous() && ownerID == p.UserID && p.Can(own)
}

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,32}$`)
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func ValidateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return Validation("invalid_username", "username must have 3-32 characters: letters, digits, '.', '_' or '-'")
	}
	return nil
}

func ValidateEmail(email string) error {
	if len(email) > 254 || !emailPattern.MatchString(email) {
		return Validation("invalid_email", "email address is not valid")
	}
	return nil
}
