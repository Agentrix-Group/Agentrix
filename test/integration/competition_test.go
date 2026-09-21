package integration

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestCompetition_ContestStateMachine(t *testing.T) {
	e := newEnv(t, envOptions{})
	org := e.user("org", model.RoleOrganizer)
	player := e.user("plr", model.RolePlayer)
	c := org.expect(http.StatusCreated, "POST", "/api/v1/contests", map[string]any{"game_id": "starfighter", "name": "SM"})
	id := c["id"].(string)
	require.Equal(t, "draft", c["state"])
	require.ElementsMatch(t, []any{"registration_open", "cancelled"}, c["allowed_transitions"])

	// Drafts are invisible to non-managers.
	require.Equal(t, http.StatusNotFound, player.do("GET", "/api/v1/contests/"+id, nil).Status)
	require.Len(t, player.expect(http.StatusOK, "GET", "/api/v1/contests", nil)["items"], 0)

	// Invalid transitions are rejected with 409; players cannot drive states.
	res := org.do("POST", "/api/v1/contests/"+id+"/transitions", map[string]string{"state": "running"})
	require.Equal(t, http.StatusConflict, res.Status)
	require.Equal(t, "invalid_transition", res.ErrorCode(t))
	require.Equal(t, http.StatusForbidden, player.do("POST", "/api/v1/contests/"+id+"/transitions", map[string]string{"state": "registration_open"}).Status)
	// Legacy aliases do not exist.
	res = org.do("POST", "/api/v1/contests/"+id+"/transitions", map[string]string{"state": "open"})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status)

	// Scoring policy is validated with a closed schema and frozen after draft.
	res = org.do("PATCH", "/api/v1/contests/"+id, map[string]any{"scoring_policy": map[string]any{"win_points": 1, "draw_points": 2,
		"loss_points": 0, "disqualification_penalty": 0, "tiebreakers": []string{}}})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status)
	res = org.do("PATCH", "/api/v1/contests/"+id, map[string]any{"scoring_policy": map[string]any{"win_points": 3, "draw_points": 1,
		"loss_points": 0, "disqualification_penalty": 0, "tiebreakers": []string{"coin_flip"}}})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status)
	res = org.do("PATCH", "/api/v1/contests/"+id, map[string]any{"starts_at": "2030-01-02T00:00:00Z", "ends_at": "2030-01-01T00:00:00Z"})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status)

	for _, to := range []string{"registration_open", "registration_closed"} {
		org.expect(http.StatusOK, "POST", "/api/v1/contests/"+id+"/transitions", map[string]string{"state": to})
	}
	res = org.do("PATCH", "/api/v1/contests/"+id, map[string]any{"scoring_policy": model.DefaultScoringPolicy()})
	require.Equal(t, http.StatusConflict, res.Status)
	// Running requires at least the minimum roster.
	res = org.do("POST", "/api/v1/contests/"+id+"/transitions", map[string]string{"state": "running"})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status)
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+id+"/transitions", map[string]string{"state": "cancelled", "reason": "empty"})
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+id+"/transitions", map[string]string{"state": "archived"})
	require.Equal(t, http.StatusConflict, org.do("POST", "/api/v1/contests/"+id+"/transitions", map[string]string{"state": "draft"}).Status)
}

