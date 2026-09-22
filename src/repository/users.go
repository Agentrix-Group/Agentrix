package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/model"
)

type Repository interface {
	// Contests & Categories
	ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	ListContests(ctx context.Context) ([]model.Contest, error)
	GetContest(ctx context.Context, id string) (*model.Contest, error)
	CreateContest(ctx context.Context, contest *model.Contest) error
	UpdateContest(ctx context.Context, contest *model.Contest) error
	ActivateContest(ctx context.Context, id string, isActive bool) error
	ListCategories(ctx context.Context) ([]model.Category, error)
	GetCategory(ctx context.Context, id string) (*model.Category, error)
	CreateCategory(ctx context.Context, category *model.Category) error
	UpdateCategory(ctx context.Context, category *model.Category) error
	ActivateCategory(ctx context.Context, id string, isActive bool) error

	// Contest Entries
	CreateContestEntry(ctx context.Context, entry *model.ContestEntry) error
	ListContestEntries(ctx context.Context, contestId string) ([]model.ContestEntry, error)
	GetContestEntry(ctx context.Context, contestId, agentId string) (*model.ContestEntry, error)

	// Users & Auth
	ListUsers(ctx context.Context) ([]model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, user *model.User) error
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error
	ActivateUser(ctx context.Context, id string, isActive bool) error
	HasPermission(ctx context.Context, userId string, permission string) (bool, error)
	GetRolePermissions(ctx context.Context, roleId string) ([]string, error)
	GetUserEffectivePermissions(ctx context.Context, userId string) ([]string, error)
	GetUserRoles(ctx context.Context, userId string) ([]string, error)

	// Sessions
	CreateSession(ctx context.Context, session *model.Session) error
	GetSessionByHash(ctx context.Context, tokenHash string) (*model.Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeSessionFamily(ctx context.Context, familyID string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	PruneOldestUserSessions(ctx context.Context, userID string, keepCount int) error
	UpdateSessionLastUsed(ctx context.Context, sessionID string) error

	// Games
	GetGame(ctx context.Context, id string) (*model.Game, error)

	// Agents
	ListAgents(ctx context.Context) ([]model.Agent, error)
	ListAgentsByOwner(ctx context.Context, ownerUserId string) ([]model.Agent, error)
	GetAgent(ctx context.Context, id string) (*model.Agent, error)
	CreateAgent(ctx context.Context, agent *model.Agent) error
	UpdateAgent(ctx context.Context, agent *model.Agent) error
	ActivateAgent(ctx context.Context, id string, isActive bool) error

	// Submissions
	ListSubmissions(ctx context.Context) ([]model.Submission, error)
	ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error)
	GetSubmission(ctx context.Context, id string) (*model.Submission, error)
	CreateSubmission(ctx context.Context, submission *model.Submission) error

	// Matches
	ListMatches(ctx context.Context) ([]model.Match, error)
	ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error)
	GetMatch(ctx context.Context, id string) (*model.Match, error)
	CreateMatch(ctx context.Context, match *model.Match) error
	UpdateMatch(ctx context.Context, match *model.Match) error
	ActivateMatch(ctx context.Context, id string, isActive bool) error

	// Results
	ListResults(ctx context.Context) ([]model.Result, error)
	ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error)
	GetResult(ctx context.Context, id string) (*model.Result, error)
	CreateResult(ctx context.Context, result *model.Result) error

	// Rankings
	ListRankings(ctx context.Context) ([]model.Ranking, error)
	ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error)
	GetRanking(ctx context.Context, id string) (*model.Ranking, error)
	GetRankingByContestAndAgent(ctx context.Context, contestId, agentId string) (*model.Ranking, error)
	UpsertRanking(ctx context.Context, ranking *model.Ranking) error
	ResetContestRankings(ctx context.Context, contestId string) error
	IsRunAppliedToRanking(ctx context.Context, contestId, runId string) (bool, error)
	RecordAppliedRun(ctx context.Context, contestId, runId, matchId string) error
	ClearAppliedRuns(ctx context.Context, contestId string) error
	CreateRankingSnapshot(ctx context.Context, snapshot *model.RankingSnapshot) error
	ListRankingSnapshots(ctx context.Context, contestId string) ([]model.RankingSnapshot, error)
	GetLatestRankingSnapshot(ctx context.Context, contestId string) (*model.RankingSnapshot, error)
	GetRankingSnapshotByVersion(ctx context.Context, contestId string, version int) (*model.RankingSnapshot, error)

	// Health & Schema
	Ping(ctx context.Context) error
	CheckSchema(ctx context.Context) error
}

