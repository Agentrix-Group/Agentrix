package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

// LockContestRankings serializes ranking recomputations and snapshot
// publications of one contest for the rest of the transaction.
func (q *Queries) LockContestRankings(ctx context.Context, contestID string) error {
	_, err := q.db.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('rankings|' || $1, 0))`, contestID)
	return classify(err)
}

// BumpRankingDirty records, inside a commit transaction, that the ranking
// projection of the contest must be recomputed.
func (q *Queries) BumpRankingDirty(ctx context.Context, contestID string) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO contest_ranking_state (contest_id, dirty_version) VALUES ($1, 1)
		ON CONFLICT (contest_id) DO UPDATE SET dirty_version = contest_ranking_state.dirty_version + 1`, contestID)
	return classify(err)
}

func (q *Queries) GetRankingState(ctx context.Context, contestID string) (model.RankingState, error) {
	state := model.RankingState{ContestID: contestID, AppliedRunsDigest: zeroDigest}
	var computed sql.NullTime
	err := q.db.QueryRowContext(ctx, `SELECT dirty_version, computed_version, applied_runs_count, applied_runs_digest, computed_at
		FROM contest_ranking_state WHERE contest_id = $1`, contestID).Scan(
		&state.DirtyVersion, &state.ComputedVersion, &state.AppliedRunsCount, &state.AppliedRunsDigest, &computed)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	state.ComputedAt = timePtr(computed)
	return state, classify(err)
}

const zeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

