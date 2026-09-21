package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Matches_TransactionalCreationAndSlots(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_match_tx_%d", time.Now().UnixNano())
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// Run canonical schema migrations via Goose
	err := database.Migrate(conn.Db)
	r.NoError(err)

	srv, _ := setupTestServer(conn)
	svc := srv.Service

	// Seed roles, permissions, users, agents, and submissions
	_, err = conn.Db.Exec(`
		INSERT INTO roles (id, description, active) VALUES
		('admin', 'Platform Administrator', TRUE),
		('pilot', 'Agent Pilot', TRUE),
		('spectator', 'Spectator', TRUE)
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-pilot-01', 'alice', 'alice@example.com', '$argon2id$v=19$m=65536,t=3,p=2$mock$mock', 'pilot', TRUE),
		('u-pilot-02', 'bob', 'bob@example.com', '$argon2id$v=19$m=65536,t=3,p=2$mock$mock', 'pilot', TRUE);

		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-pilot-01', 'pilot'),
		('u-pilot-02', 'pilot');

		INSERT INTO games (id, name, description, active) VALUES
		('starfighter', 'Starfighter 1v1', 'Official dogfight simulation', TRUE);

		INSERT INTO agents (id, name, game_id, owner_user_id, active) VALUES
		('agent-alice-01', 'AliceFighter', 'starfighter', 'u-pilot-01', TRUE),
		('agent-bob-01', 'BobFighter', 'starfighter', 'u-pilot-02', TRUE);

		INSERT INTO submissions (id, agent_id, version, code_path, language, status, active) VALUES
		('sub-alice-01', 'agent-alice-01', 1, 'games/starfighter/examples/bot_random.py', 'python', 'ready', TRUE),
		('sub-bob-01', 'agent-bob-01', 1, 'games/starfighter/examples/bot_evasive.py', 'python', 'ready', TRUE);

		INSERT INTO contests (id, name, description, game_id, state, active, starts_at, ends_at) VALUES
		('contest-open-01', 'Open Tournament', '1v1 tournament', 'starfighter', 'open', TRUE, NOW(), NOW() + INTERVAL '7 days'),
		('contest-draft-01', 'Draft Tournament', 'Draft tournament', 'starfighter', 'draft', TRUE, NOW(), NOW() + INTERVAL '7 days');

		INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status) VALUES
		('ce-alice-01', 'contest-open-01', 'agent-alice-01', 'u-pilot-01', 'sub-alice-01', 'enrolled'),
		('ce-bob-01', 'contest-open-01', 'agent-bob-01', 'u-pilot-02', 'sub-bob-01', 'active');
	`)
	r.NoError(err)

	// Test 1: Reject match creation for draft contest
	{
		draftMatch := &model.Match{
			ContestId: "contest-draft-01",
			GameId:    "starfighter",
		}
		_, err := svc.CreateMatch(ctx, draftMatch, []string{"sub-alice-01", "sub-bob-01"})
		r.Error(err, "Match creation should be rejected for draft contest")
	}

	// Test 2: Successful transactional match creation for open contest
	var createdMatchId string
	{
		match := &model.Match{
			ContestId: "contest-open-01",
			GameId:    "starfighter",
			Seed:      123456,
		}
		resp, err := svc.CreateMatch(ctx, match, []string{"sub-alice-01", "sub-bob-01"})
		r.NoError(err)
		r.NotNil(resp)
		r.NotEmpty(resp.MatchId)
		r.Equal(match.Id, resp.MatchId)
		r.Len(resp.Slots, 2)
		r.Equal(fmt.Sprintf("Match %s scheduled successfully", match.Id), resp.Message)
		createdMatchId = match.Id

		// Verify Slot 0
		r.Equal(0, resp.Slots[0].SlotIndex)
		r.Equal("sub-alice-01", resp.Slots[0].SubmissionId)
		r.NotNil(resp.Slots[0].ContestEntryId)
		r.Equal("ce-alice-01", *resp.Slots[0].ContestEntryId)
		r.Equal("AliceFighter", resp.Slots[0].AgentName)
		r.Equal("alice", resp.Slots[0].Username)

		// Verify Slot 1
		r.Equal(1, resp.Slots[1].SlotIndex)
		r.Equal("sub-bob-01", resp.Slots[1].SubmissionId)
		r.NotNil(resp.Slots[1].ContestEntryId)
		r.Equal("ce-bob-01", *resp.Slots[1].ContestEntryId)
		r.Equal("BobFighter", resp.Slots[1].AgentName)
		r.Equal("bob", resp.Slots[1].Username)
	}

	// Test 3: Verify PostgreSQL persistence of match and match_slots
	{
		var dbContestId, dbGameId, dbStatus string
		var dbSeed int64
		err = conn.Db.QueryRow(`SELECT contest_id, game_id, status, seed FROM matches WHERE id = $1`, createdMatchId).
			Scan(&dbContestId, &dbGameId, &dbStatus, &dbSeed)
		r.NoError(err)
		r.Equal("contest-open-01", dbContestId)
		r.Equal("starfighter", dbGameId)
		r.Equal("pending", dbStatus)
		r.Equal(int64(123456), dbSeed)

		rows, err := conn.Db.Query(`SELECT slot_index, contest_entry_id, submission_id, agent_name, username FROM match_slots WHERE match_id = $1 ORDER BY slot_index ASC`, createdMatchId)
		r.NoError(err)
		defer rows.Close()

		var slotCount int
		for rows.Next() {
			var slotIndex int
			var entryId, subId, agentName, username string
			r.NoError(rows.Scan(&slotIndex, &entryId, &subId, &agentName, &username))
			if slotIndex == 0 {
				r.Equal("ce-alice-01", entryId)
				r.Equal("sub-alice-01", subId)
				r.Equal("AliceFighter", agentName)
				r.Equal("alice", username)
			} else if slotIndex == 1 {
				r.Equal("ce-bob-01", entryId)
				r.Equal("sub-bob-01", subId)
				r.Equal("BobFighter", agentName)
				r.Equal("bob", username)
			}
			slotCount++
		}
		r.Equal(2, slotCount)
	}
}

