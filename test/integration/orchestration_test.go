package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/server"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

func seedOrchestrationBaseData(t *testing.T, conn *connection.Connection) {
	_, err := conn.Db.Exec(`
		INSERT INTO roles (id, description, active) VALUES
		('admin', 'Platform Administrator', TRUE),
		('pilot', 'Agent Pilot', TRUE),
		('spectator', 'Spectator', TRUE)
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO permissions (id, description, active) VALUES
		('read', 'Read system data and resources', TRUE),
		('write', 'Modify and update resources', TRUE),
		('admin', 'System administration capabilities', TRUE),
		('execute-match', 'Run and schedule matches', TRUE),
		('submit-agent', 'Upload and manage bot submissions', TRUE)
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO role_permissions (role_id, permission_id, active) VALUES
		('admin', 'read', TRUE),
		('admin', 'write', TRUE),
		('admin', 'admin', TRUE),
		('admin', 'execute-match', TRUE),
		('admin', 'submit-agent', TRUE),
		('pilot', 'read', TRUE),
		('pilot', 'execute-match', TRUE),
		('pilot', 'submit-agent', TRUE)
		ON CONFLICT (role_id, permission_id) DO NOTHING;

		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-admin-01', 'admin', 'admin@example.com', '$argon2id$v=19$m=65536,t=3,p=2$mock$mock', 'admin', TRUE),
		('u-pilot-01', 'alice', 'alice@example.com', '$argon2id$v=19$m=65536,t=3,p=2$mock$mock', 'pilot', TRUE),
		('u-pilot-02', 'bob', 'bob@example.com', '$argon2id$v=19$m=65536,t=3,p=2$mock$mock', 'pilot', TRUE)
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-admin-01', 'admin'),
		('u-pilot-01', 'pilot'),
		('u-pilot-02', 'pilot')
		ON CONFLICT (user_id, role_id) DO NOTHING;

		INSERT INTO games (id, name, description, active) VALUES
		('starfighter', 'Starfighter 1v1', 'Official dogfight simulation', TRUE)
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO agents (id, name, game_id, owner_user_id, active) VALUES
		('agent-alice-01', 'AliceFighter', 'starfighter', 'u-pilot-01', TRUE),
		('agent-bob-01', 'BobFighter', 'starfighter', 'u-pilot-02', TRUE)
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO submissions (id, agent_id, version, code_path, language, status, active) VALUES
		('sub-alice-01', 'agent-alice-01', 1, 'games/starfighter/examples/bot_random.py', 'python', 'ready', TRUE),
		('sub-bob-01', 'agent-bob-01', 1, 'games/starfighter/examples/bot_evasive.py', 'python', 'ready', TRUE)
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO contests (id, name, description, game_id, state, active, starts_at, ends_at) VALUES
		('contest-orch-01', 'Orchestration Tournament', '1v1 tournament', 'starfighter', 'open', TRUE, NOW(), NOW() + INTERVAL '7 days')
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status) VALUES
		('ce-alice-01', 'contest-orch-01', 'agent-alice-01', 'u-pilot-01', 'sub-alice-01', 'active'),
		('ce-bob-01', 'contest-orch-01', 'agent-bob-01', 'u-pilot-02', 'sub-bob-01', 'active')
		ON CONFLICT (id) DO NOTHING;
	`)
	require.NoError(t, err)
}

func setupPostgresOrchestrationServer(t *testing.T, conn *connection.Connection, workerID string) (*server.Server, *auth.Auth, service.Service, connection.JobQueue) {
	r := require.New(t)
	repo := repository.NewRepository(conn)
	queue, err := connection.NewPostgresJobQueue(conn.Db, workerID)
	r.NoError(err)

	sandbox := executor.NewSandbox(0)
	tempDir, err := os.MkdirTemp("", "agentrix-orch-artifacts-*")
	r.NoError(err)
	cfg := &config.Config{
		Artifacts: config.Artifacts{Dir: tempDir},
	}
	artifacts, err := connection.NewArtifactStore(context.Background(), cfg)
	r.NoError(err)

	svc := service.NewService(repo, artifacts, queue, sandbox)
	srv := server.NewServer(svc)
	authModule := auth.NewAuth("test-access-secret", "test-refresh-secret")
	srv.Auth = authModule
	return srv, authModule, svc, queue
}

