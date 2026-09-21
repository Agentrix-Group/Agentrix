package integration

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

// playMatch creates a competitive match with the given entries (in slot
// order), runs it through reservation and commits the given ranks/scores.
func (e *env) playMatch(comp *competition, entries []string, ranks []int, scores []int, disqualified map[int]bool) string {
	e.t.Helper()
	m := comp.organizer.expect(http.StatusCreated, "POST", "/api/v1/matches", map[string]any{"contest_id": comp.contestID,
		"mode": "competitive", "entry_ids": entries, "seed": 7})
	schedule(e.t, comp.organizer, m["id"].(string))
	res := e.reserve("w1")
	c := e.completionFor(res)
	for i := range c.Slots {
		c.Slots[i].Rank, c.Slots[i].Score, c.Slots[i].Disqualified = ranks[i], scores[i], disqualified[i]
	}
	require.NoError(e.t, e.svc.CommitRun(context.Background(), c))
	return m["id"].(string)
}

func rankingByAgent(t *testing.T, view map[string]any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, r := range view["rankings"].([]any) {
		row := r.(map[string]any)
		out[row["entry_id"].(string)] = row
	}
	return out
}

func TestRankings_ProjectionOfCommittedRuns(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w, "bot_hunter.py", "bot_evasive.py", "bot_random.py")
	a, b, c := comp.entries[0], comp.entries[1], comp.entries[2]
	ctx := context.Background()
	// The competition helper scheduled a first match between a and b: run
	// it, fail it (retryable) and commit the retry with the opposite result.
	schedule(t, comp.organizer, comp.matchID)
	first := e.reserve("w1")
	firstCompletion := e.completionFor(first) // a would win this attempt
	_ = firstCompletion
	require.NoError(t, e.svc.FailRun(ctx, *first, model.ErrorInfrastructure, "worker crash"))
	e.clock.Advance(time.Minute)
	retry := e.reserve("w1")
	rc := e.completionFor(retry)
	rc.Slots[0].Rank, rc.Slots[1].Rank = 2, 1 // b wins the committed run
	rc.Slots[0].Score, rc.Slots[1].Score = 1, 5
	require.NoError(t, e.svc.CommitRun(ctx, rc))

	e.playMatch(comp, []string{a, c}, []int{1, 2}, []int{4, 0}, nil)
	e.playMatch(comp, []string{b, c}, []int{1, 1}, []int{2, 2}, nil) // draw
	require.NoError(t, e.svc.RecomputeRankings(ctx, comp.contestID))

	view := comp.organizer.expect(http.StatusOK, "GET", "/api/v1/contests/"+comp.contestID+"/rankings", nil)
	require.Equal(t, false, view["stale"])
	require.EqualValues(t, 3, view["applied_runs_count"])
	rows := rankingByAgent(t, view)
	// b: win (3) + draw (1) = 4; a: loss + win = 3; c: loss + draw = 1
	require.EqualValues(t, 4, rows[b]["points"])
	require.EqualValues(t, 3, rows[a]["points"])
	require.EqualValues(t, 1, rows[c]["points"])
	require.EqualValues(t, 2, rows[a]["matches_played"], "the failed attempt must not count")
	require.EqualValues(t, 1, rows[b]["rank"])
	require.EqualValues(t, 2, rows[a]["rank"])
	require.EqualValues(t, 3, rows[c]["rank"])

	// Recomputing N times yields identical rows and watermark.
	before := comp.organizer.expect(http.StatusOK, "GET", "/api/v1/contests/"+comp.contestID+"/rankings", nil)
	for i := 0; i < 3; i++ {
		comp.organizer.expect(http.StatusOK, "POST", "/api/v1/contests/"+comp.contestID+"/rankings/recalculate", nil)
	}
	after := comp.organizer.expect(http.StatusOK, "GET", "/api/v1/contests/"+comp.contestID+"/rankings", nil)
	require.Equal(t, before["rankings"], after["rankings"])
	require.Equal(t, before["applied_runs_digest"], after["applied_runs_digest"])

	// Disqualifying an entry puts it last and marks the projection stale until recomputed.
	comp.organizer.expect(http.StatusOK, "POST", "/api/v1/contests/"+comp.contestID+"/entries/"+b+"/disqualify", map[string]string{"reason": "rules"})
	require.Equal(t, true, comp.organizer.expect(http.StatusOK, "GET", "/api/v1/contests/"+comp.contestID+"/rankings", nil)["stale"])
	require.NoError(t, e.svc.RecomputeRankings(ctx, comp.contestID))
	rows = rankingByAgent(t, comp.organizer.expect(http.StatusOK, "GET", "/api/v1/contests/"+comp.contestID+"/rankings", nil))
	require.EqualValues(t, 3, rows[b]["rank"])
	require.EqualValues(t, 1, rows[a]["rank"])
}