// UserReader defines read-only operations for users.
type UserReader interface {
	ListUsers(ctx context.Context) ([]model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	HasPermission(ctx context.Context, userId string, permission string) (bool, error)
	GetRolePermissions(ctx context.Context, roleId string) ([]string, error)
	GetUserEffectivePermissions(ctx context.Context, userId string) ([]string, error)
	GetUserRoles(ctx context.Context, userId string) ([]string, error)
}

// UserWriter defines write-only operations for users.
type UserWriter interface {
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, user *model.User) error
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error
	ActivateUser(ctx context.Context, id string, isActive bool) error
}

// UserRepository combines user read and write operations.
type UserRepository interface {
	UserReader
	UserWriter
}

type repository struct {
	conn *connection.Connection
}

func NewRepository(conn *connection.Connection) Repository {
	return &repository{conn: conn}
}

func (r *repository) getDb() (*sql.DB, error) {
	if r == nil || r.conn == nil || r.conn.Db == nil {
		return nil, errors.New("database not connected")
	}
	return r.conn.Db, nil
}

func (r *repository) Ping(ctx context.Context) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	return db.PingContext(ctx)
}

func (r *repository) CheckSchema(ctx context.Context) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	return database.CheckSchemaCompatible(ctx, db)
}

func (r *repository) ListUsers(ctx context.Context) ([]model.User, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, username, email, role_id, active, created_at FROM users WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.Id, &u.Username, &u.Email, &u.RoleId, &u.Active, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func (r *repository) GetUser(ctx context.Context, id string) (*model.User, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, username, email, password, role_id, active, created_at FROM users WHERE id = $1`

	var u model.User
	err = db.QueryRowContext(ctx, query, id).Scan(
		&u.Id, &u.Username, &u.Email, &u.Password, &u.RoleId, &u.Active, &u.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	r.populateUserRoles(ctx, &u)
	return &u, nil
}

func (r *repository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, username, email, password, role_id, active, created_at FROM users WHERE username = $1`

	var u model.User
	err = db.QueryRowContext(ctx, query, username).Scan(
		&u.Id, &u.Username, &u.Email, &u.Password, &u.RoleId, &u.Active, &u.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	r.populateUserRoles(ctx, &u)
	return &u, nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, username, email, password, role_id, active, created_at FROM users WHERE email = $1`

	var u model.User
	err = db.QueryRowContext(ctx, query, email).Scan(
		&u.Id, &u.Username, &u.Email, &u.Password, &u.RoleId, &u.Active, &u.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	r.populateUserRoles(ctx, &u)
	return &u, nil
}

func (r *repository) populateUserRoles(ctx context.Context, u *model.User) {
	db, err := r.getDb()
	if err != nil {
		return
	}
	rows, err := db.QueryContext(ctx, `SELECT role_id FROM user_roles WHERE user_id = $1`, u.Id)
	if err != nil {
		if u.RoleId != "" {
			u.Roles = []string{u.RoleId}
		}
		return
	}
	defer rows.Close()

	u.Roles = make([]string, 0)
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err == nil {
			u.Roles = append(u.Roles, roleID)
		}
	}
	if len(u.Roles) == 0 && u.RoleId != "" {
		u.Roles = []string{u.RoleId}
	}

	// Also populate dynamic effective capabilities
	if caps, err := r.GetUserEffectivePermissions(ctx, u.Id); err == nil {
		u.Capabilities = caps
	}
}

func (r *repository) CreateUser(ctx context.Context, user *model.User) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `INSERT INTO users (id, username, email, password, role_id, active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.ExecContext(ctx, query,
		user.Id, user.Username, user.Email, user.Password,
		user.RoleId, user.Active, user.CreatedAt,
	)
	if err != nil {
		return ClassifyDBError(err)
	}

	// Insert into user_roles
	rolesToInsert := user.Roles
	if len(rolesToInsert) == 0 && user.RoleId != "" {
		rolesToInsert = []string{user.RoleId}
	}
	for _, roleID := range rolesToInsert {
		if roleID != "" {
			_, _ = db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, user.Id, roleID)
		}
	}

	return nil
}

func (r *repository) UpdateUser(ctx context.Context, user *model.User) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE users SET email = $1, role_id = $2, active = $3 WHERE id = $4`

	_, err = db.ExecContext(ctx, query, user.Email, user.RoleId, user.Active, user.Id)
	if err != nil {
		return ClassifyDBError(err)
	}

	if user.RoleId != "" {
		_, _ = db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, user.Id, user.RoleId)
	}

	return nil
}

func (r *repository) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE users SET password = $1 WHERE id = $2`
	_, err = db.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return ClassifyDBError(err)
	}
	return nil
}

