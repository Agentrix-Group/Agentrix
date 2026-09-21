package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

var (
	errContestNotFound = model.NotFound("contest_not_found", "contest not found")
	errEntryNotFound   = model.NotFound("entry_not_found", "contest entry not found")
)

const contestColumns = `id, game_id, name, description, state, starts_at, ends_at, scoring_policy, created_by, created_at, updated_at`

func scanContest(row interface{ Scan(...any) error }) (*model.Contest, error) {
	var c model.Contest
	var starts, ends sql.NullTime
	var policy []byte
	if err := row.Scan(&c.ID, &c.GameID, &c.Name, &c.Description, &c.State, &starts, &ends, &policy,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	c.StartsAt, c.EndsAt = timePtr(starts), timePtr(ends)
	parsed, err := model.ParseScoringPolicy(policy)
	if err != nil {
		return nil, err
	}
	c.ScoringPolicy = parsed
	return &c, nil
}

func (q *Queries) CreateContest(ctx context.Context, c *model.Contest) error {
	policy, err := json.Marshal(c.ScoringPolicy)
	if err != nil {
		return err
	}
	_, err = q.db.ExecContext(ctx, `
		INSERT INTO contests (id, game_id, name, description, state, starts_at, ends_at, scoring_policy, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)`,
		c.ID, c.GameID, c.Name, c.Description, c.State, c.StartsAt, c.EndsAt, policy, c.CreatedBy, c.CreatedAt)
	return classify(err)
}

func (q *Queries) getContest(ctx context.Context, id, suffix string) (*model.Contest, error) {
	if !validUUID(id) {
		return nil, errContestNotFound
	}
	c, err := scanContest(q.db.QueryRowContext(ctx, `SELECT `+contestColumns+` FROM contests WHERE id = $1`+suffix, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errContestNotFound
	}
	return c, classify(err)
}

func (q *Queries) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	return q.getContest(ctx, id, "")
}

// LockContest takes an exclusive row lock; used by state transitions.
func (q *Queries) LockContest(ctx context.Context, id string) (*model.Contest, error) {
	return q.getContest(ctx, id, " FOR UPDATE")
}

// ShareLockContest blocks concurrent transitions while a roster change runs.
func (q *Queries) ShareLockContest(ctx context.Context, id string) (*model.Contest, error) {
	return q.getContest(ctx, id, " FOR SHARE")
}

func (q *Queries) ListContests(ctx context.Context, includeDrafts bool) ([]model.Contest, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+contestColumns+` FROM contests
		WHERE $1 OR state <> 'draft' ORDER BY created_at DESC, id`, includeDrafts)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Contest
	for rows.Next() {
		c, err := scanContest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (q *Queries) UpdateContestDetails(ctx context.Context, c *model.Contest, now time.Time) error {
	policy, err := json.Marshal(c.ScoringPolicy)
	if err != nil {
		return err
	}
	res, err := q.db.ExecContext(ctx, `UPDATE contests SET name = $2, description = $3, starts_at = $4, ends_at = $5,
		scoring_policy = $6, updated_at = $7 WHERE id = $1`,
		c.ID, c.Name, c.Description, c.StartsAt, c.EndsAt, policy, now)
	return expectOne(res, err, errContestNotFound)
}

// TransitionContest applies a compare-and-set state change.
func (q *Queries) TransitionContest(ctx context.Context, id string, from, to model.ContestState, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE contests SET state = $3, updated_at = $4 WHERE id = $1 AND state = $2`,
		id, from, to, now)
	return expectOne(res, err, model.Conflict("contest_state_changed", "contest state changed concurrently"))
}

// ---------------------------------------------------------------------------
// Entries
// ---------------------------------------------------------------------------

const entryColumns = `e.id, e.contest_id, e.game_id, e.agent_id, a.name, e.user_id, u.username, e.submission_id, s.version,
	e.status, COALESCE(e.status_reason, ''), COALESCE(e.status_changed_by::text, ''), e.status_changed_at, e.created_at, e.updated_at`

const entryJoins = ` FROM contest_entries e
	JOIN agents a ON a.id = e.agent_id
	JOIN users u ON u.id = e.user_id
	JOIN submissions s ON s.id = e.submission_id`

func scanEntry(row interface{ Scan(...any) error }) (*model.ContestEntry, error) {
	var e model.ContestEntry
	var changed sql.NullTime
	err := row.Scan(&e.ID, &e.ContestID, &e.GameID, &e.AgentID, &e.AgentName, &e.UserID, &e.Username, &e.SubmissionID,
		&e.SubmissionVer, &e.Status, &e.StatusReason, &e.StatusChangedBy, &changed, &e.CreatedAt, &e.UpdatedAt)
	e.StatusChangedAt = timePtr(changed)
	return &e, err
}

func (q *Queries) CreateEntry(ctx context.Context, e *model.ContestEntry) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO contest_entries (id, contest_id, game_id, agent_id, user_id, submission_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'enrolled', $7, $7)`,
		e.ID, e.ContestID, e.GameID, e.AgentID, e.UserID, e.SubmissionID, e.CreatedAt)
	return classify(err)
}

func (q *Queries) getEntry(ctx context.Context, contestID, entryID, suffix string) (*model.ContestEntry, error) {
	if !validUUID(entryID) || !validUUID(contestID) {
		return nil, errEntryNotFound
	}
	e, err := scanEntry(q.db.QueryRowContext(ctx, `SELECT `+entryColumns+entryJoins+
		` WHERE e.id = $1 AND e.contest_id = $2`+suffix, entryID, contestID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errEntryNotFound
	}
	return e, classify(err)
}

func (q *Queries) GetEntry(ctx context.Context, contestID, entryID string) (*model.ContestEntry, error) {
	return q.getEntry(ctx, contestID, entryID, "")
}

func (q *Queries) LockEntry(ctx context.Context, contestID, entryID string) (*model.ContestEntry, error) {
	return q.getEntry(ctx, contestID, entryID, " FOR UPDATE OF e")
}

func (q *Queries) ListEntries(ctx context.Context, contestID string) ([]model.ContestEntry, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+entryColumns+entryJoins+
		` WHERE e.contest_id = $1 ORDER BY e.created_at, e.id`, contestID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.ContestEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// ListEntriesByIDs returns entries of a contest in the requested order and
// locks them so their submission cannot change while a match is created.
func (q *Queries) LockEntriesByIDs(ctx context.Context, contestID string, ids []string) ([]model.ContestEntry, error) {
	out := make([]model.ContestEntry, 0, len(ids))
	for _, id := range ids {
		e, err := q.LockEntry(ctx, contestID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, nil
}

func (q *Queries) UpdateEntrySubmission(ctx context.Context, entryID, submissionID string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE contest_entries SET submission_id = $2, updated_at = $3
		WHERE id = $1 AND status = 'enrolled'`, entryID, submissionID, now)
	return expectOne(res, err, model.InvalidTransition("contest entry", "not enrolled", "resubmitted"))
}

func (q *Queries) UpdateEntryStatus(ctx context.Context, entryID string, to model.EntryStatus, reason, actorID string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE contest_entries SET status = $2, status_reason = $3, status_changed_by = $4,
		status_changed_at = $5, updated_at = $5 WHERE id = $1 AND status = 'enrolled'`, entryID, to, reason, actorID, now)
	return expectOne(res, err, model.InvalidTransition("contest entry", "not enrolled", to))
}
