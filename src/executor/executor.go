package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/engine"
	"github.com/F4nk1/Agentrix/src/game"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
)

type MatchExecutor interface {
	Execute(ctx context.Context, job *connection.MatchJob) error
}

// EngineClientFactory instantiates an EngineClient connected to the simulation engine.
type EngineClientFactory func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error)

// DefaultEngineClientFactory starts an engine subprocess based on configuration or standard paths.
func DefaultEngineClientFactory(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
	execPath := os.Getenv("AGENTRIX_ENGINE_BIN")
	if execPath == "" {
		if manifest := game.GetRegistry().GetManifest(job.GameId); manifest != nil && manifest.BinaryPath != "" {
			execPath = manifest.BinaryPath
		}
	}
	if execPath == "" {
		return nil, errors.New("Starfighter engine binary is not configured")
	}

	client := engine.NewSubprocessClient()
	err := client.Start(ctx, engine.StartConfig{
		BinaryPath:       execPath,
		HandshakeTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start engine subprocess: %w", err)
	}
	return client, nil
}

type matchExecutor struct {
	svc           service.Service
	sandbox       Sandbox
	engineFactory EngineClientFactory
}

func NewMatchExecutor(svc service.Service, sandbox Sandbox, engineFactory ...EngineClientFactory) MatchExecutor {
	if sandbox == nil {
		sandbox = NewSandbox(500 * time.Millisecond)
	}
	factory := DefaultEngineClientFactory
	if len(engineFactory) > 0 && engineFactory[0] != nil {
		factory = engineFactory[0]
	}
	return &matchExecutor{
		svc:           svc,
		sandbox:       sandbox,
		engineFactory: factory,
	}
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	if _, err := os.Stat("../../" + path); err == nil {
		return true
	}
	return false
}

func referenceBot(manifest *game.Manifest, index int) (string, error) {
	if manifest == nil || manifest.ID != "starfighter" {
		return "", errors.New("starfighter manifest is not loaded")
	}
	if len(manifest.ReferenceAgents) == 0 {
		return "", errors.New("starfighter manifest has no reference_agents")
	}
	path := manifest.ReferenceAgents[index%len(manifest.ReferenceAgents)].Path
	if !fileExists(path) {
		return "", fmt.Errorf("reference agent does not exist: %s", path)
	}
	return path, nil
}

func validatePerceptions(perceptions map[string]json.RawMessage, players []string, expectedTick int) error {
	for _, playerID := range players {
		raw, ok := perceptions[playerID]
		if !ok || !json.Valid(raw) {
			return fmt.Errorf("missing or invalid perception for %s at tick %d", playerID, expectedTick)
		}
		var envelope struct {
			Tick *int `json:"tick"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Tick == nil || *envelope.Tick != expectedTick {
			return fmt.Errorf("perception tick mismatch for %s: expected %d", playerID, expectedTick)
		}
	}
	return nil
}

func (e *matchExecutor) Execute(ctx context.Context, job *connection.MatchJob) error {
	startedAt := time.Now()
	ctx = tracer.WithMatchID(ctx, job.MatchId)
	ctx = tracer.WithJobID(ctx, job.JobId)
	ctx = tracer.WithAttempt(ctx, job.Attempt)
	tracer.InfoEvent(ctx, tracer.ScopeMatch, "match.started", "Partida iniciada",
		tracer.String("game", job.GameId))

	if job.GameId != "starfighter" {
		return fmt.Errorf("unsupported game %q: Agentrix MVP runs only starfighter", job.GameId)
	}
	manifest := game.GetRegistry().GetManifest("starfighter")
	if manifest == nil {
		return errors.New("starfighter manifest is not loaded")
	}

	match, err := e.svc.GetMatch(ctx, job.MatchId)
	if err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeDatabase, "match.load.failed", "No se pudo preparar la partida",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		return err
	}

	match.Status = common.MatchStatusRunning
	_ = e.svc.UpdateMatch(ctx, match)

	// Ensure at least 2 player slots
	submissions := make([]*model.Submission, 0)
	for _, subId := range job.SubmissionIds {
		sub, err := e.svc.GetSubmission(ctx, subId)
		if err == nil {
			submissions = append(submissions, sub)
		}
	}
	if len(submissions) > manifest.MaxPlayers {
		return fmt.Errorf("starfighter accepts exactly %d players", manifest.MaxPlayers)
	}

	// If fewer than 2 submissions provided, create reference bots with reference agent script
	if len(submissions) < 2 {
		for i := len(submissions); i < 2; i++ {
			path, err := referenceBot(manifest, i)
			if err != nil {
				return err
			}
			dummySub := &model.Submission{
				Id:       fmt.Sprintf("bot-ref-%d", i+1),
				AgentId:  fmt.Sprintf("reference-agent-%d", i+1),
				Language: "python",
				CodePath: path,
				Status:   common.SubmissionStatusReady,
				Active:   true,
			}
			submissions = append(submissions, dummySub)
		}
	}

	var playerIDs []string
	subMap := make(map[string]*model.Submission)
	for _, sub := range submissions {
		playerIDs = append(playerIDs, sub.Id)
		subMap[sub.Id] = sub
	}

	// Start engine subprocess client
	engineClient, err := e.engineFactory(ctx, job)
	if err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeMatch, "game.create.failed", "El motor de juego no pudo iniciarse",
			tracer.Origin(tracer.OriginGame), tracer.Err(err))
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = engineClient.Close(closeCtx)
	}()

	maxTicks := 100
	if manifest.MaxTicks > 0 {
		maxTicks = manifest.MaxTicks
	}
	fixedTimestepMs := manifest.FixedTimestepMs
	if fixedTimestepMs <= 0 {
		fixedTimestepMs = 17
	}
	config := make(map[string]interface{}, len(manifest.Settings))
	for key, value := range manifest.Settings {
		config[key] = value
	}

	initReq := engine.InitializeMatchRequest{
		MatchID:         job.MatchId,
		GameID:          job.GameId,
		Seed:            job.Seed,
		FixedTimestepMs: fixedTimestepMs,
		MaxTicks:        maxTicks,
		Players:         playerIDs,
		Config:          config,
	}

	initRes, err := engineClient.InitializeMatch(ctx, initReq)
	if err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeMatch, "game.initialize.failed", "El motor de juego rechazó la inicialización",
			tracer.Origin(tracer.OriginGame), tracer.Err(err))
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return err
	}
	if initRes.InitialTick != 0 {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return fmt.Errorf("engine initial tick mismatch: got %d, expected 0", initRes.InitialTick)
	}
	if err := validatePerceptions(initRes.Perceptions, playerIDs, 0); err != nil {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return err
	}

	agentIssues := make(map[string]*agentIssueSummary)
	var executionErr error
	persistenceFailures := 0
	resultFailures := 0

	replayID := uuid.New().String()
	replayRecord := &model.Replay{Id: replayID, MatchId: job.MatchId}
	replayWriter, err := e.svc.OpenReplay(ctx, replayRecord, model.ReplayMetadata{
		ReplayID: replayID, MatchID: job.MatchId, GameID: job.GameId, Seed: job.Seed,
		Participants: playerIDs, FixedTimestepMs: fixedTimestepMs, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return fmt.Errorf("open replay stream: %w", err)
	}
	defer replayWriter.Close()
	if err := replayWriter.WriteSnapshot(model.ReplaySnapshot{
		Tick: 0, PublicSnapshot: initRes.PublicSnapshot, Events: initRes.Events, StateHash: initRes.StateHash,
	}); err != nil {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return fmt.Errorf("write initial replay snapshot: %w", err)
	}

	// One persistent bot process per player receives init once and is reused
	// until the match ends or it dies, times out, or violates the protocol.
	codePaths := make(map[string]string, len(playerIDs))
	for _, pID := range playerIDs {
		if sub := subMap[pID]; sub != nil {
			codePaths[pID] = sub.CodePath
		}
	}
	botSession, err := e.sandbox.StartSession(ctx, job.MatchId, codePaths, job.Seed, 0)
	if err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeMatch, "agent.session.failed", "No se pudo iniciar la sesión de bots",
			tracer.Origin(tracer.OriginAgent), tracer.Err(err))
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return err
	}

	currentPerceptions := initRes.Perceptions
	isOver := false
	winner := ""
	currentTick := 0
	lastStateHash := initRes.StateHash
	terminationReason := ""

	// Run simulation loop over EngineClient IPC
	for !isOver && currentTick < maxTicks {
		actions := make(map[string]engine.PlayerActionInput)
		for _, pID := range playerIDs {
			perception := currentPerceptions[pID]
			input := botSession.ExecuteTurn(ctx, currentTick, pID, perception)
			actions[pID] = input

			if input.Status != engine.ActionStatusValid {
				recordAgentIssue(agentIssues, pID, statusToAgentError(input.Status))
				if input.Status == engine.ActionStatusDisqualified && input.ErrorDetails == "timeout" {
					terminationReason = "timeout"
				}
			}
		}

		tickRes, err := engineClient.AdvanceTick(ctx, engine.AdvanceTickRequest{
			Tick:    currentTick,
			Actions: actions,
		})
		if err != nil {
			executionErr = err
			tracer.ErrorEvent(ctx, tracer.ScopeMatch, "game.tick.failed", "El motor falló durante el avance de tick",
				tracer.Origin(tracer.OriginGame), tracer.Int("tick", currentTick), tracer.Err(err))
			break
		}
		if tickRes.Tick != currentTick+1 {
			executionErr = fmt.Errorf("engine tick mismatch: got state %d after action %d", tickRes.Tick, currentTick)
			break
		}
		if !tickRes.IsOver {
			if err := validatePerceptions(tickRes.Perceptions, playerIDs, tickRes.Tick); err != nil {
				executionErr = err
				break
			}
		}

		if err := replayWriter.WriteSnapshot(model.ReplaySnapshot{
			Tick: tickRes.Tick, PublicSnapshot: tickRes.PublicSnapshot,
			Events: tickRes.Events, StateHash: tickRes.StateHash,
		}); err != nil {
			executionErr = fmt.Errorf("write replay snapshot %d: %w", tickRes.Tick, err)
			break
		}

		currentPerceptions = tickRes.Perceptions
		isOver = tickRes.IsOver
		if tickRes.Winner != "" {
			winner = tickRes.Winner
		}
		currentTick = tickRes.Tick
		lastStateHash = tickRes.StateHash
	}

	if executionErr != nil {
		botSession.Close(ctx, "", "execution_error")
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		logAgentIssueSummaries(ctx, agentIssues)
		return executionErr
	}

	finishReason := "score_limit"
	if terminationReason != "" {
		finishReason = terminationReason
	} else if winner != "" {
		finishReason = "eliminated"
	}

	botSession.Close(ctx, winner, finishReason)

	matchRes, err := engineClient.FinishMatch(ctx, finishReason)
	if err != nil {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return fmt.Errorf("finish Starfighter match: %w", err)
	}
	if matchRes == nil {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return errors.New("engine returned no final match result")
	}
	if matchRes.FinalTick != currentTick {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return fmt.Errorf("engine final tick mismatch: got %d, expected %d", matchRes.FinalTick, currentTick)
	}
	if matchRes.Reason != finishReason {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return fmt.Errorf("engine final reason mismatch: got %q, expected %q", matchRes.Reason, finishReason)
	}
	if matchRes.FinalStateHash != lastStateHash {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return fmt.Errorf("engine final state hash does not match state %d", currentTick)
	}
	if matchRes.Scores == nil {
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return errors.New("engine returned no final scores")
	}
	scores := matchRes.Scores
	rankings := matchRes.Rankings
	finalWinner := winner
	if matchRes.Winner != "" {
		finalWinner = matchRes.Winner
	}

	if err := replayWriter.Complete(model.ReplayResult{
		FinalTick: currentTick, Winner: finalWinner, Scores: scores,
		Reason: finishReason, FinalStateHash: lastStateHash, FinishedAt: time.Now().UTC(),
	}); err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeReplay, "replay.save.failed", "No se pudo sellar el replay",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return fmt.Errorf("seal authoritative replay: %w", err)
	}
	replayRecord.DurationTicks = replayWriter.FrameCount()

	// Build rankings list if not supplied by engine
	if len(rankings) == 0 {
		type rankedScore struct {
			SubID string
			Score int
		}
		var rankedList []rankedScore
		for subID, sc := range scores {
			rankedList = append(rankedList, rankedScore{SubID: subID, Score: sc})
		}
		sort.Slice(rankedList, func(i, j int) bool {
			return rankedList[i].Score > rankedList[j].Score
		})
		for idx, item := range rankedList {
			rankings = append(rankings, engine.PlayerRank{
				PlayerID: item.SubID,
				Rank:     idx + 1,
				Score:    item.Score,
			})
		}
	}

	// Save individual results
	for _, item := range rankings {
		status := "finished"
		if item.Score <= 0 && len(rankings) > 1 {
			status = "eliminated"
		}

		res := &model.Result{
			Id:           uuid.New().String(),
			MatchId:      job.MatchId,
			SubmissionId: item.PlayerID,
			Score:        item.Score,
			Rank:         item.Rank,
			Status:       status,
			Details:      fmt.Sprintf("Score: %d, Rank: %d", item.Score, item.Rank),
			CreatedAt:    time.Now().UTC(),
		}
		if err := e.svc.CreateResult(ctx, res); err != nil {
			persistenceFailures++
			resultFailures++
		}
	}
	if resultFailures > 0 {
		tracer.ErrorEvent(ctx, tracer.ScopeDatabase, "results.save.failed", "No se pudieron guardar todos los resultados",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Int("failed_results", resultFailures))
	}

	// Mark match status
	now := time.Now().UTC()
	match.Status = common.MatchStatusFinished
	match.ReplayId = replayID
	match.FinishedAt = &now
	if err := e.svc.UpdateMatch(ctx, match); err != nil {
		persistenceFailures++
		tracer.ErrorEvent(ctx, tracer.ScopeDatabase, "match.finish.failed", "No se pudo guardar el estado final de la partida",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	}

	// Update rankings for contest if contest is set
	if job.ContestId != "" {
		if _, err := e.svc.CalculateRankings(ctx, job.ContestId); err != nil {
			persistenceFailures++
			tracer.ErrorEvent(ctx, tracer.ScopeDatabase, "rankings.update.failed", "No se pudo actualizar la clasificación",
				tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		}
	}

	logAgentIssueSummaries(ctx, agentIssues)
	completionFields := []tracer.Field{
		tracer.String("winner_id", finalWinner),
		tracer.Int("ticks", replayWriter.FrameCount()),
		tracer.Duration("elapsed", time.Since(startedAt)),
	}
	if persistenceFailures > 0 {
		completionFields = append(completionFields, tracer.Int("persistence_failures", persistenceFailures))
		tracer.WarnEvent(ctx, tracer.ScopeMatch, "match.completed_with_incidents", "Partida finalizada con incidencias", completionFields...)
	} else {
		tracer.InfoEvent(ctx, tracer.ScopeMatch, "match.completed", "Partida finalizada", completionFields...)
	}
	return nil
}

type agentIssueSummary struct {
	total          int
	timeouts       int
	invalidActions int
	unavailable    int
	execution      int
}

// statusToAgentError maps a bot protocol status into the incident summary.
func statusToAgentError(status string) error {
	switch status {
	case engine.ActionStatusTimeout:
		return ErrAgentTimeout
	case engine.ActionStatusInvalidOutput:
		return ErrAgentInvalidAction
	case engine.ActionStatusCrashed:
		return ErrAgentExecution
	case engine.ActionStatusDisqualified:
		return ErrAgentTimeout
	default:
		return ErrAgentExecution
	}
}

func recordAgentIssue(summaries map[string]*agentIssueSummary, playerID string, err error) {
	summary := summaries[playerID]
	if summary == nil {
		summary = &agentIssueSummary{}
		summaries[playerID] = summary
	}
	summary.total++
	switch {
	case errors.Is(err, ErrAgentTimeout):
		summary.timeouts++
	case errors.Is(err, ErrAgentInvalidAction):
		summary.invalidActions++
	case errors.Is(err, ErrAgentUnavailable):
		summary.unavailable++
	default:
		summary.execution++
	}
}

func logAgentIssueSummaries(ctx context.Context, summaries map[string]*agentIssueSummary) {
	playerIDs := make([]string, 0, len(summaries))
	for playerID := range summaries {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Strings(playerIDs)
	for _, playerID := range playerIDs {
		summary := summaries[playerID]
		tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.incidents.summary", "Incidencias del agente",
			tracer.Origin(tracer.OriginAgent),
			tracer.String("agent_id", playerID),
			tracer.Int("total", summary.total),
			tracer.Int("timeouts", summary.timeouts),
			tracer.Int("invalid_actions", summary.invalidActions),
			tracer.Int("unavailable", summary.unavailable),
			tracer.Int("execution_failures", summary.execution),
		)
	}
}

func structToMap(obj interface{}) (map[string]interface{}, error) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	err = json.Unmarshal(bytes, &res)
	return res, err
}