func TestCompetition_EnrollmentLocksExactSubmission(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	org := e.user("org", model.RoleOrganizer)
	alice := e.user("alice", model.RolePlayer)
	bob := e.user("bob", model.RolePlayer)
	c := org.expect(http.StatusCreated, "POST", "/api/v1/contests", map[string]any{"game_id": "starfighter", "name": "E"})
	cid := c["id"].(string)
	agentID, sub1 := alice.agentWithReadySubmission(w, "A", "bot_hunter.py")
	sub2 := alice.expect(http.StatusAccepted, "POST", "/api/v1/agents/"+agentID+"/submissions",
		bundleUpload(t, "A2", referenceBot(t, "bot_evasive.py")))["id"].(string)

	// Registration not open yet.
	res := alice.do("POST", "/api/v1/contests/"+cid+"/entries", map[string]string{"agent_id": agentID, "submission_id": sub1})
	require.Equal(t, http.StatusNotFound, res.Status, "draft contest is not visible to players")
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+cid+"/transitions", map[string]string{"state": "registration_open"})

	// submission_id is mandatory; validating submissions cannot be locked.
	res = alice.do("POST", "/api/v1/contests/"+cid+"/entries", map[string]string{"agent_id": agentID})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status)
	res = alice.do("POST", "/api/v1/contests/"+cid+"/entries", map[string]string{"agent_id": agentID, "submission_id": sub2})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status)
	require.Equal(t, "submission_not_ready", res.ErrorCode(t))
	// Bob cannot enroll Alice's agent.
	res = bob.do("POST", "/api/v1/contests/"+cid+"/entries", map[string]string{"agent_id": agentID, "submission_id": sub1})
	require.Equal(t, http.StatusNotFound, res.Status)

	entry := alice.expect(http.StatusCreated, "POST", "/api/v1/contests/"+cid+"/entries", map[string]string{"agent_id": agentID, "submission_id": sub1})
	require.Equal(t, sub1, entry["submission_id"])
	require.EqualValues(t, 1, entry["submission_version"])
	require.Equal(t, http.StatusConflict, alice.do("POST", "/api/v1/contests/"+cid+"/entries",
		map[string]string{"agent_id": agentID, "submission_id": sub1}).Status)
	// No ranking row is created by enrolling.
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM rankings`))

	// Re-lock to v2 once it is admitted, while registration is open.
	admitAll(t, e.svc, w)
	entryID := entry["id"].(string)
	relocked := alice.expect(http.StatusOK, "PUT", "/api/v1/contests/"+cid+"/entries/"+entryID+"/submission", map[string]string{"submission_id": sub2})
	require.EqualValues(t, 2, relocked["submission_version"])

	// After closing registration the roster is frozen.
	_, bobSub := bob.agentWithReadySubmission(w, "B", "bot_random.py")
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+cid+"/transitions", map[string]string{"state": "registration_closed"})
	res = alice.do("PUT", "/api/v1/contests/"+cid+"/entries/"+entryID+"/submission", map[string]string{"submission_id": sub1})
	require.Equal(t, http.StatusConflict, res.Status)
	bobAgents := bob.expect(http.StatusOK, "GET", "/api/v1/agents", nil)["items"].([]any)
	res = bob.do("POST", "/api/v1/contests/"+cid+"/entries", map[string]string{"agent_id": bobAgents[0].(map[string]any)["id"].(string), "submission_id": bobSub})
	require.Equal(t, http.StatusConflict, res.Status)

	// Withdraw requires a reason and is audited; disqualify needs entries:manage:any.
	require.Equal(t, http.StatusUnprocessableEntity, alice.do("POST", "/api/v1/contests/"+cid+"/entries/"+entryID+"/withdraw", map[string]string{"reason": ""}).Status)
	require.Equal(t, http.StatusForbidden, alice.do("POST", "/api/v1/contests/"+cid+"/entries/"+entryID+"/disqualify", map[string]string{"reason": "x"}).Status)
	dq := org.expect(http.StatusOK, "POST", "/api/v1/contests/"+cid+"/entries/"+entryID+"/disqualify", map[string]string{"reason": "cheating"})
	require.Equal(t, "disqualified", dq["status"])
	require.Equal(t, "cheating", dq["status_reason"])
	audit, err := e.store.ListAudit(context.Background(), "contest_entry", entryID)
	require.NoError(t, err)
	var actions []string
	for _, a := range audit {
		actions = append(actions, a.Action)
	}
	require.Contains(t, actions, "entry.enrolled")
	require.Contains(t, actions, "entry.resubmitted")
	require.Contains(t, actions, "entry.disqualified")
	require.Equal(t, http.StatusConflict, alice.do("POST", "/api/v1/contests/"+cid+"/entries/"+entryID+"/withdraw", map[string]string{"reason": "late"}).Status)
}

func TestCompetition_MatchCreationValidatesRoster(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	org := comp.organizer
	cid := comp.contestID

	cases := []struct {
		name string
		body map[string]any
		code int
	}{
		{"duplicate competitor", map[string]any{"contest_id": cid, "mode": "competitive", "entry_ids": []string{comp.entries[0], comp.entries[0]}}, 422},
		{"too few players", map[string]any{"contest_id": cid, "mode": "competitive", "entry_ids": []string{comp.entries[0]}}, 422},
		{"too many players", map[string]any{"contest_id": cid, "mode": "competitive", "entry_ids": []string{comp.entries[0], comp.entries[1], comp.entries[0]}}, 422},
		{"unknown entry", map[string]any{"contest_id": cid, "mode": "competitive", "entry_ids": []string{comp.entries[0], newUUID()}}, 404},
		{"unknown contest", map[string]any{"contest_id": newUUID(), "mode": "competitive", "entry_ids": comp.entries}, 404},
		{"competitive without contest", map[string]any{"mode": "competitive", "entry_ids": comp.entries}, 422},
		{"competitive with raw submissions", map[string]any{"contest_id": cid, "mode": "competitive", "submission_ids": comp.subs}, 422},
		{"legacy mode", map[string]any{"contest_id": cid, "mode": "pending", "entry_ids": comp.entries}, 422},
	}
	for _, tc := range cases {
		res := org.do("POST", "/api/v1/matches", tc.body)
		require.Equal(t, tc.code, res.Status, "%s: %s", tc.name, res.Body)
	}
	// Players cannot create matches.
	require.Equal(t, http.StatusForbidden, comp.players[0].do("POST", "/api/v1/matches",
		map[string]any{"contest_id": cid, "mode": "competitive", "entry_ids": comp.entries}).Status)

	// Exhibition with the same submission twice (mirror) is allowed and never ranked.
	ex := org.expect(http.StatusCreated, "POST", "/api/v1/matches", map[string]any{"mode": "exhibition", "submission_ids": []string{comp.subs[0], comp.subs[0]}})
	require.Equal(t, "exhibition", ex["mode"])
	require.Equal(t, 2, len(ex["slots"].([]any)))
	for _, s := range ex["slots"].([]any) {
		require.Nil(t, s.(map[string]any)["contest_entry_id"])
	}
}

func TestCompetition_AdmissionRejectsBrokenBots(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	p := e.user("author", model.RolePlayer)
	agent := p.expect(http.StatusCreated, "POST", "/api/v1/agents", map[string]string{"game_id": "starfighter", "name": "Broken"})
	id := agent["id"].(string)

	// Invalid bundles never create a submission.
	for _, bad := range [][]byte{[]byte("not a zip"), nil} {
		res := p.do("POST", "/api/v1/agents/"+id+"/submissions", &multipartBody{data: bad, contentType: "application/zip"})
		require.Contains(t, []int{400, 422}, res.Status)
	}
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM submissions`))

	cases := map[string]string{
		"syntax":  "def broken(:\n",
		"silent":  "import sys\nfor line in sys.stdin: pass\n",
		"garbage": "import sys\nsys.stdin.readline()\nsys.stdin.readline()\nprint('hello')\n",
		"crash":   "raise SystemExit(3)\n",
	}
	ids := map[string]string{}
	for name, code := range cases {
		sub := p.expect(http.StatusAccepted, "POST", "/api/v1/agents/"+id+"/submissions", bundleUpload(t, name, []byte(code)))
		ids[name] = sub["id"].(string)
	}
	for i := 0; i < 3; i++ {
		admitAll(t, e.svc, w)
	}
	for name, subID := range ids {
		sub := p.expect(http.StatusOK, "GET", "/api/v1/submissions/"+subID, nil)
		require.Equal(t, "rejected", sub["status"], name)
		require.NotEmpty(t, sub["admission_error"], name)
		require.NotContains(t, sub["admission_error"], e.artifacts.Root(), "admission errors must not leak host paths")
	}
	// A rejected submission cannot be enrolled or used in matches.
	versions := p.expect(http.StatusOK, "GET", "/api/v1/agents/"+id+"/submissions", nil)["items"].([]any)
	require.Len(t, versions, len(cases))
}

// Concurrent enrollment of the same agent creates exactly one entry.
func TestCompetition_ConcurrentEnrollment(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	org := e.user("org", model.RoleOrganizer)
	p := e.user("racer", model.RolePlayer)
	c := org.expect(http.StatusCreated, "POST", "/api/v1/contests", map[string]any{"game_id": "starfighter", "name": "Race"})
	cid := c["id"].(string)
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+cid+"/transitions", map[string]string{"state": "registration_open"})
	agentID, subID := p.agentWithReadySubmission(w, "R", "bot_random.py")
	var wg sync.WaitGroup
	statuses := make([]int, 6)
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses[i] = p.do("POST", "/api/v1/contests/"+cid+"/entries", map[string]string{"agent_id": agentID, "submission_id": subID}).Status
		}(i)
	}
	wg.Wait()
	created := 0
	for _, s := range statuses {
		if s == http.StatusCreated {
			created++
		} else {
			require.Equal(t, http.StatusConflict, s)
		}
	}
	require.Equal(t, 1, created)
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM contest_entries WHERE contest_id = $1`, cid))
}
