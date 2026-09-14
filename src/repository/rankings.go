package repository

import (
	"context"
	"database/sql"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
)

func (r *repository) ListRankings(ctx context.Context) ([]model.Ranking, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing database query to fetch rankings")
	query := `SELECT id, contest_id, agent_id, participant_id, score, matches_played, wins, losses, draws, "rank", updated_at FROM rankings ORDER BY "rank" ASC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing rankings: %s", err)
		return nil, err
	}
	defer rows.Close()

	var rankings []model.Ranking
	for rows.Next() {
		var rk model.Ranking
		if err := rows.Scan(&rk.Id, &rk.ContestId, &rk.AgentId, &rk.ParticipantId, &rk.Score, &rk.MatchesPlayed, &rk.Wins, &rk.Losses, &rk.Draws, &rk.Rank, &rk.UpdatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for rankings: %s", err)
			return nil, err
		}
		rankings = append(rankings, rk)
	}

	tracer.Debugf(ctx, "Retrieved %d rankings from database", len(rankings))
	return rankings, nil
}

func (r *repository) ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying rankings for contest %s", contestId)
	query := `SELECT id, contest_id, agent_id, participant_id, score, matches_played, wins, losses, draws, "rank", updated_at FROM rankings WHERE contest_id = $1 ORDER BY "rank" ASC`

	rows, err := db.QueryContext(ctx, query, contestId)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for contest rankings: %s", err)
		return nil, err
	}
	defer rows.Close()

	var rankings []model.Ranking
	for rows.Next() {
		var rk model.Ranking
		if err := rows.Scan(&rk.Id, &rk.ContestId, &rk.AgentId, &rk.ParticipantId, &rk.Score, &rk.MatchesPlayed, &rk.Wins, &rk.Losses, &rk.Draws, &rk.Rank, &rk.UpdatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for rankings: %s", err)
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

	tracer.Debugf(ctx, "Querying database for ranking %s", id)
	query := `SELECT id, contest_id, agent_id, participant_id, score, matches_played, wins, losses, draws, "rank", updated_at FROM rankings WHERE id = $1`

	var rk model.Ranking
	err = db.QueryRowContext(ctx, query, id).Scan(
		&rk.Id, &rk.ContestId, &rk.AgentId, &rk.ParticipantId, &rk.Score, &rk.MatchesPlayed, &rk.Wins, &rk.Losses, &rk.Draws, &rk.Rank, &rk.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Ranking %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for ranking %s: %s", id, err)
		return nil, err
	}

	return &rk, nil
}

func (r *repository) GetRankingByContestAndAgent(ctx context.Context, contestId, agentId string) (*model.Ranking, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying ranking for contest %s and agent %s", contestId, agentId)
	query := `SELECT id, contest_id, agent_id, participant_id, score, matches_played, wins, losses, draws, "rank", updated_at FROM rankings WHERE contest_id = $1 AND agent_id = $2`

	var rk model.Ranking
	err = db.QueryRowContext(ctx, query, contestId, agentId).Scan(
		&rk.Id, &rk.ContestId, &rk.AgentId, &rk.ParticipantId, &rk.Score, &rk.MatchesPlayed, &rk.Wins, &rk.Losses, &rk.Draws, &rk.Rank, &rk.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Debugf(ctx, "No ranking found for contest %s and agent %s", contestId, agentId)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for contest/agent ranking: %s", err)
		return nil, err
	}

	return &rk, nil
}

func (r *repository) UpsertRanking(ctx context.Context, ranking *model.Ranking) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Upserting ranking for contest '%s' agent '%s'", ranking.ContestId, ranking.AgentId)
	query := `
		INSERT INTO rankings (id, contest_id, agent_id, participant_id, score, matches_played, wins, losses, draws, "rank", updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (contest_id, agent_id) DO UPDATE SET
			score = EXCLUDED.score,
			matches_played = EXCLUDED.matches_played,
			wins = EXCLUDED.wins,
			losses = EXCLUDED.losses,
			draws = EXCLUDED.draws,
			"rank" = EXCLUDED."rank",
			updated_at = EXCLUDED.updated_at`

	_, err = db.ExecContext(ctx, query,
		ranking.Id, ranking.ContestId, ranking.AgentId, ranking.ParticipantId,
		ranking.Score, ranking.MatchesPlayed, ranking.Wins, ranking.Losses,
		ranking.Draws, ranking.Rank, ranking.UpdatedAt,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database upsert failed for ranking: %s", err)
		return err
	}

	return nil
}
