package repository

import (
	"context"
	"database/sql"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
)

func (r *repository) ListResults(ctx context.Context) ([]model.Result, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing database query to fetch results")
	query := `SELECT id, match_id, submission_id, score, "rank", status, details, created_at FROM results ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing results: %s", err)
		return nil, err
	}
	defer rows.Close()

	var results []model.Result
	for rows.Next() {
		var res model.Result
		if err := rows.Scan(&res.Id, &res.MatchId, &res.SubmissionId, &res.Score, &res.Rank, &res.Status, &res.Details, &res.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for results: %s", err)
			return nil, err
		}
		results = append(results, res)
	}

	tracer.Debugf(ctx, "Retrieved %d results from database", len(results))
	return results, nil
}

func (r *repository) ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying results for match %s", matchId)
	query := `SELECT id, match_id, submission_id, score, "rank", status, details, created_at FROM results WHERE match_id = $1 ORDER BY "rank" ASC`

	rows, err := db.QueryContext(ctx, query, matchId)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for match results: %s", err)
		return nil, err
	}
	defer rows.Close()

	var results []model.Result
	for rows.Next() {
		var res model.Result
		if err := rows.Scan(&res.Id, &res.MatchId, &res.SubmissionId, &res.Score, &res.Rank, &res.Status, &res.Details, &res.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed: %s", err)
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

func (r *repository) GetResult(ctx context.Context, id string) (*model.Result, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying database for result %s", id)
	query := `SELECT id, match_id, submission_id, score, "rank", status, details, created_at FROM results WHERE id = $1`

	var res model.Result
	err = db.QueryRowContext(ctx, query, id).Scan(
		&res.Id, &res.MatchId, &res.SubmissionId, &res.Score, &res.Rank, &res.Status, &res.Details, &res.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Result %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for result %s: %s", id, err)
		return nil, err
	}

	return &res, nil
}

func (r *repository) CreateResult(ctx context.Context, result *model.Result) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Creating result '%s' for match '%s'", result.Id, result.MatchId)
	query := `INSERT INTO results (id, match_id, submission_id, score, "rank", status, details, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = db.ExecContext(ctx, query,
		result.Id, result.MatchId, result.SubmissionId, result.Score,
		result.Rank, result.Status, result.Details, result.CreatedAt,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database insert failed for result %s: %s", result.Id, err)
		return err
	}

	return nil
}
