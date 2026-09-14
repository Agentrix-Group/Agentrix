package repository

import (
	"context"
	"database/sql"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
)

// ContestReader defines read-only repository operations for contests.
type ContestReader interface {
	ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	ListContests(ctx context.Context) ([]model.Contest, error)
	GetContest(ctx context.Context, id string) (*model.Contest, error)
}

// ContestWriter defines write operations for contests.
type ContestWriter interface {
	CreateContest(ctx context.Context, contest *model.Contest) error
	UpdateContest(ctx context.Context, contest *model.Contest) error
	ActivateContest(ctx context.Context, id string, isActive bool) error
}

// ContestRepository aggregates contest operations under ATD-015.
type ContestRepository interface {
	ContestReader
	ContestWriter
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

	tracer.Debugf(ctx, "Executing PostgreSQL query for public contests (state='%s', include_archived=%t)", filter.State, filter.IncludeArchived)

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
		tracer.Errorf(ctx, "PostgreSQL query failed for public contests: %s", err)
		return nil, err
	}
	defer rows.Close()

	contests := make([]model.PublicContestSummary, 0)
	for rows.Next() {
		var c model.PublicContestSummary
		var stateStr string
		var startsAt, endsAt sql.NullTime

		if err := rows.Scan(&c.Id, &c.Name, &c.Description, &stateStr, &startsAt, &endsAt); err != nil {
			tracer.Errorf(ctx, "Failed to scan public contest row: %s", err)
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
		tracer.Errorf(ctx, "Rows iteration error for public contests: %s", err)
		return nil, err
	}

	tracer.Debugf(ctx, "Retrieved %d public contests from PostgreSQL", len(contests))
	return contests, nil
}

func (r *repository) ListContests(ctx context.Context) ([]model.Contest, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing PostgreSQL query to fetch all contests")
	query := `SELECT id, name, description, game_id, category_id, starts_at, ends_at, state, active, created_at FROM contests ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing contests: %s", err)
		return nil, err
	}
	defer rows.Close()

	contests := make([]model.Contest, 0)
	for rows.Next() {
		var c model.Contest
		var stateStr string
		var gameId, categoryId sql.NullString
		var startsAt, endsAt sql.NullTime

		if err := rows.Scan(&c.Id, &c.Name, &c.Description, &gameId, &categoryId, &startsAt, &endsAt, &stateStr, &c.Active, &c.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for contests: %s", err)
			return nil, err
		}

		c.State = model.ContestState(stateStr)
		c.GameId = gameId.String
		c.CategoryId = categoryId.String
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
		tracer.Errorf(ctx, "Rows error after listing contests: %s", err)
		return nil, err
	}

	tracer.Debugf(ctx, "Retrieved %d contests from database", len(contests))
	return contests, nil
}

func (r *repository) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying PostgreSQL for contest %s", id)
	query := `SELECT id, name, description, game_id, category_id, starts_at, ends_at, state, active, created_at FROM contests WHERE id = $1`

	var c model.Contest
	var stateStr string
	var gameId, categoryId sql.NullString
	var startsAt, endsAt sql.NullTime

	err = db.QueryRowContext(ctx, query, id).Scan(
		&c.Id, &c.Name, &c.Description, &gameId, &categoryId, &startsAt, &endsAt, &stateStr, &c.Active, &c.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Contest %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for contest %s: %s", id, err)
		return nil, err
	}

	c.State = model.ContestState(stateStr)
	c.GameId = gameId.String
	c.CategoryId = categoryId.String
	if startsAt.Valid {
		c.StartsAt = &startsAt.Time
		c.StartDate = startsAt.Time
	}
	if endsAt.Valid {
		c.EndsAt = &endsAt.Time
		c.EndDate = endsAt.Time
	}

	tracer.Debugf(ctx, "Successfully retrieved contest %s", id)
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

	tracer.Debugf(ctx, "Creating contest '%s' in PostgreSQL", contest.Id)
	query := `INSERT INTO contests (id, name, description, game_id, category_id, state, active, starts_at, ends_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = db.ExecContext(ctx, query,
		contest.Id, contest.Name, contest.Description, contest.GameId,
		contest.CategoryId, string(state), contest.Active, contest.StartsAt,
		contest.EndsAt, contest.CreatedAt, contest.CreatedAt,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database insert failed for contest %s: %s", contest.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully created contest %s", contest.Id)
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

	tracer.Debugf(ctx, "Updating contest '%s' in PostgreSQL", contest.Id)
	query := `UPDATE contests SET name = $1, description = $2, game_id = $3, category_id = $4, state = $5, active = $6, starts_at = $7, ends_at = $8, updated_at = CURRENT_TIMESTAMP WHERE id = $9`

	_, err = db.ExecContext(ctx, query,
		contest.Name, contest.Description, contest.GameId, contest.CategoryId,
		string(state), contest.Active, contest.StartsAt, contest.EndsAt, contest.Id,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database update failed for contest %s: %s", contest.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully updated contest %s", contest.Id)
	return nil
}

func (r *repository) ActivateContest(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Setting contest '%s' active status to %t in PostgreSQL", id, isActive)
	query := `UPDATE contests SET active = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		tracer.Errorf(ctx, "Database activation update failed for contest %s: %s", id, err)
		return err
	}
	return nil
}

func (r *repository) ListCategories(ctx context.Context) ([]model.Category, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing database query to fetch active categories")
	query := `SELECT id, description, active FROM categories WHERE active = TRUE`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing categories: %s", err)
		return nil, err
	}
	defer rows.Close()

	categories := make([]model.Category, 0)
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.Id, &c.Description, &c.Active); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for category: %s", err)
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

	tracer.Debugf(ctx, "Querying database for category %s", id)
	query := `SELECT id, description, active FROM categories WHERE id = $1 AND active = TRUE`

	var c model.Category
	err = db.QueryRowContext(ctx, query, id).Scan(&c.Id, &c.Description, &c.Active)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Category %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for category %s: %s", id, err)
		return nil, err
	}
	return &c, nil
}

func (r *repository) CreateCategory(ctx context.Context, category *model.Category) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Creating category '%s'", category.Id)
	query := `INSERT INTO categories (id, description, active) VALUES ($1, $2, $3)`

	_, err = db.ExecContext(ctx, query, category.Id, category.Description, category.Active)
	if err != nil {
		tracer.Errorf(ctx, "Database insert failed for category %s: %s", category.Id, err)
		return err
	}
	return nil
}

func (r *repository) UpdateCategory(ctx context.Context, category *model.Category) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Updating category '%s'", category.Id)
	query := `UPDATE categories SET description = $1, active = $2 WHERE id = $3`

	_, err = db.ExecContext(ctx, query, category.Description, category.Active, category.Id)
	if err != nil {
		tracer.Errorf(ctx, "Database update failed for category %s: %s", category.Id, err)
		return err
	}
	return nil
}

func (r *repository) ActivateCategory(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Setting category '%s' active status to %t", id, isActive)
	query := `UPDATE categories SET active = $1 WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}
