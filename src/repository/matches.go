package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/google/uuid"
)

func (r *repository) ListMatches(ctx context.Context) ([]model.Match, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, COALESCE(contest_id, ''), game_id, status, seed, COALESCE(replay_id, ''), active, created_at, finished_at, committed_run_id FROM matches WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []model.Match
	for rows.Next() {
		var m model.Match
		if err := rows.Scan(&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt, &m.CommittedRunId); err != nil {
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

	query := `SELECT id, COALESCE(contest_id, ''), game_id, status, seed, COALESCE(replay_id, ''), active, created_at, finished_at, committed_run_id FROM matches WHERE contest_id = $1 AND active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query, contestId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []model.Match
	for rows.Next() {
		var m model.Match
		if err := rows.Scan(&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt, &m.CommittedRunId); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// MatchSlotReader defines read operations for match slots.
type MatchSlotReader interface {
	ListMatchSlots(ctx context.Context, matchId string) ([]model.MatchSlot, error)
}

// MatchSlotWriter defines write operations for match slots.
type MatchSlotWriter interface {
	CreateMatchSlots(ctx context.Context, slots []model.MatchSlot) error
	CreateMatchWithSlots(ctx context.Context, match *model.Match, slots []model.MatchSlot) error
}

// MatchSlotRepository bundles slot operations.
type MatchSlotRepository interface {
	MatchSlotReader
	MatchSlotWriter
}

// MatchCASRepository defines optimistic concurrency operations for matches.
type MatchCASRepository interface {
	UpdateMatchStatusCAS(ctx context.Context, matchId string, expectedStatus, newStatus model.MatchStatus) (bool, error)
	// ClaimMatchForRun pasa la partida a queued y registra su run activo en
	// el mismo UPDATE: quien vea queued ve también el run.
	ClaimMatchForRun(ctx context.Context, matchId string, expectedStatus model.MatchStatus, runId string) (bool, error)
}

func (r *repository) ListMatchSlots(ctx context.Context, matchId string) ([]model.MatchSlot, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, match_id, slot_index, contest_entry_id, submission_id, agent_name, username, created_at
	FROM match_slots WHERE match_id = $1 ORDER BY slot_index ASC`

	rows, err := db.QueryContext(ctx, query, matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]model.MatchSlot, 0)
	for rows.Next() {
		var s model.MatchSlot
		if err := rows.Scan(
			&s.Id, &s.MatchId, &s.SlotIndex, &s.ContestEntryId,
			&s.SubmissionId, &s.AgentName, &s.Username, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, nil
}

func (r *repository) CreateMatchSlots(ctx context.Context, slots []model.MatchSlot) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `INSERT INTO match_slots (id, match_id, slot_index, contest_entry_id, submission_id, agent_name, username, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	for _, slot := range slots {
		_, err := db.ExecContext(ctx, query,
			slot.Id, slot.MatchId, slot.SlotIndex, slot.ContestEntryId,
			slot.SubmissionId, slot.AgentName, slot.Username, slot.CreatedAt,
		)
		if err != nil {
			return ClassifyDBError(err)
		}
	}
	return nil
}

func (r *repository) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, COALESCE(contest_id, ''), game_id, status, seed, COALESCE(replay_id, ''), active, created_at, finished_at, committed_run_id, COALESCE(run_id, '') FROM matches WHERE id = $1 AND active = TRUE`

	var m model.Match
	err = db.QueryRowContext(ctx, query, id).Scan(
		&m.Id, &m.ContestId, &m.GameId, &m.Status, &m.Seed, &m.ReplayId, &m.Active, &m.CreatedAt, &m.FinishedAt, &m.CommittedRunId, &m.RunId,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	slots, err := r.ListMatchSlots(ctx, id)
	if err == nil {
		m.Slots = slots
	}

	return &m, nil
}

func (r *repository) CreateMatch(ctx context.Context, match *model.Match) error {
	if len(match.Slots) > 0 {
		return r.CreateMatchWithSlots(ctx, match, match.Slots)
	}

	db, err := r.getDb()
	if err != nil {
		return err
	}

	var contestID any = match.ContestId
	if match.ContestId == "" {
		contestID = nil
	}

	query := `INSERT INTO matches (id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at, committed_run_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = db.ExecContext(ctx, query,
		match.Id, contestID, match.GameId, match.Status,
		match.Seed, match.ReplayId, match.Active, match.CreatedAt, match.FinishedAt, match.CommittedRunId,
	)
	if err != nil {
		return ClassifyDBError(err)
	}

	return nil
}

func (r *repository) CreateMatchWithSlots(ctx context.Context, match *model.Match, slots []model.MatchSlot) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin match tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var contestID any = match.ContestId
	if match.ContestId == "" {
		contestID = nil
	}

	query := `INSERT INTO matches (id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at, committed_run_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = tx.ExecContext(ctx, query,
		match.Id, contestID, match.GameId, match.Status,
		match.Seed, match.ReplayId, match.Active, match.CreatedAt, match.FinishedAt, match.CommittedRunId,
	)
	if err != nil {
		return ClassifyDBError(err)
	}

	insertSlotQuery := `INSERT INTO match_slots (id, match_id, slot_index, contest_entry_id, submission_id, agent_name, username, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	for _, slot := range slots {
		slotID := slot.Id
		if slotID == "" {
			slotID = uuid.New().String()
		}
		createdAt := slot.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		var entryID any = slot.ContestEntryId
		if slot.ContestEntryId != nil && *slot.ContestEntryId == "" {
			entryID = nil
		}

		_, err := tx.ExecContext(ctx, insertSlotQuery,
			slotID, match.Id, slot.SlotIndex, entryID,
			slot.SubmissionId, slot.AgentName, slot.Username, createdAt,
		)
		if err != nil {
			return fmt.Errorf("insert match slot %d: %w", slot.SlotIndex, ClassifyDBError(err))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit match tx: %w", err)
	}

	return nil
}

func (r *repository) UpdateMatchStatusCAS(ctx context.Context, matchId string, expectedStatus, newStatus model.MatchStatus) (bool, error) {
	db, err := r.getDb()
	if err != nil {
		return false, err
	}

	query := `UPDATE matches SET status = $1 WHERE id = $2 AND status = $3`
	res, err := db.ExecContext(ctx, query, string(newStatus), matchId, string(expectedStatus))
	if err != nil {
		return false, ClassifyDBError(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *repository) ClaimMatchForRun(ctx context.Context, matchId string, expectedStatus model.MatchStatus, runId string) (bool, error) {
	db, err := r.getDb()
	if err != nil {
		return false, err
	}
	query := `UPDATE matches SET status = $1, run_id = $2 WHERE id = $3 AND status = $4`
	res, err := db.ExecContext(ctx, query, string(model.MatchStatusQueued), runId, matchId, string(expectedStatus))
	if err != nil {
		return false, ClassifyDBError(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *repository) UpdateMatch(ctx context.Context, match *model.Match) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE matches SET status = $1, replay_id = $2, finished_at = $3, active = $4, committed_run_id = $5 WHERE id = $6`

	_, err = db.ExecContext(ctx, query, match.Status, match.ReplayId, match.FinishedAt, match.Active, match.CommittedRunId, match.Id)
	if err != nil {
		return ClassifyDBError(err)
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