func TestIntegration_StateMachines_ContestAndMatchCAS(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_cas_%d", time.Now().UnixNano())
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	err := database.Migrate(conn.Db)
	r.NoError(err)

	srv, _ := setupTestServer(conn)
	svc := srv.Service

	// Seed basic game and contest
	_, err = conn.Db.Exec(`
		INSERT INTO games (id, name, description, active) VALUES
		('starfighter', 'Starfighter 1v1', 'Official dogfight simulation', TRUE);

		INSERT INTO contests (id, name, description, game_id, state, active) VALUES
		('contest-cas-01', 'CAS Contest', 'Tournament for CAS testing', 'starfighter', 'open', TRUE);

		INSERT INTO matches (id, contest_id, game_id, status, seed, active) VALUES
		('match-cas-01', 'contest-cas-01', 'starfighter', 'pending', 42, TRUE);
	`)
	r.NoError(err)

	// Test 1: Contest CAS transition open -> live
	{
		// Invalid transition open -> draft is rejected
		_, err := svc.UpdateContestStateCAS(ctx, "contest-cas-01", model.ContestStateOpen, model.ContestStateDraft)
		r.ErrorIs(err, model.ErrInvalidStateTransition)

		// Valid transition open -> live succeeds
		ok, err := svc.UpdateContestStateCAS(ctx, "contest-cas-01", model.ContestStateOpen, model.ContestStateLive)
		r.NoError(err)
		r.True(ok)

		// Concurrent/stale CAS expecting open now fails (returns false, nil)
		ok, err = svc.UpdateContestStateCAS(ctx, "contest-cas-01", model.ContestStateOpen, model.ContestStateLive)
		r.NoError(err)
		r.False(ok, "Stale CAS should return false without error")

		// Verify state in database is 'live'
		var currentState string
		err = conn.Db.QueryRow(`SELECT state FROM contests WHERE id = $1`, "contest-cas-01").Scan(&currentState)
		r.NoError(err)
		r.Equal("live", currentState)
	}

	// Test 2: MatchRun CAS transitions
	{
		run := &model.MatchRun{
			Id:           "run-cas-01",
			MatchId:      "match-cas-01",
			WorkerId:     "worker-test-01",
			FencingToken: 1,
			Status:       model.MatchRunStatusCreated,
			StartedAt:    time.Now().UTC(),
		}
		r.NoError(svc.CreateMatchRun(ctx, run))

		// Invalid transition created -> completed rejected
		_, err = svc.UpdateMatchRunStatusCAS(ctx, "run-cas-01", model.MatchRunStatusCreated, model.MatchRunStatusCompleted)
		r.ErrorIs(err, model.ErrInvalidStateTransition)

		// Valid transition created -> dispatching
		ok, err := svc.UpdateMatchRunStatusCAS(ctx, "run-cas-01", model.MatchRunStatusCreated, model.MatchRunStatusDispatching)
		r.NoError(err)
		r.True(ok)

		// Stale CAS expecting created returns false
		ok, err = svc.UpdateMatchRunStatusCAS(ctx, "run-cas-01", model.MatchRunStatusCreated, model.MatchRunStatusDispatching)
		r.NoError(err)
		r.False(ok)

		// Valid transition dispatching -> running
		ok, err = svc.UpdateMatchRunStatusCAS(ctx, "run-cas-01", model.MatchRunStatusDispatching, model.MatchRunStatusRunning)
		r.NoError(err)
		r.True(ok)

		// Stale CAS expecting dispatching returns false
		ok, err = svc.UpdateMatchRunStatusCAS(ctx, "run-cas-01", model.MatchRunStatusDispatching, model.MatchRunStatusRunning)
		r.NoError(err)
		r.False(ok)

		// Verify state in database is 'running'
		var currentRunStatus string
		err = conn.Db.QueryRow(`SELECT status FROM match_runs WHERE id = $1`, "run-cas-01").Scan(&currentRunStatus)
		r.NoError(err)
		r.Equal("running", currentRunStatus)
	}
}
