package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateMachinesHaveNoAliasesAndTerminalStates(t *testing.T) {
	require.False(t, ContestState("open").Valid())
	require.False(t, ContestState("in_progress").Valid())
	require.False(t, MatchState("pending").Valid())
	require.False(t, MatchState("completed").Valid())
	require.True(t, ContestDraft.CanTransitionTo(ContestRegistrationOpen))
	require.False(t, ContestRunning.CanTransitionTo(ContestRegistrationOpen))
	require.False(t, ContestArchived.CanTransitionTo(ContestDraft))
	require.True(t, MatchFailed.CanTransitionTo(MatchQueued))
	require.False(t, MatchFinished.CanTransitionTo(MatchQueued))
	require.False(t, MatchRunning.CanTransitionTo(MatchCancelled))
	for _, s := range []RunState{RunCommitted, RunFailed, RunTimedOut, RunAborted} {
		require.True(t, s.Terminal(), s)
		for to := range RunTransitions {
			require.False(t, s.CanTransitionTo(to), "%s -> %s", s, to)
		}
	}
	// Every transition target is itself a known state.
	for from, tos := range MatchTransitions {
		for _, to := range tos {
			_, ok := MatchTransitions[to]
			require.True(t, ok, "%s -> %s", from, to)
		}
	}
}

func validSpec() ExecutionSpec {
	sha := "0000000000000000000000000000000000000000000000000000000000000001"
	return ExecutionSpec{SpecVersion: ExecutionSpecVersion, MatchID: "m", Mode: ModeCompetitive, Seed: 7,
		TickRate: TickRate{60, 1},
		Limits: ExecutionLimits{MaxTicks: 10, TurnTimeoutMs: 1, InitTimeoutMs: 1, WallTimeMs: 1, MemoryMB: 1, CPUSeconds: 1,
			MaxProcesses: 1, MaxOutputKB: 1, MaxFileSizeKB: 1, MaxStderrKB: 1, MaxLineBytes: 1},
		Game:    SpecGame{ID: "g", Version: "1.0.0", Config: map[string]any{"b": 2, "a": 0.7, "n": map[string]any{"z": 1}}},
		Engine:  SpecEngine{Version: "1", SHA256: sha, ProtocolVersion: "p/1"},
		Runtime: SpecRuntime{BotProtocolVersion: "b/1", SandboxProfile: "s/1"}, ReplayFormat: "r/2",
		Slots: []SpecSlot{
			{Index: 0, SlotID: "s0", ContestEntryID: "e0", AgentID: "a0", SubmissionID: "x0", ArtifactKey: "k0", ArtifactSHA256: sha, Runtime: "python3", Entrypoint: "bot.py"},
			{Index: 1, SlotID: "s1", ContestEntryID: "e1", AgentID: "a1", SubmissionID: "x1", ArtifactKey: "k1", ArtifactSHA256: sha, Runtime: "python3", Entrypoint: "bot.py"},
		}}
}

func TestExecutionSpecSealRoundTrip(t *testing.T) {
	spec := validSpec()
	raw, hash, err := spec.Seal()
	require.NoError(t, err)
	// Simulate the JSONB round trip: key order and number formatting change.
	var generic map[string]any
	require.NoError(t, json.Unmarshal(raw, &generic))
	reordered, err := json.Marshal(generic)
	require.NoError(t, err)
	parsed, err := ParseExecutionSpec(reordered, hash)
	require.NoError(t, err)
	again, err := parsed.Hash()
	require.NoError(t, err)
	require.Equal(t, hash, again)
	_, err = ParseExecutionSpec(reordered, "0"+hash[1:])
	require.ErrorContains(t, err, "hash mismatch")
}

func TestExecutionSpecRejectsIncompleteSpecs(t *testing.T) {
	mutations := map[string]func(*ExecutionSpec){
		"no engine digest":     func(s *ExecutionSpec) { s.Engine.SHA256 = "" },
		"no artifact digest":   func(s *ExecutionSpec) { s.Slots[1].ArtifactSHA256 = "" },
		"no entry competitive": func(s *ExecutionSpec) { s.Slots[0].ContestEntryID = "" },
		"duplicate submission": func(s *ExecutionSpec) { s.Slots[1].SubmissionID = "x0" },
		"bad tick rate":        func(s *ExecutionSpec) { s.TickRate = TickRate{0, 1} },
		"slot order":           func(s *ExecutionSpec) { s.Slots[1].Index = 5 },
		"missing limits":       func(s *ExecutionSpec) { s.Limits.MemoryMB = 0 },
		"unknown mode":         func(s *ExecutionSpec) { s.Mode = "pending" },
	}
	for name, mutate := range mutations {
		spec := validSpec()
		mutate(&spec)
		_, _, err := spec.Seal()
		require.Error(t, err, name)
	}
	ex := validSpec()
	ex.Mode = ModeExhibition
	ex.Slots[0].ContestEntryID, ex.Slots[1].ContestEntryID = "", ""
	ex.Slots[1].SubmissionID = "x0"
	_, _, err := ex.Seal()
	require.NoError(t, err, "exhibition allows mirror matches without entries")
}

func TestScoringPolicyClosedSchema(t *testing.T) {
	_, err := ParseScoringPolicy([]byte(`{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["wins"],"bonus":1}`))
	require.Error(t, err)
	_, err = ParseScoringPolicy([]byte(`{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0}`))
	require.Error(t, err, "tiebreakers must be declared")
	_, err = ParseScoringPolicy([]byte(`{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["rival_survival"]}`))
	require.Error(t, err, "game-specific tiebreakers are not platform rules")
	p, err := ParseScoringPolicy([]byte(`{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":2,"tiebreakers":[]}`))
	require.NoError(t, err)
	require.Equal(t, -2, p.Points(OutcomeDisqualified))
}

func TestPrincipalOwnership(t *testing.T) {
	p := Principal{UserID: "u", Capabilities: []Capability{CapAgentsUpdateOwn}}
	require.True(t, p.CanFor("u", CapAgentsUpdateOwn, CapAgentsUpdateAny))
	require.False(t, p.CanFor("v", CapAgentsUpdateOwn, CapAgentsUpdateAny))
	anon := Principal{Capabilities: []Capability{CapAgentsUpdateOwn}}
	require.False(t, anon.CanFor("", CapAgentsUpdateOwn, CapAgentsUpdateAny))
}
