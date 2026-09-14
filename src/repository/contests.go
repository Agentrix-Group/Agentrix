package repository

import (
	"context"
	"database/sql"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
)

func (r *repository) ListContests(ctx context.Context) ([]model.Contest, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing database query to fetch all active contests")
	query := `SELECT id, name, description, game_id, category_id, start_date, end_date, status, active, created_at FROM contests WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing contests: %s", err)
		return nil, err
	}
	defer rows.Close()

	var contests []model.Contest
	for rows.Next() {
		var c model.Contest
		if err := rows.Scan(&c.Id, &c.Name, &c.Description, &c.GameId, &c.CategoryId, &c.StartDate, &c.EndDate, &c.Status, &c.Active, &c.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for contests: %s", err)
			return nil, err
		}
		contests = append(contests, c)
	}

	tracer.Debugf(ctx, "Retrieved %d contests from database", len(contests))
	return contests, nil
}

func (r *repository) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying database for contest %s", id)
	query := `SELECT id, name, description, game_id, category_id, start_date, end_date, status, active, created_at FROM contests WHERE id = ? AND active = TRUE`

	var c model.Contest
	err = db.QueryRowContext(ctx, query, id).Scan(
		&c.Id, &c.Name, &c.Description, &c.GameId, &c.CategoryId, &c.StartDate, &c.EndDate, &c.Status, &c.Active, &c.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Contest %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for contest %s: %s", id, err)
		return nil, err
	}

	tracer.Debugf(ctx, "Successfully retrieved contest %s", id)
	return &c, nil
}

func (r *repository) CreateContest(ctx context.Context, contest *model.Contest) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Creating contest '%s'", contest.Id)
	query := `INSERT INTO contests (id, name, description, game_id, category_id, start_date, end_date, status, active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.ExecContext(ctx, query,
		contest.Id, contest.Name, contest.Description, contest.GameId,
		contest.CategoryId, contest.StartDate, contest.EndDate, contest.Status,
		contest.Active, contest.CreatedAt,
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

	tracer.Debugf(ctx, "Updating contest '%s'", contest.Id)
	query := `UPDATE contests SET name = ?, description = ?, game_id = ?, category_id = ?, start_date = ?, end_date = ?, status = ?, active = ? WHERE id = ?`

	_, err = db.ExecContext(ctx, query,
		contest.Name, contest.Description, contest.GameId, contest.CategoryId,
		contest.StartDate, contest.EndDate, contest.Status, contest.Active, contest.Id,
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

	tracer.Debugf(ctx, "Setting contest '%s' active status to %t", id, isActive)
	query := `UPDATE contests SET active = ? WHERE id = ?`

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

	var categories []model.Category
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
	query := `SELECT id, description, active FROM categories WHERE id = ? AND active = TRUE`

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
	query := `INSERT INTO categories (id, description, active) VALUES (?, ?, ?)`

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
	query := `UPDATE categories SET description = ?, active = ? WHERE id = ?`

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
	query := `UPDATE categories SET active = ? WHERE id = ?`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}