func TestRankings_TiebreakersAndDisqualifiedResults(t *testing.T) {
	scored := model.ComputeRankings("c", []model.RankingEntry{
		{EntryID: "a", CreatedAt: time.Unix(1, 0)}, {EntryID: "b", CreatedAt: time.Unix(2, 0)}, {EntryID: "c", CreatedAt: time.Unix(3, 0)},
	}, []model.CommittedMatchResult{
		{MatchID: "m1", EntryID: "a", Score: 5, Outcome: model.OutcomeWin}, {MatchID: "m1", EntryID: "b", Score: 1, Outcome: model.OutcomeLoss},
		{MatchID: "m2", EntryID: "b", Score: 9, Outcome: model.OutcomeWin}, {MatchID: "m2", EntryID: "c", Score: 0, Outcome: model.OutcomeLoss},
		{MatchID: "m3", EntryID: "c", Score: 3, Outcome: model.OutcomeWin}, {MatchID: "m3", EntryID: "a", Score: 2, Outcome: model.OutcomeDisqualified},
	}, model.ScoringPolicy{WinPoints: 3, DrawPoints: 1, DisqualificationPenalty: 2, Tiebreakers: []model.Tiebreaker{model.TiebreakScoreDiff}})
	byID := map[string]model.Ranking{}
	for _, r := range scored {
		byID[r.EntryID] = r
	}
	// a: 3 - 2 = 1 point; b: 3; c: 3. b and c tie on points; score diff b=+5, c=-6.
	require.Equal(t, 1, byID["a"].Points)
	require.Equal(t, 1, byID["b"].Rank)
	require.Equal(t, 2, byID["c"].Rank)
	require.Equal(t, 3, byID["a"].Rank)
	require.Equal(t, 1, byID["a"].Disqualifications)

	// Full tie keeps a shared rank; output order is deterministic.
	tied := model.ComputeRankings("c", []model.RankingEntry{{EntryID: "x", CreatedAt: time.Unix(2, 0)}, {EntryID: "y", CreatedAt: time.Unix(1, 0)}},
		[]model.CommittedMatchResult{{MatchID: "m", EntryID: "x", Score: 1, Outcome: model.OutcomeDraw}, {MatchID: "m", EntryID: "y", Score: 1, Outcome: model.OutcomeDraw}},
		model.DefaultScoringPolicy())
	require.Equal(t, 1, tied[0].Rank)
	require.Equal(t, 1, tied[1].Rank)
	require.Equal(t, "y", tied[0].EntryID, "enrollment order breaks display ties")
}

func TestRankings_SnapshotsAreVersionedImmutableAndConcurrent(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	schedule(t, comp.organizer, comp.matchID)
	require.NoError(t, e.svc.CommitRun(context.Background(), e.completionFor(e.reserve("w1"))))

	// Players cannot publish.
	require.Equal(t, http.StatusForbidden, comp.players[0].do("POST", "/api/v1/contests/"+comp.contestID+"/rankings/snapshots", nil).Status)

	var wg sync.WaitGroup
	statuses := make([]int, 5)
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses[i] = comp.organizer.do("POST", "/api/v1/contests/"+comp.contestID+"/rankings/snapshots", nil).Status
		}(i)
	}
	wg.Wait()
	for _, s := range statuses {
		require.Equal(t, http.StatusCreated, s)
	}
	snaps := comp.organizer.expect(http.StatusOK, "GET", "/api/v1/contests/"+comp.contestID+"/rankings/snapshots", nil)["items"].([]any)
	require.Len(t, snaps, 5)
	for i, s := range snaps {
		snap := s.(map[string]any)
		require.EqualValues(t, i+1, snap["version"])
		require.Equal(t, snaps[0].(map[string]any)["rankings_sha256"], snap["rankings_sha256"], "same standings, same digest")
		require.EqualValues(t, 1, snap["applied_runs_count"])
	}
	v1 := e.client().expect(http.StatusOK, "GET", "/api/v1/contests/"+comp.contestID+"/rankings/snapshots/1", nil)
	require.Len(t, v1["rankings"], 2)
	_, err := e.db.Exec(`UPDATE ranking_snapshots SET version = 99 WHERE contest_id = $1`, comp.contestID)
	require.Error(t, err)

	// Snapshots cannot be published once the contest is archived... or before it runs.
	org := comp.organizer
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+comp.contestID+"/transitions", map[string]string{"state": "finished"})
	org.expect(http.StatusCreated, "POST", "/api/v1/contests/"+comp.contestID+"/rankings/snapshots", nil)
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+comp.contestID+"/transitions", map[string]string{"state": "archived"})
	require.Equal(t, http.StatusConflict, org.do("POST", "/api/v1/contests/"+comp.contestID+"/rankings/snapshots", nil).Status)
}
