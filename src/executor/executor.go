package executor

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/replay"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/google/uuid"
)

// RunError classifies an execution failure. The worker reports the class
// to FailRun, which decides whether a new attempt is scheduled.
type RunError struct {
	Class model.ErrorClass
	Err   error
}

func (e *RunError) Error() string { return fmt.Sprintf("%s: %v", e.Class, e.Err) }
func (e *RunError) Unwrap() error { return e.Err }

func fail(class model.ErrorClass, format string, args ...any) *RunError {
	return &RunError{Class: class, Err: fmt.Errorf(format, args...)}
}

// EngineFactory starts an engine process whose binary digest must equal
// expectedSHA (verified by the client before the handshake).
type EngineFactory func(ctx context.Context, expectedSHA string) (engine.EngineClient, error)

// SubprocessEngineFactory launches the configured engine binary.
func SubprocessEngineFactory(binary string) EngineFactory {
	return func(ctx context.Context, expectedSHA string) (engine.EngineClient, error) {
		client := engine.NewSubprocessClient()
		if err := client.Start(ctx, engine.StartConfig{BinaryPath: binary, ExpectedDigest: expectedSHA,
			HandshakeTimeout: 10 * time.Second}); err != nil {
			return nil, err
		}
		return client, nil
	}
}

type Executor struct {
	runtime   BotRuntime
	artifacts *connection.ArtifactStore
	engines   EngineFactory
}

func NewExecutor(runtime BotRuntime, artifacts *connection.ArtifactStore, engines EngineFactory) *Executor {
	return &Executor{runtime: runtime, artifacts: artifacts, engines: engines}
}

