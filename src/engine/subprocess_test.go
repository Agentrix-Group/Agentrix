package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var fakeEngineBinary string

func TestMain(m *testing.M) {
	// Build fake engine binary once for testing
	tempDir, err := os.MkdirTemp("", "fake-engine-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	binPath := filepath.Join(tempDir, "fake-engine")
	cmd := exec.Command("go", "build", "-o", binPath, "../../cmd/fake-engine")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(string(out))
	}
	fakeEngineBinary = binPath

	code := m.Run()
	os.Exit(code)
}

func TestSubprocessLifecycle_Success(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := NewSubprocessClient()
	cfg := StartConfig{
		BinaryPath:       fakeEngineBinary,
		Args:             []string{"--mode=normal"},
		HandshakeTimeout: 3 * time.Second,
	}

	err := client.Start(ctx, cfg)
	r.NoError(err)

	// InitializeMatch
	initRes, err := client.InitializeMatch(ctx, InitializeMatchRequest{
		MatchID:         "m-test-1",
		GameID:          "starfighter",
		Seed:            42,
		FixedTimestepMs: 50,
		MaxTicks:        10,
		Players:         []string{"bot-1", "bot-2"},
	})
	r.NoError(err)
	r.Equal("m-test-1", initRes.MatchID)
	r.Equal(0, initRes.InitialTick)
	r.NotEmpty(initRes.StateHash)
	r.Contains(initRes.Perceptions, "bot-1")

	// Action tick 0 produces state tick 1.
	tickRes, err := client.AdvanceTick(ctx, AdvanceTickRequest{
		Tick: 0,
		Actions: map[string]PlayerActionInput{
			"bot-1": {Status: ActionStatusValid, Payload: json.RawMessage(`{"thrust":"FORWARD"}`)},
			"bot-2": {Status: ActionStatusValid, Payload: json.RawMessage(`{"shoot":true}`)},
		},
	})
	r.NoError(err)
	r.Equal(1, tickRes.Tick)
	r.False(tickRes.IsOver)
	r.NotEmpty(tickRes.StateHash)
	r.NotEmpty(tickRes.Events)

	// FinishMatch
	matchRes, err := client.FinishMatch(ctx, "normal_finish")
	r.NoError(err)
	r.Equal("normal_finish", matchRes.Reason)
	r.NotEmpty(matchRes.FinalStateHash)

	// Clean Close
	r.NoError(client.Close(ctx))
}

func TestSubprocess_ExecutableNotFound(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	client := NewSubprocessClient()
	err := client.Start(ctx, StartConfig{
		BinaryPath: "/path/to/nonexistent/engine-binary",
	})
	r.Error(err)
	r.ErrorIs(err, ErrExecutableNotFound)
}

func TestSubprocess_TimeoutOnHandshake(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	client := NewSubprocessClient()
	err := client.Start(ctx, StartConfig{
		BinaryPath:       fakeEngineBinary,
		Args:             []string{"--mode=timeout_on_start"},
		HandshakeTimeout: 50 * time.Millisecond,
	})
	r.Error(err)
}

func TestSubprocess_InvalidJSONOnStart(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	client := NewSubprocessClient()
	err := client.Start(ctx, StartConfig{
		BinaryPath:       fakeEngineBinary,
		Args:             []string{"--mode=invalid_json_on_start"},
		HandshakeTimeout: 2 * time.Second,
	})
	r.Error(err)
	r.ErrorIs(err, ErrInvalidJSON)
}

func TestSubprocess_IncompatibleVersion(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	client := NewSubprocessClient()
	err := client.Start(ctx, StartConfig{
		BinaryPath:       fakeEngineBinary,
		Args:             []string{"--mode=incompatible_version"},
		HandshakeTimeout: 2 * time.Second,
	})
	r.Error(err)
	r.ErrorIs(err, ErrIncompatibleVersion)
}

func TestSubprocess_CrashOnStart(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	client := NewSubprocessClient()
	err := client.Start(ctx, StartConfig{
		BinaryPath:       fakeEngineBinary,
		Args:             []string{"--mode=crash_on_start"},
		HandshakeTimeout: 2 * time.Second,
	})
	r.Error(err)
}

