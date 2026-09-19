package executor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/config"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/game"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

type memoryAuditRepo struct {
	matches    map[string]*model.Match
	matchRuns  map[string]*model.MatchRun
	results    map[string][]model.Result
	tokenByJob map[string]int64
	jobStatus  map[string]string
}

func newMemoryAuditRepo() *memoryAuditRepo {
	return &memoryAuditRepo{
		matches:    make(map[string]*model.Match),
		matchRuns:  make(map[string]*model.MatchRun),
		results:    make(map[string][]model.Result),
		tokenByJob: make(map[string]int64),
		jobStatus:  make(map[string]string),
	}
}

func (m *memoryAuditRepo) CommitMatchResult(ctx context.Context, commit model.MatchResultCommit) error {
	match := m.matches[commit.MatchID]
	if match == nil {
		return fmt.Errorf("match not found: %s", commit.MatchID)
	}
	match.Status = commit.Status
	match.ReplayId = commit.ReplayID
	match.FinishedAt = &commit.FinishedAt

	m.matchRuns[commit.RunID] = &model.MatchRun{
		Id:           commit.RunID,
		MatchId:      commit.MatchID,
		WorkerId:     commit.WorkerID,
		FencingToken: commit.FencingToken,
		Status:       model.MatchRunStatusCommitted,
		StartedAt:    commit.FinishedAt.Add(-100 * time.Millisecond),
		FinishedAt:   &commit.FinishedAt,
	}

	m.results[commit.MatchID] = commit.Results
	m.jobStatus[commit.MatchID] = "completed"
	return nil
}

func (m *memoryAuditRepo) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	match := m.matches[id]
	if match == nil {
		return nil, fmt.Errorf("match not found: %s", id)
	}
	return match, nil
}

func (m *memoryAuditRepo) UpdateMatch(ctx context.Context, match *model.Match) error {
	m.matches[match.Id] = match
	return nil
}

func (m *memoryAuditRepo) ListMatches(ctx context.Context) ([]model.Match, error) { return nil, nil }
func (m *memoryAuditRepo) ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error) {
	return nil, nil
}
func (m *memoryAuditRepo) CreateMatch(ctx context.Context, match *model.Match) error {
	m.matches[match.Id] = match
	return nil
}
func (m *memoryAuditRepo) ActivateMatch(ctx context.Context, id string, isActive bool) error {
	return nil
}
func (m *memoryAuditRepo) ListResults(ctx context.Context) ([]model.Result, error) { return nil, nil }
func (m *memoryAuditRepo) ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error) {
	return m.results[matchId], nil
}
func (m *memoryAuditRepo) GetResult(ctx context.Context, id string) (*model.Result, error) {
	return nil, nil
}
func (m *memoryAuditRepo) CreateResult(ctx context.Context, result *model.Result) error {
	m.results[result.MatchId] = append(m.results[result.MatchId], *result)
	return nil
}

func (m *memoryAuditRepo) CreateMatchRun(ctx context.Context, run *model.MatchRun) error {
	m.matchRuns[run.Id] = run
	return nil
}
func (m *memoryAuditRepo) GetMatchRun(ctx context.Context, id string) (*model.MatchRun, error) {
	return m.matchRuns[id], nil
}
func (m *memoryAuditRepo) UpdateMatchRunHeartbeat(ctx context.Context, runID string, t time.Time) error {
	if r := m.matchRuns[runID]; r != nil {
		r.HeartbeatAt = &t
	}
	return nil
}

