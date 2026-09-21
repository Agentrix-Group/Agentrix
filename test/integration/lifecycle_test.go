package integration

import (
	"compress/gzip"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/replay"
	"github.com/stretchr/testify/require"
)

// The canonical chain end to end: User -> Agent -> Submission -> ContestEntry
// -> Match -> MatchSlot -> MatchRun -> Result/Replay -> Ranking.
func TestLifecycle_CompetitiveMatchEndToEnd(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	org := comp.organizer

	// Scheduling requires an idempotency key.
	res := org.do("POST", "/api/v1/matches/"+comp.matchID+"/runs", nil)
	require.Equal(t, http.StatusBadRequest, res.Status)
	require.Equal(t, "idempotency_key_required", res.ErrorCode(t))

	ticket := org.expect(http.StatusAccepted, "POST", "/api/v1/matches/"+comp.matchID+"/runs", nil, "Idempotency-Key", "run-key-0001")
	require.NotEmpty(t, ticket["run_id"])
	require.NotEmpty(t, ticket["job_id"])
	require.Equal(t, "queued", ticket["state"])
	require.EqualValues(t, 1, ticket["attempt"])

	// Same key -> same ticket, nothing new created.
	again := org.do("POST", "/api/v1/matches/"+comp.matchID+"/runs", nil, "Idempotency-Key", "run-key-0001")
	require.Equal(t, http.StatusAccepted, again.Status)
	require.Equal(t, "true", again.Header.Get("Idempotency-Replayed"))
	require.Equal(t, ticket["run_id"], again.JSON(t)["run_id"])
	// Different key while queued -> conflict that names the real run.
	conflict := org.do("POST", "/api/v1/matches/"+comp.matchID+"/runs", nil, "Idempotency-Key", "run-key-0002")
	require.Equal(t, http.StatusConflict, conflict.Status)
	require.Equal(t, "run_in_progress", conflict.ErrorCode(t))
	require.Equal(t, ticket["run_id"], conflict.JSON(t)["error"].(map[string]any)["details"].(map[string]any)["run_id"])
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_runs WHERE match_id = $1`, comp.matchID))
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_jobs WHERE match_id = $1`, comp.matchID))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { w.Run(ctx); close(done) }()
	defer func() { cancel(); <-done }()

	match := org.waitMatch(comp.matchID, 60*time.Second, "finished", "failed")
	require.Equal(t, "finished", match["state"], "runs: %v", match["runs"])
	require.Equal(t, ticket["run_id"], match["committed_run_id"])
	results := match["results"].([]any)
	require.Len(t, results, 2)
	runs := match["runs"].([]any)
	require.Len(t, runs, 1)
	require.Equal(t, "committed", runs[0].(map[string]any)["state"])

	// Results are bound to the committed run and to each slot exactly once.
	require.Equal(t, 2, e.count(`SELECT COUNT(*) FROM results WHERE match_run_id = $1`, ticket["run_id"]))
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_jobs WHERE match_id = $1 AND state = 'completed'`, comp.matchID))

	// Replay: published, digest verified, served as gzip NDJSON.
	deadline := time.Now().Add(20 * time.Second)
	for match["replay"] == nil || match["replay"].(map[string]any)["available"] != true {
		require.True(t, time.Now().Before(deadline), "replay not published: %v", match["replay"])
		time.Sleep(100 * time.Millisecond)
		match = org.expect(http.StatusOK, "GET", "/api/v1/matches/"+comp.matchID, nil)
	}
	replayID := match["replay"].(map[string]any)["id"].(string)
	anonymous := e.client()
	meta := anonymous.expect(http.StatusOK, "GET", "/api/v1/replays/"+replayID, nil)
	require.Equal(t, "published", meta["publication_state"])
	stream := anonymous.do("GET", "/api/v1/replays/"+replayID+"/stream", nil)
	require.Equal(t, http.StatusOK, stream.Status)
	require.Equal(t, meta["sha256"], stream.Header.Get("X-Replay-SHA256"))
	// net/http negotiated gzip and decoded the body transparently; the
	// stored artifact itself is checked against the digest below.
	doc, err := replay.Decode(bytesReader(stream.Body))
	require.NoError(t, err)
	require.Equal(t, comp.matchID, doc.Metadata.MatchID)
	require.Equal(t, ticket["run_id"], doc.Metadata.RunID)
	require.Equal(t, 60, doc.Metadata.TickRate.Numerator)

	// Ranking: projection of the committed run only.
	var rankings map[string]any
	deadline = time.Now().Add(20 * time.Second)
	for {
		rankings = anonymous.expect(http.StatusOK, "GET", "/api/v1/contests/"+comp.contestID+"/rankings", nil)
		if rankings["stale"] == false && rankings["applied_runs_count"] == float64(1) {
			break
		}
		require.True(t, time.Now().Before(deadline), "rankings not refreshed: %v", rankings)
		time.Sleep(100 * time.Millisecond)
	}
	rows := rankings["rankings"].([]any)
	require.Len(t, rows, 2)
	for _, row := range rows {
		r := row.(map[string]any)
		require.EqualValues(t, 1, r["matches_played"])
		require.NotEmpty(t, r["agent_name"])
		require.NotEmpty(t, r["username"])
	}

	// A finished match cannot be scheduled again.
	res = org.do("POST", "/api/v1/matches/"+comp.matchID+"/runs", nil, "Idempotency-Key", "run-key-0003")
	require.Equal(t, http.StatusConflict, res.Status)

	stored, _, err := e.artifacts.Digest("replays/published/" + ticket["run_id"].(string) + ".ndjson.gz")
	require.NoError(t, err)
	require.Equal(t, meta["sha256"], stored)
	gz, err := e.artifacts.Open("replays/published/" + ticket["run_id"].(string) + ".ndjson.gz")
	require.NoError(t, err)
	defer gz.Close()
	zr, err := gzip.NewReader(gz)
	require.NoError(t, err)
	_, err = replay.Decode(zr)
	require.NoError(t, err)

	orphans, err := e.svc.Orphans(context.Background())
	require.NoError(t, err)
	require.Zero(t, orphans.ActiveJobsWithoutLiveRun+orphans.RunningMatchesWithoutJob+orphans.QueuedMatchesWithoutJob+orphans.CommittedWithoutResults)
}