// Execute runs a reserved run exclusively from its persisted ExecutionSpec.
func (e *Executor) Execute(ctx context.Context, res model.Reservation) (*model.RunCompletion, error) {
	spec := res.Spec
	if err := spec.Validate(); err != nil {
		return nil, &RunError{Class: model.ErrorSpec, Err: err}
	}
	if spec.ReplayFormat != replay.FormatVersion {
		return nil, fail(model.ErrorSpec, "replay format %q is not supported by this worker", spec.ReplayFormat)
	}
	wallCtx, cancel := context.WithTimeout(ctx, time.Duration(spec.Limits.WallTimeMs)*time.Millisecond)
	defer cancel()

	launches := make([]BotLaunch, 0, len(spec.Slots))
	digests := make(map[string]string, len(spec.Slots))
	players := make([]string, 0, len(spec.Slots))
	participants := make([]replay.Participant, 0, len(spec.Slots))
	for _, slot := range spec.Slots {
		if err := e.artifacts.Verify(slot.ArtifactKey, slot.ArtifactSHA256); err != nil {
			return nil, fail(model.ErrorArtifact, "slot %d artifact cannot be verified: %v", slot.Index, err)
		}
		path, err := e.artifacts.Path(slot.ArtifactKey)
		if err != nil {
			return nil, fail(model.ErrorArtifact, "slot %d artifact key is invalid", slot.Index)
		}
		launches = append(launches, BotLaunch{PlayerID: slot.SlotID, CodePath: path, Runtime: slot.Runtime})
		digests[slot.SlotID] = slot.ArtifactSHA256
		players = append(players, slot.SlotID)
		participants = append(participants, replay.Participant{PlayerID: slot.SlotID, SlotIndex: slot.Index})
	}

	client, err := e.engines(wallCtx, spec.Engine.SHA256)
	if err != nil {
		if errors.Is(err, engine.ErrEngineDigestMismatch) {
			return nil, fail(model.ErrorSpec, "engine binary does not match the pinned digest")
		}
		return nil, fail(model.ErrorInfrastructure, "engine start failed: %v", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer closeCancel()
		_ = client.Close(closeCtx)
	}()
	if client.EngineVersion() != spec.Engine.Version {
		return nil, fail(model.ErrorSpec, "engine version %q does not match spec %q", client.EngineVersion(), spec.Engine.Version)
	}

	init, err := client.InitializeMatch(wallCtx, engine.InitializeMatchRequest{
		MatchID: spec.MatchID, GameID: spec.Game.ID, GameVersion: spec.Game.Version, ExpectedEngineVersion: spec.Engine.Version,
		Seed: spec.Seed, TickRate: engine.TickRate{Numerator: spec.TickRate.Numerator, Denominator: spec.TickRate.Denominator},
		MaxTicks: spec.Limits.MaxTicks, Players: players, Config: spec.Game.Config, ParticipantArtifactDigests: digests,
	})
	if err != nil {
		return nil, fail(model.ErrorEngine, "engine rejected initialization: %v", err)
	}
	if init.InitialTick != 0 {
		return nil, fail(model.ErrorEngine, "engine initial tick is %d, expected 0", init.InitialTick)
	}

	replayID := uuid.NewString()
	stagingKey, storageKey := service.ReplayKeys(res.RunID, "gzip")
	pending, err := e.artifacts.Create(stagingKey)
	if err != nil {
		return nil, fail(model.ErrorInfrastructure, "open replay staging: %v", err)
	}
	committedReplay := false
	defer func() {
		if !committedReplay {
			_ = pending.Abort()
		}
	}()
	gz := gzip.NewWriter(pending)
	writer, err := replay.NewWriter(gz, replay.Metadata{
		ReplayID: replayID, MatchID: spec.MatchID, RunID: res.RunID, SpecHash: mustHash(spec), GameID: spec.Game.ID,
		GameVersion: spec.Game.Version, ConfigHash: spec.Game.ConfigHash, EngineVersion: spec.Engine.Version,
		EngineSHA256: spec.Engine.SHA256, Seed: spec.Seed,
		TickRate:     replay.TickRate{Numerator: spec.TickRate.Numerator, Denominator: spec.TickRate.Denominator},
		Participants: participants, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, fail(model.ErrorInfrastructure, "write replay metadata: %v", err)
	}
	if err := writer.WriteSnapshot(replay.Snapshot{Tick: 0, StateHash: init.StateHash, Events: init.Events,
		PublicSnapshot: init.PublicSnapshot}); err != nil {
		return nil, fail(model.ErrorEngine, "initial snapshot rejected: %v", err)
	}

	session, err := StartSession(wallCtx, e.runtime, spec.MatchID, spec.Game.ID, spec.Seed, spec.Limits, launches)
	if err != nil {
		return nil, fail(model.ErrorSandbox, "%v", err)
	}
	sessionClosed := false
	defer func() {
		if !sessionClosed {
			session.Close("", "aborted")
		}
	}()

	perceptions := init.Perceptions
	tick, stateHash, winner, over := 0, init.StateHash, "", false
	for !over {
		if err := wallCtx.Err(); err != nil {
			if ctx.Err() != nil {
				return nil, &RunError{Class: model.ErrorLeaseLost, Err: ctx.Err()}
			}
			return nil, fail(model.ErrorInfrastructure, "run exceeded wall time of %dms", spec.Limits.WallTimeMs)
		}
		if tick >= spec.Limits.MaxTicks {
			return nil, fail(model.ErrorEngine, "engine did not finish within max_ticks")
		}
		actions := session.Turn(wallCtx, tick, perceptions)
		result, err := client.AdvanceTick(wallCtx, engine.AdvanceTickRequest{Tick: tick, Actions: actions})
		if err != nil {
			if ctx.Err() != nil {
				return nil, &RunError{Class: model.ErrorLeaseLost, Err: ctx.Err()}
			}
			return nil, fail(model.ErrorEngine, "advance tick %d: %v", tick, err)
		}
		if result.Tick != tick+1 {
			return nil, fail(model.ErrorEngine, "engine returned state %d after action %d", result.Tick, tick)
		}
		if err := writer.WriteSnapshot(replay.Snapshot{Tick: result.Tick, StateHash: result.StateHash, Events: result.Events,
			PublicSnapshot: result.PublicSnapshot}); err != nil {
			return nil, fail(model.ErrorEngine, "snapshot %d rejected: %v", result.Tick, err)
		}
		tick, stateHash, perceptions, over = result.Tick, result.StateHash, result.Perceptions, result.IsOver
		if result.Winner != "" {
			winner = result.Winner
		}
	}

	reason := "score_limit"
	if dq := session.Disqualifications(); len(dq) > 0 {
		reason = "timeout"
	} else if winner != "" {
		reason = "eliminated"
	}
	session.Close(winner, reason)
	sessionClosed = true
	final, err := client.FinishMatch(wallCtx, reason)
	if err != nil {
		return nil, fail(model.ErrorEngine, "finish match: %v", err)
	}
	if final.FinalTick != tick || final.FinalStateHash != stateHash {
		return nil, fail(model.ErrorEngine, "engine final state does not match the last simulated tick")
	}
	if final.Reason != reason {
		return nil, fail(model.ErrorEngine, "engine final reason %q, expected %q", final.Reason, reason)
	}
	ranks := map[string]engine.PlayerRank{}
	for _, r := range final.Rankings {
		ranks[r.PlayerID] = r
	}
	disq := session.Disqualifications()
	outcomes := make([]model.SlotOutcome, 0, len(players))
	for _, id := range players {
		rank, ok := ranks[id]
		score, scored := final.Scores[id]
		if !ok || !scored {
			return nil, fail(model.ErrorEngine, "engine returned no rank or score for slot %s", id)
		}
		details := map[string]any{}
		if cause, ok := disq[id]; ok {
			details["disqualification"] = cause
		}
		outcomes = append(outcomes, model.SlotOutcome{SlotID: id, Score: score, Rank: rank.Rank, Disqualified: disq[id] != "", Details: details})
	}
	if final.Winner != "" {
		winner = final.Winner
	}
	if err := writer.Complete(replay.Result{FinalTick: tick, Winner: winner, Scores: final.Scores, Reason: reason,
		FinalStateHash: stateHash, FinishedAt: time.Now().UTC()}); err != nil {
		return nil, fail(model.ErrorInfrastructure, "seal replay: %v", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fail(model.ErrorInfrastructure, "compress replay: %v", err)
	}
	if err := pending.Commit(); err != nil {
		return nil, fail(model.ErrorInfrastructure, "commit replay staging: %v", err)
	}
	committedReplay = true
	sha, size, err := e.artifacts.Digest(stagingKey)
	if err != nil {
		return nil, fail(model.ErrorInfrastructure, "digest replay: %v", err)
	}
	return &model.RunCompletion{Reservation: res, TerminationReason: reason, FinalTick: tick, FinalStateHash: stateHash,
		Slots: outcomes, Replay: model.ReplayArtifact{ID: replayID, FormatVersion: spec.ReplayFormat, Compression: "gzip",
			StagingKey: stagingKey, StorageKey: storageKey, SHA256: sha, SizeBytes: size, FrameCount: writer.FrameCount()}}, nil
}

func mustHash(spec model.ExecutionSpec) string {
	h, err := spec.Hash()
	if err != nil {
		return ""
	}
	return h
}

// DiscardStagedReplay removes the staged replay of a run that will not be
// committed. Called by the worker after FailRun.
func (e *Executor) DiscardStagedReplay(runID string) error {
	staging, _ := service.ReplayKeys(runID, "gzip")
	return e.artifacts.Delete(staging)
}