// TestMVPCanonicalCertification executes the complete MVP end-to-end pipeline:
// Real Starfighter engine + Real Python bots + Bubblewrap/Process sandbox +
// Exact 60Hz timestep + Bilateral protocol state machine + Fenced atomic commit +
// Replay SHA256 sealing and atomic publishing.
func TestMVPCanonicalCertification(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	engineBin := resolveTestPath("bin/starfighter-engine")
	if _, err := os.Stat(engineBin); err != nil {
		t.Skip("Official starfighter-engine binary not compiled")
	}

	botHunter := resolveTestPath("games/starfighter/examples/bot_hunter.py")
	botEvasive := resolveTestPath("games/starfighter/examples/bot_evasive.py")
	r.FileExists(botHunter)
	r.FileExists(botEvasive)

	// Configure environment
	tempDir := t.TempDir()
	store, err := connection.NewArtifactStore(ctx, &config.Config{
		Artifacts: config.Artifacts{Dir: tempDir},
	})
	r.NoError(err)

	origManifest := game.GetRegistry().GetManifest("starfighter")
	defer func() {
		if origManifest != nil {
			game.GetRegistry().RegisterManifest(origManifest)
		}
	}()

	gamesDir := resolveTestPath("games")
	r.NoError(game.GetRegistry().LoadGamesFromDir(gamesDir))

	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	manifest.BinaryPath = engineBin
	manifest.MaxTicks = 10

	repo := newMemoryAuditRepo()
	repo.matches["m-cert-1"] = &model.Match{
		Id:     "m-cert-1",
		GameId: "starfighter",
		Status: common.MatchStatusPending,
		Active: true,
	}

	queue := connection.NewJobQueue(10)
	defer queue.Close()

	sandbox := NewSandbox(0)
	svc := service.NewService(nil, store, queue, sandbox)

	// Adapter for mock service using audit repo
	certSvc := &certServiceAdapter{
		Service: svc,
		repo:    repo,
		store:   store,
	}

	executor := NewMatchExecutor(certSvc, sandbox)

	job := &connection.MatchJob{
		JobId:         "job-cert-1",
		RunId:         "run-cert-1",
		MatchId:       "m-cert-1",
		GameId:        "starfighter",
		SubmissionIds: []string{botHunter, botEvasive},
		Seed:          12345,
		FencingToken:  100,
	}

	// 1. Execute match
	execErr := executor.Execute(ctx, job)
	r.NoError(execErr, "execution must succeed cleanly")

	// 2. Verify match status in repository
	match, err := repo.GetMatch(ctx, "m-cert-1")
	r.NoError(err)
	r.Equal(common.MatchStatusFinished, match.Status)
	r.NotEmpty(match.ReplayId)
	r.NotNil(match.FinishedAt)

	// 3. Verify match run audit
	run, err := repo.GetMatchRun(ctx, "run-cert-1")
	r.NoError(err)
	r.NotNil(run)
	r.Equal(model.MatchRunStatusCommitted, run.Status)
	r.Equal(int64(100), run.FencingToken)

	// 4. Verify results
	results := repo.results["m-cert-1"]
	r.Len(results, 2, "must have results for both fighters")
	r.NotEmpty(results[0].Rank)
	r.NotEmpty(results[1].Rank)

	// 5. Verify Replay was published atomically and NOT left in tmp
	replayID := match.ReplayId
	r.False(store.Exists(fmt.Sprintf("replays/tmp/%s.ndjson", replayID)), "temporary replay must be moved")
	r.True(store.Exists(fmt.Sprintf("replays/%s.ndjson", replayID)), "canonical replay must exist")

	// 6. Decode and verify replay integrity
	rawReplay, err := certSvc.StreamReplay(ctx, replayID)
	r.NoError(err)
	r.NotEmpty(rawReplay)

	hasher := sha256.New()
	hasher.Write(rawReplay)
	expectedSha := hex.EncodeToString(hasher.Sum(nil))

	fetchedReplay, err := certSvc.GetReplay(ctx, replayID)
	r.NoError(err)
	r.Equal(expectedSha, fetchedReplay.Sha256, "replay SHA256 digest must match bit-for-bit")
	r.Equal(int64(len(rawReplay)), fetchedReplay.SizeBytes)
	r.GreaterOrEqual(fetchedReplay.DurationTicks, 1)

	t.Logf("MVP Certification Succeeded: match=%s run=%s replay=%s sha256=%s ticks=%d",
		match.Id, run.Id, replayID, expectedSha[:16], fetchedReplay.DurationTicks)
}

type certServiceAdapter struct {
	service.Service
	repo  *memoryAuditRepo
	store connection.ArtifactStore
}

func (c *certServiceAdapter) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	return c.repo.GetMatch(ctx, id)
}

func (c *certServiceAdapter) UpdateMatch(ctx context.Context, match *model.Match) error {
	return c.repo.UpdateMatch(ctx, match)
}

func (c *certServiceAdapter) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	return &model.Submission{
		Id:       id,
		AgentId:  id + "-agent",
		Language: "python",
		CodePath: id,
		Status:   common.SubmissionStatusReady,
		Active:   true,
	}, nil
}

func (c *certServiceAdapter) CommitMatchResult(ctx context.Context, commit model.MatchResultCommit) error {
	return c.repo.CommitMatchResult(ctx, commit)
}

func (c *certServiceAdapter) CreateResult(ctx context.Context, res *model.Result) error {
	return c.repo.CreateResult(ctx, res)
}