func (r *repository) ActivateUser(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE users SET active = $1 WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}

func (r *repository) HasPermission(ctx context.Context, userId string, permission string) (bool, error) {
	db, err := r.getDb()
	if err != nil {
		return false, err
	}

	query := `
		SELECT COUNT(*)
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN users u ON u.id = $1 AND u.active = TRUE
		JOIN (
			SELECT role_id FROM user_roles WHERE user_id = $1
			UNION
			SELECT role_id FROM users WHERE id = $1 AND role_id IS NOT NULL AND role_id != ''
		) ur ON ur.role_id = rp.role_id
		JOIN roles r ON r.id = ur.role_id
		WHERE p.id = $2 AND p.active = TRUE AND rp.active = TRUE AND r.active = TRUE`

	var count int
	err = db.QueryRowContext(ctx, query, userId, permission).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *repository) GetRolePermissions(ctx context.Context, roleId string) ([]string, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT rp.permission_id
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		JOIN roles r ON r.id = rp.role_id
		WHERE rp.role_id = $1 AND rp.active = TRUE AND p.active = TRUE AND r.active = TRUE`

	rows, err := db.QueryContext(ctx, query, roleId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *repository) GetUserEffectivePermissions(ctx context.Context, userId string) ([]string, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT DISTINCT p.id
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN (
			SELECT role_id FROM user_roles WHERE user_id = $1
			UNION
			SELECT role_id FROM users WHERE id = $1 AND role_id IS NOT NULL AND role_id != ''
		) ur ON ur.role_id = rp.role_id
		JOIN roles r ON r.id = ur.role_id
		WHERE p.active = TRUE AND rp.active = TRUE AND r.active = TRUE
		ORDER BY p.id ASC`

	rows, err := db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, ClassifyDBError(err)
	}
	defer rows.Close()

	perms := make([]string, 0)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, ClassifyDBError(err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *repository) GetUserRoles(ctx context.Context, userId string) ([]string, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT DISTINCT sub.role_id FROM (
			SELECT role_id FROM user_roles WHERE user_id = $1
			UNION
			SELECT role_id FROM users WHERE id = $1 AND role_id IS NOT NULL AND role_id != ''
		) sub
		JOIN roles r ON r.id = sub.role_id
		WHERE r.active = TRUE
		ORDER BY sub.role_id ASC`

	rows, err := db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, ClassifyDBError(err)
	}
	defer rows.Close()

	roles := make([]string, 0)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, ClassifyDBError(err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}