func TestSubprocess_EngineDeclaredError(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	client := NewSubprocessClient()
	err := client.Start(ctx, StartConfig{
		BinaryPath: fakeEngineBinary,
		Args:       []string{"--mode=engine_error"},
	})
	r.NoError(err)
	defer client.Close(ctx)

	_, err = client.InitializeMatch(ctx, InitializeMatchRequest{
		MatchID: "m-err",
		GameID:  "starfighter",
		Players: []string{"b1"},
	})
	r.Error(err)
	var declaredErr *EngineDeclaredError
	r.ErrorAs(err, &declaredErr)
	r.Equal("ERR_FAKE_SIMULATION_ABORT", declaredErr.Code)
	r.True(declaredErr.Fatal)
}

func TestSubprocess_InvalidSequence(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	client := NewSubprocessClient()
	err := client.Start(ctx, StartConfig{
		BinaryPath: fakeEngineBinary,
		Args:       []string{"--mode=invalid_sequence"},
	})
	r.NoError(err)
	defer client.Close(ctx)

	_, err = client.InitializeMatch(ctx, InitializeMatchRequest{
		MatchID: "m-seq",
		GameID:  "starfighter",
		Players: []string{"b1"},
	})
	r.Error(err)
	r.ErrorIs(err, ErrInvalidSequence)
}

func TestSubprocess_LargeMessageRejected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	client := NewSubprocessClient()
	err := client.Start(ctx, StartConfig{
		BinaryPath:   fakeEngineBinary,
		Args:         []string{"--mode=large_message"},
		MaxLineBytes: 4096, // Limit to 4 KB
	})
	r.NoError(err)
	defer client.Close(ctx)

	_, err = client.InitializeMatch(ctx, InitializeMatchRequest{
		MatchID: "m-large",
		GameID:  "starfighter",
		Players: []string{"b1"},
	})
	r.NoError(err)

	_, err = client.AdvanceTick(ctx, AdvanceTickRequest{Tick: 0})
	r.Error(err)
	r.ErrorIs(err, ErrLineTooLong)
}

func TestDeterminism_FakeEngine(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	runSimulation := func(seed int64, actions []string) (string, []string) {
		client := NewSubprocessClient()
		err := client.Start(ctx, StartConfig{
			BinaryPath: fakeEngineBinary,
			Args:       []string{"--mode=normal"},
		})
		r.NoError(err)
		defer client.Close(ctx)

		_, err = client.InitializeMatch(ctx, InitializeMatchRequest{
			MatchID:         "m-det",
			GameID:          "starfighter",
			Seed:            seed,
			FixedTimestepMs: 50,
			MaxTicks:        int(len(actions)),
			Players:         []string{"bot-1"},
		})
		r.NoError(err)

		var lastHash string
		var allEvents []string
		for i, act := range actions {
			res, err := client.AdvanceTick(ctx, AdvanceTickRequest{
				Tick: i,
				Actions: map[string]PlayerActionInput{
					"bot-1": {Status: ActionStatusValid, Payload: json.RawMessage(fmt.Sprintf(`{"action":%q}`, act))},
				},
			})
			r.NoError(err)
			lastHash = res.StateHash
			allEvents = append(allEvents, res.Events...)
		}
		return lastHash, allEvents
	}

	actions := []string{"UP", "RIGHT", "ATTACK", "SHIELD"}

	// Two executions with same seed and same actions
	hash1, events1 := runSimulation(12345, actions)
	hash2, events2 := runSimulation(12345, actions)

	r.NotEmpty(hash1)
	r.Equal(hash1, hash2, "Identical runs with same seed must produce identical stateHash")
	r.Equal(events1, events2, "Identical runs with same seed must produce identical events")

	// Run with different seed
	hashDiff, _ := runSimulation(99999, actions)
	r.NotEqual(hash1, hashDiff, "Different seeds must produce different state hashes")
}
