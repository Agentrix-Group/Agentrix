package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/model"
)

type Repository interface {
	// Contests & Categories
	ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	ListContests(ctx context.Context) ([]model.Contest, error)
	GetContest(ctx context.Context, id string) (*model.Contest, error)
	CreateContest(ctx context.Context, contest *model.Contest) error
	UpdateContest(ctx context.Context, contest *model.Contest) error
	ActivateContest(ctx context.Context, id string, isActive bool) error
	ListCategories(ctx context.Context) ([]model.Category, error)
	GetCategory(ctx context.Context, id string) (*model.Category, error)
	CreateCategory(ctx context.Context, category *model.Category) error
	UpdateCategory(ctx context.Context, category *model.Category) error
	ActivateCategory(ctx context.Context, id string, isActive bool) error

	// Participants & Auth
	ListParticipants(ctx context.Context) ([]model.Participant, error)
	GetParticipant(ctx context.Context, id string) (*model.Participant, error)
	GetParticipantByUsername(ctx context.Context, username string) (*model.Participant, error)
	GetParticipantByEmail(ctx context.Context, email string) (*model.Participant, error)
	CreateParticipant(ctx context.Context, participant *model.Participant) error
	UpdateParticipant(ctx context.Context, participant *model.Participant) error
	ActivateParticipant(ctx context.Context, id string, isActive bool) error
	HasPermission(ctx context.Context, participantId string, permission string) (bool, error)

	// Games
	ListGames(ctx context.Context) ([]model.Game, error)
	GetGame(ctx context.Context, id string) (*model.Game, error)
	CreateGame(ctx context.Context, game *model.Game) error
	UpdateGame(ctx context.Context, game *model.Game) error
	ActivateGame(ctx context.Context, id string, isActive bool) error

	// Agents
	ListAgents(ctx context.Context) ([]model.Agent, error)
	ListAgentsByParticipant(ctx context.Context, participantId string) ([]model.Agent, error)
	GetAgent(ctx context.Context, id string) (*model.Agent, error)
	CreateAgent(ctx context.Context, agent *model.Agent) error
	UpdateAgent(ctx context.Context, agent *model.Agent) error
	ActivateAgent(ctx context.Context, id string, isActive bool) error

	// Submissions
	ListSubmissions(ctx context.Context) ([]model.Submission, error)
	ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error)
	GetSubmission(ctx context.Context, id string) (*model.Submission, error)
	CreateSubmission(ctx context.Context, submission *model.Submission) error
	UpdateSubmission(ctx context.Context, submission *model.Submission) error
	ActivateSubmission(ctx context.Context, id string, isActive bool) error

	// Matches
	ListMatches(ctx context.Context) ([]model.Match, error)
	ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error)
	GetMatch(ctx context.Context, id string) (*model.Match, error)
	CreateMatch(ctx context.Context, match *model.Match) error
	UpdateMatch(ctx context.Context, match *model.Match) error
	ActivateMatch(ctx context.Context, id string, isActive bool) error

	// Results
	ListResults(ctx context.Context) ([]model.Result, error)
	ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error)
	GetResult(ctx context.Context, id string) (*model.Result, error)
	CreateResult(ctx context.Context, result *model.Result) error

	// Rankings
	ListRankings(ctx context.Context) ([]model.Ranking, error)
	ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error)
	GetRanking(ctx context.Context, id string) (*model.Ranking, error)
	GetRankingByContestAndAgent(ctx context.Context, contestId, agentId string) (*model.Ranking, error)
	UpsertRanking(ctx context.Context, ranking *model.Ranking) error
}

// ParticipantReader defines read-only operations for participants (ATD-015).
type ParticipantReader interface {
	ListParticipants(ctx context.Context) ([]model.Participant, error)
	GetParticipant(ctx context.Context, id string) (*model.Participant, error)
	GetParticipantByUsername(ctx context.Context, username string) (*model.Participant, error)
	GetParticipantByEmail(ctx context.Context, email string) (*model.Participant, error)
	HasPermission(ctx context.Context, participantId string, permission string) (bool, error)
}

// ParticipantWriter defines write-only operations for participants (ATD-015).
type ParticipantWriter interface {
	CreateParticipant(ctx context.Context, participant *model.Participant) error
	UpdateParticipant(ctx context.Context, participant *model.Participant) error
	ActivateParticipant(ctx context.Context, id string, isActive bool) error
}

// ParticipantRepository combines participant read and write operations.
type ParticipantRepository interface {
	ParticipantReader
	ParticipantWriter
}

type repository struct {
	conn *connection.Connection
}

func NewRepository(conn *connection.Connection) Repository {
	return &repository{conn: conn}
}

func (r *repository) getDb() (*sql.DB, error) {
	if r == nil || r.conn == nil || r.conn.Db == nil {
		return nil, errors.New("database not connected")
	}
	return r.conn.Db, nil
}

func (r *repository) ListParticipants(ctx context.Context) ([]model.Participant, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, username, email, role_id, active, created_at FROM participants WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []model.Participant
	for rows.Next() {
		var p model.Participant
		if err := rows.Scan(&p.Id, &p.Username, &p.Email, &p.RoleId, &p.Active, &p.CreatedAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, nil
}

func (r *repository) GetParticipant(ctx context.Context, id string) (*model.Participant, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, username, email, password, role_id, active, created_at FROM participants WHERE id = $1 AND active = TRUE`

	var p model.Participant
	err = db.QueryRowContext(ctx, query, id).Scan(
		&p.Id, &p.Username, &p.Email, &p.Password, &p.RoleId, &p.Active, &p.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &p, nil
}

func (r *repository) GetParticipantByUsername(ctx context.Context, username string) (*model.Participant, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, username, email, password, role_id, active, created_at FROM participants WHERE username = $1 AND active = TRUE`

	var p model.Participant
	err = db.QueryRowContext(ctx, query, username).Scan(
		&p.Id, &p.Username, &p.Email, &p.Password, &p.RoleId, &p.Active, &p.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &p, nil
}

func (r *repository) GetParticipantByEmail(ctx context.Context, email string) (*model.Participant, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, username, email, password, role_id, active, created_at FROM participants WHERE email = $1 AND active = TRUE`

	var p model.Participant
	err = db.QueryRowContext(ctx, query, email).Scan(
		&p.Id, &p.Username, &p.Email, &p.Password, &p.RoleId, &p.Active, &p.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &p, nil
}

func (r *repository) CreateParticipant(ctx context.Context, participant *model.Participant) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `INSERT INTO participants (id, username, email, password, role_id, active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.ExecContext(ctx, query,
		participant.Id, participant.Username, participant.Email, participant.Password,
		participant.RoleId, participant.Active, participant.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) UpdateParticipant(ctx context.Context, participant *model.Participant) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE participants SET email = $1, role_id = $2, active = $3 WHERE id = $4`

	_, err = db.ExecContext(ctx, query, participant.Email, participant.RoleId, participant.Active, participant.Id)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) ActivateParticipant(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE participants SET active = $1 WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) HasPermission(ctx context.Context, participantId string, permission string) (bool, error) {
	db, err := r.getDb()
	if err != nil {
		return false, err
	}

	query := `
		SELECT COUNT(*)
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN participants part ON part.role_id = rp.role_id
		WHERE part.id = $1 AND p.id = $2 AND p.active = TRUE AND rp.active = TRUE AND part.active = TRUE`

	var count int
	err = db.QueryRowContext(ctx, query, participantId, permission).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