// 1. Mandatory Test: Two concurrent calls to POST /matches/{id}/run produce exactly one run and one job
func TestIntegration_Orchestration_ConcurrentRunMatch(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_orch_concur_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	srv, authMod, svc, _ := setupPostgresOrchestrationServer(t, conn, "worker-pool-main")
	tokens, err := authMod.GenerateAuthToken("u-admin-01", "admin")
	r.NoError(err)

	// Schedule match with 2 slots
	match := &model.Match{
		ContestId: "contest-orch-01",
		GameId:    "starfighter",
		Seed:      98765,
	}
	createResp, err := svc.CreateMatch(ctx, match, []string{"sub-alice-01", "sub-bob-01"})
	r.NoError(err)
	r.NotEmpty(createResp.MatchId)
	matchId := createResp.MatchId

	// Verify match is pending with 0 runs and 0 jobs
	var runCount, jobCount int
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM match_runs WHERE match_id = $1`, matchId).Scan(&runCount)
	r.NoError(err)
	r.Equal(0, runCount)
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM match_jobs WHERE match_id = $1`, matchId).Scan(&jobCount)
	r.NoError(err)
	r.Equal(0, jobCount)

	// Spawn 10 concurrent HTTP requests to POST /api/v1/matches/{id}/run
	const numConcurrent = 10
	var wg sync.WaitGroup
	type callResult struct {
		statusCode int
		body       model.RunMatchResponse
		err        error
	}
	results := make([]callResult, numConcurrent)

	// Barrier to start all goroutines simultaneously
	startBarrier := make(chan struct{})

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startBarrier

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/run", matchId), nil)
			req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
			rec := httptest.NewRecorder()
			srv.Handler.ServeHTTP(rec, req)

			var runResp model.RunMatchResponse
			unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &runResp)
			results[idx] = callResult{
				statusCode: rec.Code,
				body:       runResp,
				err:        unmarshalErr,
			}
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	// Assert every concurrent request succeeded with 200 or 202
	var expectedRunId, expectedJobId string
	for i, res := range results {
		r.NoError(res.err, "request %d failed to unmarshal JSON: %s", i)
		r.True(res.statusCode == http.StatusAccepted || res.statusCode == http.StatusOK,
			"request %d returned status code %d", i, res.statusCode)
		r.Equal(matchId, res.body.MatchId)
		r.Equal("queued", res.body.Status)
		r.NotEmpty(res.body.RunId)
		r.NotEmpty(res.body.JobId)

		if expectedRunId == "" {
			expectedRunId = res.body.RunId
			expectedJobId = res.body.JobId
		} else {
			// All concurrent requests must return the exact same run_id and job_id
			r.Equal(expectedRunId, res.body.RunId, "request %d received different run_id", i)
			r.Equal(expectedJobId, res.body.JobId, "request %d received different job_id", i)
		}
	}

	// Verify database persistence: EXACTLY 1 run and EXACTLY 1 job created
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM match_runs WHERE match_id = $1`, matchId).Scan(&runCount)
	r.NoError(err)
	r.Equal(1, runCount, "Database must have exactly 1 match_run")

	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM match_jobs WHERE match_id = $1`, matchId).Scan(&jobCount)
	r.NoError(err)
	r.Equal(1, jobCount, "Database must have exactly 1 match_job")

	// Verify match status in PostgreSQL
	var matchStatus string
	err = conn.Db.QueryRow(`SELECT status FROM matches WHERE id = $1`, matchId).Scan(&matchStatus)
	r.NoError(err)
	r.Equal("queued", matchStatus)
}

// 2. Mandatory Test: Rerun of failed match produces second run without destroying the first
func TestIntegration_Orchestration_RerunFailedMatch(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_orch_rerun_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	srv, authMod, svc, queue := setupPostgresOrchestrationServer(t, conn, "worker-1")
	tokens, err := authMod.GenerateAuthToken("u-admin-01", "admin")
	r.NoError(err)

	// Schedule match
	match := &model.Match{
		ContestId: "contest-orch-01",
		GameId:    "starfighter",
		Seed:      54321,
	}
	createResp, err := svc.CreateMatch(ctx, match, []string{"sub-alice-01", "sub-bob-01"})
	r.NoError(err)
	matchId := createResp.MatchId

	// First execution trigger
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/run", matchId), nil)
	req1.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec1 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec1, req1)
	r.Equal(http.StatusAccepted, rec1.Code)

	var resp1 model.RunMatchResponse
	r.NoError(json.Unmarshal(rec1.Body.Bytes(), &resp1))
	r.Equal(fmt.Sprintf("%s-run-1", matchId), resp1.RunId)

	// Worker dequeues the job
	job, err := queue.Dequeue(ctx)
	r.NoError(err)
	r.Equal(matchId, job.MatchId)
	r.Equal(resp1.RunId, job.RunId)

	// Simulate execution failure
	r.NoError(svc.FailMatchRun(ctx, resp1.RunId, "agent crashed: segmentation fault"))
	_, err = conn.Db.Exec(`UPDATE matches SET status = 'failed' WHERE id = $1`, matchId)
	r.NoError(err)
	_, err = conn.Db.Exec(`UPDATE match_jobs SET status = 'failed', last_error = 'agent crashed' WHERE match_id = $1`, matchId)
	r.NoError(err)

	// Second execution trigger (RERUN)
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/run", matchId), nil)
	req2.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec2, req2)
	r.Equal(http.StatusAccepted, rec2.Code)

	var resp2 model.RunMatchResponse
	r.NoError(json.Unmarshal(rec2.Body.Bytes(), &resp2))
	r.Equal(matchId, resp2.MatchId)
	r.Equal(fmt.Sprintf("%s-run-2", matchId), resp2.RunId)
	r.Equal("queued", resp2.Status)

	// Verify complete audit history in match_runs: Both Run 1 and Run 2 exist!
	rows, err := conn.Db.Query(`
		SELECT id, attempt, status, COALESCE(last_error, '')
		FROM match_runs
		WHERE match_id = $1
		ORDER BY attempt ASC
	`, matchId)
	r.NoError(err)
	defer rows.Close()

	type runRecord struct {
		id        string
		attempt   int
		status    string
		lastError string
	}
	var auditRuns []runRecord
	for rows.Next() {
		var rec runRecord
		r.NoError(rows.Scan(&rec.id, &rec.attempt, &rec.status, &rec.lastError))
		auditRuns = append(auditRuns, rec)
	}

	r.Len(auditRuns, 2, "Both runs must be preserved in match_runs table")

	// Run 1 check
	r.Equal(resp1.RunId, auditRuns[0].id)
	r.Equal(1, auditRuns[0].attempt)
	r.Equal("failed", auditRuns[0].status)
	r.Equal("agent crashed: segmentation fault", auditRuns[0].lastError)

	// Run 2 check
	r.Equal(resp2.RunId, auditRuns[1].id)
	r.Equal(2, auditRuns[1].attempt)
	r.Equal("created", auditRuns[1].status)
}