func (q *Queries) DirtyRankingContests(ctx context.Context, limit int) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT contest_id FROM contest_ranking_state
		WHERE computed_version < dirty_version ORDER BY contest_id LIMIT $1`, limit)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// RankingEntries lists the entries that can appear in the projection.
func (q *Queries) RankingEntries(ctx context.Context, contestID string) ([]model.RankingEntry, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT e.id, e.agent_id, a.name, e.user_id, u.username, e.status, e.created_at
		FROM contest_entries e JOIN agents a ON a.id = e.agent_id JOIN users u ON u.id = e.user_id
		WHERE e.contest_id = $1 ORDER BY e.created_at, e.id`, contestID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.RankingEntry
	for rows.Next() {
		var e model.RankingEntry
		if err := rows.Scan(&e.EntryID, &e.AgentID, &e.AgentName, &e.UserID, &e.Username, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CommittedResults returns the slot results of the committed run of every
// finished competitive match of the contest. Results of other runs of the
// same match are never read.
func (q *Queries) CommittedResults(ctx context.Context, contestID string) ([]model.CommittedMatchResult, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT m.id, r.match_run_id, s.contest_entry_id, r.score, r.outcome
		FROM matches m
		JOIN results r ON r.match_run_id = m.committed_run_id
		JOIN match_slots s ON s.id = r.slot_id
		WHERE m.contest_id = $1 AND m.mode = 'competitive' AND m.state = 'finished'
		ORDER BY m.id, s.slot_index`, contestID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.CommittedMatchResult
	for rows.Next() {
		var r model.CommittedMatchResult
		if err := rows.Scan(&r.MatchID, &r.RunID, &r.EntryID, &r.Score, &r.Outcome); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReplaceRankings rewrites the projection, the applied-runs ledger and the
// watermark. Must run in the transaction holding LockContestRankings.
func (q *Queries) ReplaceRankings(ctx context.Context, contestID string, rankings []model.Ranking, appliedRuns map[string]string,
	digest string, coveredVersion int64, now time.Time) error {
	if _, err := q.db.ExecContext(ctx, `DELETE FROM rankings WHERE contest_id = $1`, contestID); err != nil {
		return classify(err)
	}
	for _, r := range rankings {
		if _, err := q.db.ExecContext(ctx, `
			INSERT INTO rankings (contest_id, entry_id, agent_id, user_id, rank, points, matches_played, wins, draws, losses,
				disqualifications, score_for, score_against)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			contestID, r.EntryID, r.AgentID, r.UserID, r.Rank, r.Points, r.MatchesPlayed, r.Wins, r.Draws, r.Losses,
			r.Disqualifications, r.ScoreFor, r.ScoreAgainst); err != nil {
			return classify(err)
		}
	}
	if _, err := q.db.ExecContext(ctx, `DELETE FROM ranking_applied_runs WHERE contest_id = $1`, contestID); err != nil {
		return classify(err)
	}
	for runID, matchID := range appliedRuns {
		if _, err := q.db.ExecContext(ctx, `INSERT INTO ranking_applied_runs (contest_id, run_id, match_id) VALUES ($1, $2, $3)`,
			contestID, runID, matchID); err != nil {
			return classify(err)
		}
	}
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO contest_ranking_state (contest_id, dirty_version, computed_version, applied_runs_count, applied_runs_digest, computed_at)
		VALUES ($1, $2, $2, $3, $4, $5)
		ON CONFLICT (contest_id) DO UPDATE SET computed_version = GREATEST(contest_ranking_state.computed_version, EXCLUDED.computed_version),
			applied_runs_count = EXCLUDED.applied_runs_count, applied_runs_digest = EXCLUDED.applied_runs_digest,
			computed_at = EXCLUDED.computed_at`,
		contestID, coveredVersion, len(appliedRuns), digest, now)
	return classify(err)
}

func (q *Queries) ListRankings(ctx context.Context, contestID string) ([]model.Ranking, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT r.contest_id, r.entry_id, r.agent_id, a.name, r.user_id, u.username, r.rank, r.points,
		r.matches_played, r.wins, r.draws, r.losses, r.disqualifications, r.score_for, r.score_against
		FROM rankings r JOIN agents a ON a.id = r.agent_id JOIN users u ON u.id = r.user_id JOIN contest_entries e ON e.id = r.entry_id
		WHERE r.contest_id = $1 ORDER BY r.rank, e.created_at, r.entry_id`, contestID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Ranking
	for rows.Next() {
		var r model.Ranking
		if err := rows.Scan(&r.ContestID, &r.EntryID, &r.AgentID, &r.AgentName, &r.UserID, &r.Username, &r.Rank, &r.Points,
			&r.MatchesPlayed, &r.Wins, &r.Draws, &r.Losses, &r.Disqualifications, &r.ScoreFor, &r.ScoreAgainst); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) NextSnapshotVersion(ctx context.Context, contestID string) (int, error) {
	var v int
	err := q.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM ranking_snapshots WHERE contest_id = $1`, contestID).Scan(&v)
	return v, classify(err)
}

func (q *Queries) InsertSnapshot(ctx context.Context, s *model.RankingSnapshot, rankingsJSON []byte) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO ranking_snapshots (id, contest_id, version, rankings, rankings_sha256, applied_runs_digest, applied_runs_count,
			published_by, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		s.ID, s.ContestID, s.Version, rankingsJSON, s.RankingsSHA256, s.AppliedRunsDigest, s.AppliedRunsCount, s.PublishedBy, s.PublishedAt)
	return classify(err)
}

const snapshotColumns = `id, contest_id, version, rankings, rankings_sha256, applied_runs_digest, applied_runs_count, published_by, published_at`

func scanSnapshot(row interface{ Scan(...any) error }) (*model.RankingSnapshot, error) {
	var s model.RankingSnapshot
	var raw []byte
	if err := row.Scan(&s.ID, &s.ContestID, &s.Version, &raw, &s.RankingsSHA256, &s.AppliedRunsDigest, &s.AppliedRunsCount,
		&s.PublishedBy, &s.PublishedAt); err != nil {
		return nil, err
	}
	var rows []model.SnapshotRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		s.Rankings = append(s.Rankings, row.Ranking(s.ContestID))
	}
	return &s, nil
}

func (q *Queries) ListSnapshots(ctx context.Context, contestID string) ([]model.RankingSnapshot, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+snapshotColumns+` FROM ranking_snapshots WHERE contest_id = $1 ORDER BY version`, contestID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.RankingSnapshot
	for rows.Next() {
		s, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (q *Queries) GetSnapshot(ctx context.Context, contestID string, version int) (*model.RankingSnapshot, error) {
	s, err := scanSnapshot(q.db.QueryRowContext(ctx, `SELECT `+snapshotColumns+` FROM ranking_snapshots
		WHERE contest_id = $1 AND version = $2`, contestID, version))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("snapshot_not_found", "ranking snapshot not found")
	}
	return s, classify(err)
}
