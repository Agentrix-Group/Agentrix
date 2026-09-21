package repository

import (
	"context"
	"database/sql"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

// ContestReader defines read-only repository operations for contests.
type ContestReader interface {
	ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	ListContests(ctx context.Context) ([]model.Contest, error)
	GetContest(ctx context.Context, id string) (*model.Contest, error)
	GetContestEntryBySubmission(ctx context.Context, contestId, submissionId string) (*model.ContestEntry, error)
}

// ContestWriter defines write operations for contests.
type ContestWriter interface {
	CreateContest(ctx context.Context, contest *model.Contest) error
	UpdateContest(ctx context.Context, contest *model.Contest) error
	ActivateContest(ctx context.Context, id string, isActive bool) error
}

// ContestCASRepository defines optimistic concurrency operations for contests.
type ContestCASRepository interface {
	UpdateContestStateCAS(ctx context.Context, contestId string, expectedState, newState model.ContestState) (bool, error)
}

// ContestRepository aggregates contest operations under ATD-015.
type ContestRepository interface {
	ContestReader
	ContestWriter
	ContestCASRepository
}

// ListPublicContests queries PostgreSQL for public contests adhering to blueprint visibility rules:
// - Never returns 'draft' contests.
// - Excludes 'archived' contests unless filter.IncludeArchived is true.
// - Supports filtering by a specific valid public state.
// - Guaranteed to return an empty slice (not nil) when 0 records match.
func (r *repository) ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	var query string
	var args []any

	if filter.State != "" {
		query = `SELECT id, name, description, state, starts_at, ends_at FROM contests WHERE state != 'draft' AND state = $1 ORDER BY created_at DESC`
		args = append(args, string(filter.State))
	} else if filter.IncludeArchived {
		query = `SELECT id, name, description, state, starts_at, ends_at FROM contests WHERE state != 'draft' ORDER BY created_at DESC`
	} else {
		query = `SELECT id, name, description, state, starts_at, ends_at FROM contests WHERE state != 'draft' AND state != 'archived' ORDER BY created_at DESC`
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contests := make([]model.PublicContestSummary, 0)
	for rows.Next() {
		var c model.PublicContestSummary
		var stateStr string
		var startsAt, endsAt sql.NullTime

		if err := rows.Scan(&c.Id, &c.Name, &c.Description, &stateStr, &startsAt, &endsAt); err != nil {
			return nil, err
		}

		c.State = model.ContestState(stateStr)
		if startsAt.Valid {
			c.StartsAt = &startsAt.Time
		}
		if endsAt.Valid {
			c.EndsAt = &endsAt.Time
		}

		contests = append(contests, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return contests, nil
}

func (r *repository) ListContests(ctx context.Context) ([]model.Contest, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, name, description, game_id, category_id, starts_at, ends_at, state, active, scoring_policy, created_at FROM contests ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contests := make([]model.Contest, 0)
	for rows.Next() {
		var c model.Contest
		var stateStr string
		var gameId, categoryId sql.NullString
		var startsAt, endsAt sql.NullTime
		var scoringPolicy model.ScoringPolicy

		if err := rows.Scan(&c.Id, &c.Name, &c.Description, &gameId, &categoryId, &startsAt, &endsAt, &stateStr, &c.Active, &scoringPolicy, &c.CreatedAt); err != nil {
			return nil, err
		}

		c.State = model.ContestState(stateStr)
		c.GameId = gameId.String
		c.CategoryId = categoryId.String
		c.ScoringPolicy = &scoringPolicy
		if startsAt.Valid {
			c.StartsAt = &startsAt.Time
			c.StartDate = startsAt.Time
		}
		if endsAt.Valid {
			c.EndsAt = &endsAt.Time
			c.EndDate = endsAt.Time
		}
		contests = append(contests, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return contests, nil
}

func (r *repository) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, name, description, game_id, category_id, starts_at, ends_at, state, active, scoring_policy, created_at FROM contests WHERE id = $1`

	var c model.Contest
	var stateStr string
	var gameId, categoryId sql.NullString
	var startsAt, endsAt sql.NullTime
	var scoringPolicy model.ScoringPolicy

	err = db.QueryRowContext(ctx, query, id).Scan(
		&c.Id, &c.Name, &c.Description, &gameId, &categoryId, &startsAt, &endsAt, &stateStr, &c.Active, &scoringPolicy, &c.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	c.State = model.ContestState(stateStr)
	c.GameId = gameId.String
	c.CategoryId = categoryId.String
	c.ScoringPolicy = &scoringPolicy
	if startsAt.Valid {
		c.StartsAt = &startsAt.Time
		c.StartDate = startsAt.Time
	}
	if endsAt.Valid {
		c.EndsAt = &endsAt.Time
		c.EndDate = endsAt.Time
	}

	return &c, nil
}

func (r *repository) CreateContest(ctx context.Context, contest *model.Contest) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	state := contest.State
	if state == "" {
		state = model.ContestStateDraft
	}

	var categoryID any = contest.CategoryId
	if contest.CategoryId == "" {
		categoryID = nil
	}

	if contest.ScoringPolicy == nil {
		def := model.DefaultScoringPolicy()
		contest.ScoringPolicy = &def
	}
	policyVal, _ := contest.ScoringPolicy.Value()

	query := `INSERT INTO contests (id, name, description, game_id, category_id, state, active, starts_at, ends_at, scoring_policy, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = db.ExecContext(ctx, query,
		contest.Id, contest.Name, contest.Description, contest.GameId,
		categoryID, string(state), contest.Active, contest.StartsAt,
		contest.EndsAt, policyVal, contest.CreatedAt, contest.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) UpdateContest(ctx context.Context, contest *model.Contest) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	state := contest.State
	if state == "" {
		state = model.ContestStateDraft
	}

	var categoryID any = contest.CategoryId
	if contest.CategoryId == "" {
		categoryID = nil
	}

	if contest.ScoringPolicy == nil {
		def := model.DefaultScoringPolicy()
		contest.ScoringPolicy = &def
	}
	policyVal, _ := contest.ScoringPolicy.Value()

	query := `UPDATE contests SET name = $1, description = $2, game_id = $3, category_id = $4, state = $5, active = $6, starts_at = $7, ends_at = $8, scoring_policy = $9, updated_at = CURRENT_TIMESTAMP WHERE id = $10`

	_, err = db.ExecContext(ctx, query,
		contest.Name, contest.Description, contest.GameId, categoryID,
		string(state), contest.Active, contest.StartsAt, contest.EndsAt, policyVal, contest.Id,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) ActivateContest(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE contests SET active = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) ListCategories(ctx context.Context) ([]model.Category, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, description, active FROM categories WHERE active = TRUE`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]model.Category, 0)
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.Id, &c.Description, &c.Active); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *repository) GetCategory(ctx context.Context, id string) (*model.Category, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, description, active FROM categories WHERE id = $1 AND active = TRUE`

	var c model.Category
	err = db.QueryRowContext(ctx, query, id).Scan(&c.Id, &c.Description, &c.Active)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &c, nil
}

func (r *repository) CreateCategory(ctx context.Context, category *model.Category) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `INSERT INTO categories (id, description, active) VALUES ($1, $2, $3)`

	_, err = db.ExecContext(ctx, query, category.Id, category.Description, category.Active)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) UpdateCategory(ctx context.Context, category *model.Category) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE categories SET description = $1, active = $2 WHERE id = $3`

	_, err = db.ExecContext(ctx, query, category.Description, category.Active, category.Id)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) ActivateCategory(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE categories SET active = $1 WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}

func (r *repository) CreateContestEntry(ctx context.Context, entry *model.ContestEntry) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `
		INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status, enrolled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.ExecContext(ctx, query,
		entry.Id, entry.ContestId, entry.AgentId, entry.UserId, entry.SubmissionId, entry.Status, entry.EnrolledAt,
	)
	if err != nil {
		return ClassifyDBError(err)
	}
	return nil
}

func (r *repository) ListContestEntries(ctx context.Context, contestId string) ([]model.ContestEntry, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT ce.id, ce.contest_id, ce.agent_id, ce.user_id, ce.submission_id, ce.status, ce.enrolled_at,
		       a.name, a.game_id, u.username
		FROM contest_entries ce
		JOIN agents a ON ce.agent_id = a.id
		JOIN users u ON ce.user_id = u.id
		WHERE ce.contest_id = $1
		ORDER BY ce.enrolled_at ASC`

	rows, err := db.QueryContext(ctx, query, contestId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.ContestEntry
	for rows.Next() {
		var entry model.ContestEntry
		var agentName, gameId, username string
		if err := rows.Scan(
			&entry.Id, &entry.ContestId, &entry.AgentId, &entry.UserId, &entry.SubmissionId, &entry.Status, &entry.EnrolledAt,
			&agentName, &gameId, &username,
		); err != nil {
			return nil, err
		}
		entry.Agent = &model.Agent{
			Id:          entry.AgentId,
			Name:        agentName,
			GameId:      gameId,
			OwnerUserId: entry.UserId,
		}
		entry.User = &model.User{
			Id:       entry.UserId,
			Username: username,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (r *repository) GetContestEntry(ctx context.Context, contestId, agentId string) (*model.ContestEntry, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT ce.id, ce.contest_id, ce.agent_id, ce.user_id, ce.submission_id, ce.status, ce.enrolled_at,
		       a.name, a.game_id, u.username
		FROM contest_entries ce
		JOIN agents a ON ce.agent_id = a.id
		JOIN users u ON ce.user_id = u.id
		WHERE ce.contest_id = $1 AND ce.agent_id = $2`

	var entry model.ContestEntry
	var agentName, gameId, username string
	err = db.QueryRowContext(ctx, query, contestId, agentId).Scan(
		&entry.Id, &entry.ContestId, &entry.AgentId, &entry.UserId, &entry.SubmissionId, &entry.Status, &entry.EnrolledAt,
		&agentName, &gameId, &username,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	entry.Agent = &model.Agent{
		Id:          entry.AgentId,
		Name:        agentName,
		GameId:      gameId,
		OwnerUserId: entry.UserId,
	}
	entry.User = &model.User{
		Id:       entry.UserId,
		Username: username,
	}

	return &entry, nil
}

func (r *repository) GetContestEntryBySubmission(ctx context.Context, contestId, submissionId string) (*model.ContestEntry, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT ce.id, ce.contest_id, ce.agent_id, ce.user_id, ce.submission_id, ce.status, ce.enrolled_at,
		       a.name, a.game_id, u.username
		FROM contest_entries ce
		JOIN agents a ON ce.agent_id = a.id
		JOIN users u ON ce.user_id = u.id
		WHERE ce.contest_id = $1 AND ce.submission_id = $2`

	var entry model.ContestEntry
	var agentName, gameId, username string
	err = db.QueryRowContext(ctx, query, contestId, submissionId).Scan(
		&entry.Id, &entry.ContestId, &entry.AgentId, &entry.UserId, &entry.SubmissionId, &entry.Status, &entry.EnrolledAt,
		&agentName, &gameId, &username,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	entry.Agent = &model.Agent{
		Id:          entry.AgentId,
		Name:        agentName,
		GameId:      gameId,
		OwnerUserId: entry.UserId,
	}
	entry.User = &model.User{
		Id:       entry.UserId,
		Username: username,
	}

	return &entry, nil
}

func (r *repository) UpdateContestStateCAS(ctx context.Context, contestId string, expectedState, newState model.ContestState) (bool, error) {
	db, err := r.getDb()
	if err != nil {
		return false, err
	}

	query := `UPDATE contests SET state = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND state = $3`
	res, err := db.ExecContext(ctx, query, string(newState), contestId, string(expectedState))
	if err != nil {
		return false, ClassifyDBError(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}