// 3. Mandatory Test: Simulated worker death releases lease and second worker completes match; zombie worker is rejected
func TestIntegration_Orchestration_WorkerLeaseRecoveryAndZombieRejection(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_orch_lease_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	// Set up two workers connected to PostgreSQL
	worker1Queue, err := connection.NewPostgresJobQueue(conn.Db, "worker-node-alpha")
	r.NoError(err)
	worker2Queue, err := connection.NewPostgresJobQueue(conn.Db, "worker-node-beta")
	r.NoError(err)

	repo := repository.NewRepository(conn)
	sandbox := executor.NewSandbox(0)
	tempDir, err := os.MkdirTemp("", "agentrix-orch-lease-*")
	r.NoError(err)
	cfg := &config.Config{
		Artifacts: config.Artifacts{Dir: tempDir},
	}
	artifacts, err := connection.NewArtifactStore(ctx, cfg)
	r.NoError(err)
	svc := service.NewService(repo, artifacts, worker1Queue, sandbox)

	// Create and schedule match
	match := &model.Match{
		ContestId: "contest-orch-01",
		GameId:    "starfighter",
		Seed:      777,
	}
	createResp, err := svc.CreateMatch(ctx, match, []string{"sub-alice-01", "sub-bob-01"})
	r.NoError(err)
	matchId := createResp.MatchId

	runResp, err := svc.RunMatch(ctx, matchId)
	r.NoError(err)
	r.Equal("queued", runResp.Status)

	// Step 1: Worker 1 reserves the job
	job1, err := worker1Queue.Dequeue(ctx)
	r.NoError(err)
	r.Equal(matchId, job1.MatchId)
	r.Equal(int64(1), job1.FencingToken, "First reservation must receive fencing token 1")

	// Step 2: While Worker 1 has active lease, Worker 2 cannot dequeue it
	_, err = worker2Queue.Dequeue(ctx)
	r.ErrorIs(err, connection.ErrQueueEmpty, "Worker 2 must not be able to steal an active lease")

	// Step 3: Simulate Worker 1 crashing / expiring its lease
	_, err = conn.Db.Exec(`
		UPDATE match_jobs
		SET lease_until = NOW() - INTERVAL '5 seconds'
		WHERE match_id = $1
	`, matchId)
	r.NoError(err)

	// Step 4: Worker 2 reclaims the job via lease recovery
	job2, err := worker2Queue.Dequeue(ctx)
	r.NoError(err)
	r.Equal(matchId, job2.MatchId)
	r.Equal(int64(2), job2.FencingToken, "Second worker must receive monotonically incremented fencing token 2")

	// Step 5: Worker 2 executes and commits the match result with fencing token 2
	now := time.Now().UTC()
	commitWorker2 := model.MatchResultCommit{
		MatchID:           matchId,
		RunID:             job2.RunId,
		WorkerID:          "worker-node-beta",
		FencingToken:      2,
		Status:            common.MatchStatusFinished,
		TerminationReason: "all_ticks_completed",
		FinalTick:         100,
		FinalStateHash:    "hash-state-verified",
		ReplayID:          "rep-orch-99",
		ReplaySHA256:      "sha256-verified-commit",
		FinishedAt:        now,
		Results: []model.Result{
			{
				Id:           "res-alice-01",
				MatchId:      matchId,
				SubmissionId: "sub-alice-01",
				Score:        10,
				Rank:         1,
				Status:       "finished",
			},
			{
				Id:           "res-bob-01",
				MatchId:      matchId,
				SubmissionId: "sub-bob-01",
				Score:        5,
				Rank:         2,
				Status:       "finished",
			},
		},
	}
	r.NoError(svc.CommitMatchResult(ctx, commitWorker2), "Worker 2 commit with valid token 2 must succeed")

	// Verify match and job status in DB
	var dbMatchStatus, committedRunId, dbJobStatus string
	err = conn.Db.QueryRow(`SELECT status, committed_run_id FROM matches WHERE id = $1`, matchId).
		Scan(&dbMatchStatus, &committedRunId)
	r.NoError(err)
	r.Equal(common.MatchStatusFinished, dbMatchStatus)
	r.Equal(job2.RunId, committedRunId)

	err = conn.Db.QueryRow(`SELECT status FROM match_jobs WHERE match_id = $1`, matchId).Scan(&dbJobStatus)
	r.NoError(err)
	r.Equal("completed", dbJobStatus)

	// Step 6: Zombie Worker 1 wakes up and attempts to commit with stale fencing token 1
	commitZombieWorker1 := model.MatchResultCommit{
		MatchID:           matchId,
		RunID:             job1.RunId,
		WorkerID:          "worker-node-alpha",
		FencingToken:      1, // STALE TOKEN!
		Status:            common.MatchStatusFinished,
		TerminationReason: "zombie_late_result",
		FinalTick:         80,
		FinishedAt:        now,
		Results: []model.Result{
			{
				Id:           "res-alice-zombie",
				MatchId:      matchId,
				SubmissionId: "sub-alice-01",
				Score:        0,
				Rank:         2,
				Status:       "eliminated",
			},
		},
	}

	zombieErr := svc.CommitMatchResult(ctx, commitZombieWorker1)
	r.Error(zombieErr, "Zombie worker commit with stale fencing token 1 must be rejected")
	r.ErrorIs(zombieErr, repository.ErrMatchAlreadyCommitted)

	// Verify results in database were NOT corrupted by the zombie worker
	var countResults int
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM results WHERE match_id = $1`, matchId).Scan(&countResults)
	r.NoError(err)
	r.Equal(2, countResults, "Only Worker 2 results must exist")

	var scoreAlice int
	err = conn.Db.QueryRow(`SELECT score FROM results WHERE match_id = $1 AND submission_id = 'sub-alice-01'`, matchId).Scan(&scoreAlice)
	r.NoError(err)
	r.Equal(10, scoreAlice, "Alice score must remain 10 from Worker 2, not overwritten by zombie")
}

// 4. Test: Idempotency-Key header deduplication and persistence
func TestIntegration_Orchestration_IdempotencyKeyHeader(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_orch_idem_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	srv, authMod, svc, _ := setupPostgresOrchestrationServer(t, conn, "worker-idem")
	tokens, err := authMod.GenerateAuthToken("u-admin-01", "admin")
	r.NoError(err)

	match := &model.Match{
		ContestId: "contest-orch-01",
		GameId:    "starfighter",
		Seed:      112233,
	}
	createResp, err := svc.CreateMatch(ctx, match, []string{"sub-alice-01", "sub-bob-01"})
	r.NoError(err)
	matchId := createResp.MatchId

	idempotencyKey := fmt.Sprintf("idem-key-%d", time.Now().UnixNano())

	// Request 1 with Idempotency-Key
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/run", matchId), nil)
	req1.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	rec1 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec1, req1)
	r.Equal(http.StatusAccepted, rec1.Code)

	var resp1 model.RunMatchResponse
	r.NoError(json.Unmarshal(rec1.Body.Bytes(), &resp1))
	r.Equal(matchId, resp1.MatchId)
	r.Equal("queued", resp1.Status)

	// Request 2 with same Idempotency-Key
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/run", matchId), nil)
	req2.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	rec2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec2, req2)
	r.Equal(http.StatusAccepted, rec2.Code)

	var resp2 model.RunMatchResponse
	r.NoError(json.Unmarshal(rec2.Body.Bytes(), &resp2))

	// Must return exact same response
	r.Equal(resp1.RunId, resp2.RunId)
	r.Equal(resp1.JobId, resp2.JobId)
	r.Equal(resp1.Status, resp2.Status)

	// Verify idempotency record in database
	var countKey int
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM idempotency_keys WHERE key = $1`, idempotencyKey).Scan(&countKey)
	r.NoError(err)
	r.Equal(1, countKey)
}
