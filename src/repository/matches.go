package repository

import (
	"context"
	"database/sql"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
)

func (r *repository) ListMatches(ctx context.Context) ([]model.Match, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing database query to fetch active matches")
	query := `SELECT id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at FROM matches WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing matches: %s", err)
		return nil, err
	}
	defer rows.Close()

	var matches []model.Match
	for rows.Next() {
		var m model.Match
		if err := rows.Scan(&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for matches: %s", err)
			return nil, err
		}
		matches = append(matches, m)
	}

	tracer.Debugf(ctx, "Retrieved %d matches from database", len(matches))
	return matches, nil
}

func (r *repository) ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying matches for contest %s", contestId)
	query := `SELECT id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at FROM matches WHERE contest_id = ? AND active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query, contestId)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for contest matches: %s", err)
		return nil, err
	}
	defer rows.Close()

	var matches []model.Match
	for rows.Next() {
		var m model.Match
		if err := rows.Scan(&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed: %s", err)
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

func (r *repository) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying database for match %s", id)
	query := `SELECT id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at FROM matches WHERE id = ? AND active = TRUE`

	var m model.Match
	err = db.QueryRowContext(ctx, query, id).Scan(
		&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Match %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for match %s: %s", id, err)
		return nil, err
	}

	tracer.Debugf(ctx, "Successfully retrieved match %s", id)
	return &m, nil
}

func (r *repository) CreateMatch(ctx context.Context, match *model.Match) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Creating match '%s'", match.Id)
	query := `INSERT INTO matches (id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.ExecContext(ctx, query,
		match.Id, match.ContestId, match.GameId, match.Status,
		match.Seed, match.ReplayId, match.Active, match.CreatedAt, match.FinishedAt,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database insert failed for match %s: %s", match.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully created match %s", match.Id)
	return nil
}

func (r *repository) UpdateMatch(ctx context.Context, match *model.Match) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Updating match '%s'", match.Id)
	query := `UPDATE matches SET status = ?, replay_id = ?, finished_at = ?, active = ? WHERE id = ?`

	_, err = db.ExecContext(ctx, query, match.Status, match.ReplayId, match.FinishedAt, match.Active, match.Id)
	if err != nil {
		tracer.Errorf(ctx, "Database update failed for match %s: %s", match.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully updated match %s", match.Id)
	return nil
}

func (r *repository) ActivateMatch(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Setting match '%s' active status to %t", id, isActive)
	query := `UPDATE matches SET active = ? WHERE id = ?`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}
