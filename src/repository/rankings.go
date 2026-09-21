package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func (r *repository) ListRankings(ctx context.Context) ([]model.Ranking, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, contest_id, agent_id, user_id, score, points, matches_played, wins, losses, draws, disqualifications, tiebreaker_score, "rank", updated_at FROM rankings ORDER BY "rank" ASC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rankings := make([]model.Ranking, 0)
	for rows.Next() {
		var rk model.Ranking
		if err := rows.Scan(&rk.Id, &rk.ContestId, &rk.AgentId, &rk.UserId, &rk.Score, &rk.Points, &rk.MatchesPlayed, &rk.Wins, &rk.Losses, &rk.Draws, &rk.Disqualifications, &rk.TiebreakerScore, &rk.Rank, &rk.UpdatedAt); err != nil {
			return nil, err
		}
		rankings = append(rankings, rk)
	}

	return rankings, nil
}

func (r *repository) ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, contest_id, agent_id, user_id, score, points, matches_played, wins, losses, draws, disqualifications, tiebreaker_score, "rank", updated_at FROM rankings WHERE contest_id = $1 ORDER BY "rank" ASC`

	rows, err := db.QueryContext(ctx, query, contestId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rankings := make([]model.Ranking, 0)
	for rows.Next() {
		var rk model.Ranking
		if err := rows.Scan(&rk.Id, &rk.ContestId, &rk.AgentId, &rk.UserId, &rk.Score, &rk.Points, &rk.MatchesPlayed, &rk.Wins, &rk.Losses, &rk.Draws, &rk.Disqualifications, &rk.TiebreakerScore, &rk.Rank, &rk.UpdatedAt); err != nil {
			return nil, err
		}
		rankings = append(rankings, rk)
	}

	return rankings, nil
}

func (r *repository) GetRanking(ctx context.Context, id string) (*model.Ranking, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, contest_id, agent_id, user_id, score, points, matches_played, wins, losses, draws, disqualifications, tiebreaker_score, "rank", updated_at FROM rankings WHERE id = $1`

	var rk model.Ranking
	err = db.QueryRowContext(ctx, query, id).Scan(
		&rk.Id, &rk.ContestId, &rk.AgentId, &rk.UserId, &rk.Score, &rk.Points, &rk.MatchesPlayed, &rk.Wins, &rk.Losses, &rk.Draws, &rk.Disqualifications, &rk.TiebreakerScore, &rk.Rank, &rk.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &rk, nil
}

func (r *repository) GetRankingByContestAndAgent(ctx context.Context, contestId, agentId string) (*model.Ranking, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, contest_id, agent_id, user_id, score, points, matches_played, wins, losses, draws, disqualifications, tiebreaker_score, "rank", updated_at FROM rankings WHERE contest_id = $1 AND agent_id = $2`

	var rk model.Ranking
	err = db.QueryRowContext(ctx, query, contestId, agentId).Scan(
		&rk.Id, &rk.ContestId, &rk.AgentId, &rk.UserId, &rk.Score, &rk.Points, &rk.MatchesPlayed, &rk.Wins, &rk.Losses, &rk.Draws, &rk.Disqualifications, &rk.TiebreakerScore, &rk.Rank, &rk.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &rk, nil
}

func (r *repository) UpsertRanking(ctx context.Context, ranking *model.Ranking) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	if ranking.UpdatedAt.IsZero() {
		ranking.UpdatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO rankings (id, contest_id, agent_id, user_id, score, points, matches_played, wins, losses, draws, disqualifications, tiebreaker_score, "rank", updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (contest_id, agent_id)
		DO UPDATE SET
			user_id = EXCLUDED.user_id,
			score = EXCLUDED.score,
			points = EXCLUDED.points,
			matches_played = EXCLUDED.matches_played,
			wins = EXCLUDED.wins,
			losses = EXCLUDED.losses,
			draws = EXCLUDED.draws,
			disqualifications = EXCLUDED.disqualifications,
			tiebreaker_score = EXCLUDED.tiebreaker_score,
			"rank" = EXCLUDED.rank,
			updated_at = EXCLUDED.updated_at
	`

	_, err = db.ExecContext(ctx, query,
		ranking.Id, ranking.ContestId, ranking.AgentId, ranking.UserId,
		ranking.Score, ranking.Points, ranking.MatchesPlayed, ranking.Wins,
		ranking.Losses, ranking.Draws, ranking.Disqualifications, ranking.TiebreakerScore,
		ranking.Rank, ranking.UpdatedAt,
	)
	return err
}

