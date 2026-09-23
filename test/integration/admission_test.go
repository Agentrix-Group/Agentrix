package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

// ADR-0014 (N4): the admission queue lives in submissions.status. Concurrent
// workers never claim the same submission, a claim abandoned by a dead
// worker is reclaimed after staleAfter, and only `validating` rows finish.
func TestIntegration_Admission_ClaimQueue(t *testing.T) {
	r := require.New(t)
	dbName := fmt.Sprintf("agentrix_admission_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()
	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	for i := 0; i < 3; i++ {
		_, err := conn.Db.Exec(`
			INSERT INTO submissions (id, agent_id, version, code_path, language, status, active, created_at)
			VALUES ($1, 'agent-alice-01', $2, 'unused.py', 'python', 'validating', TRUE, NOW() + make_interval(secs => $3))`,
			fmt.Sprintf("sub-validating-%d", i), 100+i, float64(i))
		r.NoError(err)
	}

	repo, ok := repository.NewRepository(conn).(repository.SubmissionAdmissionRepository)
	r.True(ok)
	ctx := context.Background()

	// Three concurrent claims get three different submissions; a fourth
	// finds nothing because all claims are fresh.
	var mu sync.Mutex
	claimed := map[string]bool{}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub, err := repo.ClaimNextAdmission(ctx, time.Minute)
			r.NoError(err)
			r.NotNil(sub)
			mu.Lock()
			claimed[sub.Id] = true
			mu.Unlock()
		}()
	}
	wg.Wait()
	r.Len(claimed, 3)
	sub, err := repo.ClaimNextAdmission(ctx, time.Minute)
	r.NoError(err)
	r.Nil(sub, "fresh claims must not be handed out twice")

	// A claim older than staleAfter is reclaimed.
	_, err = conn.Db.Exec(`UPDATE submissions SET admission_claimed_at = NOW() - INTERVAL '3 minutes' WHERE id = 'sub-validating-1'`)
	r.NoError(err)
	sub, err = repo.ClaimNextAdmission(ctx, 2*time.Minute)
	r.NoError(err)
	r.NotNil(sub)
	r.Equal("sub-validating-1", sub.Id)

	// Finishing sets status and reason once; a late second finish is ignored.
	r.NoError(repo.FinishAdmission(ctx, "sub-validating-1", "rejected", "the bot process stopped: boom"))
	r.NoError(repo.FinishAdmission(ctx, "sub-validating-1", "ready", ""))
	var status, detail string
	r.NoError(conn.Db.QueryRow(`SELECT status, COALESCE(error_detail, '') FROM submissions WHERE id = 'sub-validating-1'`).Scan(&status, &detail))
	r.Equal("rejected", status)
	r.Equal("the bot process stopped: boom", detail)
}

// A submission that has not passed admission cannot play.
func TestIntegration_Admission_RunMatchRequiresReadySubmissions(t *testing.T) {
	r := require.New(t)
	dbName := fmt.Sprintf("agentrix_admission_run_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()
	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)
	t.Setenv("AGENTRIX_ENGINE_DIGEST", fmt.Sprintf("%064d", 0))

	_, _, svc, queue := setupPostgresOrchestrationServer(t, conn, "worker-admission-01")
	defer queue.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, status := range []string{"validating", "rejected"} {
		createResp, err := svc.CreateMatch(ctx, &model.Match{
			ContestId: "contest-orch-01",
			GameId:    "starfighter",
			Seed:      7,
		}, []string{"sub-alice-01", "sub-bob-01"})
		r.NoError(err)
		_, err = conn.Db.Exec(`UPDATE submissions SET status = $1 WHERE id = 'sub-bob-01'`, status)
		r.NoError(err)

		_, err = svc.RunMatch(ctx, createResp.MatchId, "")
		r.ErrorIs(err, service.ErrInvalidSubmissions, status)
		r.Contains(err.Error(), "is not ready")
	}
}
