package repository

import (
	"context"
	"database/sql"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func (r *repository) ListMatches(ctx context.Context) ([]model.Match, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at FROM matches WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []model.Match
	for rows.Next() {
		var m model.Match
		if err := rows.Scan(&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}

	return matches, nil
}

func (r *repository) ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at FROM matches WHERE contest_id = $1 AND active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query, contestId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []model.Match
	for rows.Next() {
		var m model.Match
		if err := rows.Scan(&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt); err != nil {
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

	query := `SELECT id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at FROM matches WHERE id = $1 AND active = TRUE`

	var m model.Match
	err = db.QueryRowContext(ctx, query, id).Scan(
		&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &m, nil
}

func (r *repository) CreateMatch(ctx context.Context, match *model.Match) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `INSERT INTO matches (id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = db.ExecContext(ctx, query,
		match.Id, match.ContestId, match.GameId, match.Status,
		match.Seed, match.ReplayId, match.Active, match.CreatedAt, match.FinishedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) UpdateMatch(ctx context.Context, match *model.Match) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE matches SET status = $1, replay_id = $2, finished_at = $3, active = $4 WHERE id = $5`

	_, err = db.ExecContext(ctx, query, match.Status, match.ReplayId, match.FinishedAt, match.Active, match.Id)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) ActivateMatch(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE matches SET active = $1 WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}