func (r *repository) ResetContestRankings(ctx context.Context, contestId string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `
		UPDATE rankings
		SET score = 0, points = 0, matches_played = 0, wins = 0, losses = 0, draws = 0,
		    disqualifications = 0, tiebreaker_score = 0, "rank" = 1, updated_at = NOW()
		WHERE contest_id = $1
	`
	_, err = db.ExecContext(ctx, query, contestId)
	return err
}

func (r *repository) IsRunAppliedToRanking(ctx context.Context, contestId, runId string) (bool, error) {
	db, err := r.getDb()
	if err != nil {
		return false, err
	}

	var exists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM ranking_applied_runs WHERE contest_id = $1 AND run_id = $2
		)
	`, contestId, runId).Scan(&exists)
	return exists, err
}

func (r *repository) RecordAppliedRun(ctx context.Context, contestId, runId, matchId string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO ranking_applied_runs (contest_id, run_id, match_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (contest_id, run_id) DO NOTHING
	`, contestId, runId, matchId)
	return err
}

func (r *repository) ClearAppliedRuns(ctx context.Context, contestId string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `DELETE FROM ranking_applied_runs WHERE contest_id = $1`, contestId)
	return err
}

func (r *repository) CreateRankingSnapshot(ctx context.Context, snapshot *model.RankingSnapshot) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	snapshotJSON, err := json.Marshal(snapshot.Rankings)
	if err != nil {
		return fmt.Errorf("serialize snapshot rankings: %w", err)
	}

	query := `
		INSERT INTO contest_rankings_snapshots (id, contest_id, version, snapshot_data, published_by, published_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = db.ExecContext(ctx, query, snapshot.Id, snapshot.ContestId, snapshot.Version, snapshotJSON, snapshot.PublishedBy, snapshot.PublishedAt)
	return err
}

func (r *repository) ListRankingSnapshots(ctx context.Context, contestId string) ([]model.RankingSnapshot, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, contest_id, version, snapshot_data, published_by, published_at
		FROM contest_rankings_snapshots
		WHERE contest_id = $1
		ORDER BY version DESC
	`

	rows, err := db.QueryContext(ctx, query, contestId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []model.RankingSnapshot
	for rows.Next() {
		var s model.RankingSnapshot
		var rawData []byte
		var pubBy sql.NullString
		if err := rows.Scan(&s.Id, &s.ContestId, &s.Version, &rawData, &pubBy, &s.PublishedAt); err != nil {
			return nil, err
		}
		if pubBy.Valid {
			s.PublishedBy = &pubBy.String
		}
		if err := json.Unmarshal(rawData, &s.Rankings); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, nil
}

func (r *repository) GetLatestRankingSnapshot(ctx context.Context, contestId string) (*model.RankingSnapshot, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, contest_id, version, snapshot_data, published_by, published_at
		FROM contest_rankings_snapshots
		WHERE contest_id = $1
		ORDER BY version DESC
		LIMIT 1
	`

	var s model.RankingSnapshot
	var rawData []byte
	var pubBy sql.NullString
	err = db.QueryRowContext(ctx, query, contestId).Scan(&s.Id, &s.ContestId, &s.Version, &rawData, &pubBy, &s.PublishedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if pubBy.Valid {
		s.PublishedBy = &pubBy.String
	}
	if err := json.Unmarshal(rawData, &s.Rankings); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) GetRankingSnapshotByVersion(ctx context.Context, contestId string, version int) (*model.RankingSnapshot, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, contest_id, version, snapshot_data, published_by, published_at
		FROM contest_rankings_snapshots
		WHERE contest_id = $1 AND version = $2
	`

	var s model.RankingSnapshot
	var rawData []byte
	var pubBy sql.NullString
	err = db.QueryRowContext(ctx, query, contestId, version).Scan(&s.Id, &s.ContestId, &s.Version, &rawData, &pubBy, &s.PublishedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if pubBy.Valid {
		s.PublishedBy = &pubBy.String
	}
	if err := json.Unmarshal(rawData, &s.Rankings); err != nil {
		return nil, err
	}
	return &s, nil
}
