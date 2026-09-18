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
		if _, err := os.Stat("bin/agentrix-engine"); err == nil {
			execPath = "bin/agentrix-engine"
		} else if _, err := os.Stat("bin/fake-engine"); err == nil {
			execPath = "bin/fake-engine"
		} else {
			return nil, errors.New("no engine binary found (checked AGENTRIX_ENGINE_BIN, bin/agentrix-engine, bin/fake-engine)")
		}
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

func (e *matchExecutor) Execute(ctx context.Context, job *connection.MatchJob) error {
	startedAt := time.Now()
	ctx = tracer.WithMatchID(ctx, job.MatchId)
	ctx = tracer.WithJobID(ctx, job.JobId)
	ctx = tracer.WithAttempt(ctx, job.Attempt)
	tracer.InfoEvent(ctx, tracer.ScopeMatch, "match.started", "Partida iniciada",
		tracer.String("game", job.GameId))

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

	// If fewer than 2 submissions provided, create reference bots with reference agent script
	if len(submissions) < 2 {
		for i := len(submissions); i < 2; i++ {
			dummySub := &model.Submission{
				Id:       fmt.Sprintf("bot-ref-%d", i+1),
				AgentId:  fmt.Sprintf("reference-agent-%d", i+1),
				Language: "python",
				CodePath: "games/arena-basica/examples/bot_hunter.py",
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
	if manifest := game.GetRegistry().GetManifest(job.GameId); manifest != nil && manifest.MaxTicks > 0 {
		maxTicks = manifest.MaxTicks
	}

	initReq := engine.InitializeMatchRequest{
		MatchID:         job.MatchId,
		GameID:          job.GameId,
		Seed:            job.Seed,
		FixedTimestepMs: 50,
		MaxTicks:        maxTicks,
		Players:         playerIDs,
	}

	initRes, err := engineClient.InitializeMatch(ctx, initReq)
	if err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeMatch, "game.initialize.failed", "El motor de juego rechazó la inicialización",
			tracer.Origin(tracer.OriginGame), tracer.Err(err))
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return err
	}

	replayFrames := make([]model.ReplayFrame, 0)
	agentIssues := make(map[string]*agentIssueSummary)
	var executionErr error
	persistenceFailures := 0
	resultFailures := 0

	// Record initial frame
	replayFrames = append(replayFrames, model.ReplayFrame{
		Tick:   0,
		Events: initRes.Events,
		State: map[string]interface{}{
			"stateHash":   initRes.StateHash,
			"perceptions": initRes.Perceptions,
		},
	})

	// One persistent bot process per player for the whole match, instead
	// of a fresh process per tick per bot (RF-044/ATD-007-style: the
	// process is started once, handshakes once, and is reused every tick
	// until the match ends or it dies/times out/misbehaves).
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
	currentTick := 1

	// Run simulation loop over EngineClient IPC
	for !isOver && currentTick <= maxTicks {
		actions := make(map[string]engine.PlayerActionInput)
		actionMapForReplay := make(map[string]interface{})

		for _, pID := range playerIDs {
			perception := currentPerceptions[pID]
			input := botSession.ExecuteTurn(ctx, currentTick, pID, perception)
			actions[pID] = input

			if input.Status != engine.ActionStatusValid {
				recordAgentIssue(agentIssues, pID, statusToAgentError(input.Status))
				actionMapForReplay[pID] = "REST"
			} else {
				actionMapForReplay[pID] = input.ActionType
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

		frameState := map[string]interface{}{
			"stateHash":   tickRes.StateHash,
			"publicState": tickRes.PublicState,
		}
		replayFrames = append(replayFrames, model.ReplayFrame{
			Tick:    tickRes.Tick,
			Events:  tickRes.Events,
			State:   frameState,
			Actions: actionMapForReplay,
		})

		currentPerceptions = tickRes.Perceptions
		isOver = tickRes.IsOver
		if tickRes.Winner != "" {
			winner = tickRes.Winner
		}
		currentTick++
	}

	finishReason := "time_limit"
	if executionErr != nil {
		finishReason = "aborted"
	} else if winner != "" {
		finishReason = "victory"
	}

	botSession.Close(ctx, winner, finishReason)

	matchRes, err := engineClient.FinishMatch(ctx, finishReason)
	var scores map[string]int
	var rankings []engine.PlayerRank
	finalWinner := winner
	if err == nil && matchRes != nil {
		scores = matchRes.Scores
		rankings = matchRes.Rankings
		if matchRes.Winner != "" {
			finalWinner = matchRes.Winner
		}
	} else {
		if executionErr == nil {
			executionErr = err
		}
		scores = make(map[string]int)
		for _, pID := range playerIDs {
			scores[pID] = 0
		}
	}

	// Save match replay
	replayID := uuid.New().String()
	replayData := &model.ReplayData{
		GameId:   job.GameId,
		MatchId:  job.MatchId,
		Seed:     job.Seed,
		Players:  playerIDs,
		MaxTicks: maxTicks,
		Frames:   replayFrames,
		Winner:   finalWinner,
		Scores:   scores,
	}

	replay := &model.Replay{
		Id:      replayID,
		MatchId: job.MatchId,
	}
	if err := e.svc.SaveReplay(ctx, replay, replayData); err != nil {
		persistenceFailures++
		tracer.ErrorEvent(ctx, tracer.ScopeReplay, "replay.save.failed", "No se pudo guardar el replay",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	}

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
	if executionErr != nil {
		match.Status = common.MatchStatusFailed
	} else {
		match.Status = common.MatchStatusFinished
	}
	match.ReplayId = replayID
	match.FinishedAt = &now
	if err := e.svc.UpdateMatch(ctx, match); err != nil {
		persistenceFailures++
		tracer.ErrorEvent(ctx, tracer.ScopeDatabase, "match.finish.failed", "No se pudo guardar el estado final de la partida",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	}

	// Update rankings for contest if contest is set
	if job.ContestId != "" && executionErr == nil {
		if _, err := e.svc.CalculateRankings(ctx, job.ContestId); err != nil {
			persistenceFailures++
			tracer.ErrorEvent(ctx, tracer.ScopeDatabase, "rankings.update.failed", "No se pudo actualizar la clasificación",
				tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		}
	}

	logAgentIssueSummaries(ctx, agentIssues)
	completionFields := []tracer.Field{
		tracer.String("winner_id", finalWinner),
		tracer.Int("ticks", len(replayFrames)),
		tracer.Duration("elapsed", time.Since(startedAt)),
	}
	if executionErr != nil || persistenceFailures > 0 {
		completionFields = append(completionFields, tracer.Int("persistence_failures", persistenceFailures))
		tracer.WarnEvent(ctx, tracer.ScopeMatch, "match.completed_with_incidents", "Partida finalizada con incidencias", completionFields...)
	} else {
		tracer.InfoEvent(ctx, tracer.ScopeMatch, "match.completed", "Partida finalizada", completionFields...)
	}
	return executionErr
}

type agentIssueSummary struct {
	total          int
	timeouts       int
	invalidActions int
	unavailable    int
	execution      int
}

// statusToAgentError maps a BotSession.ExecuteTurn status back to one of
// the sentinel errors recordAgentIssue already classifies by, so the
// per-match agent incident summary keeps working unchanged after switching
// the live loop from ExecuteTurnWithPerception (which returned an error) to
// BotSession.ExecuteTurn (which returns a status string and never an
// error, per RF-044 -- a failed agent turn is not a platform error).
func statusToAgentError(status string) error {
	switch status {
	case engine.ActionStatusTimeout:
		return ErrAgentTimeout
	case engine.ActionStatusInvalidOutput:
		return ErrAgentInvalidAction
	case engine.ActionStatusCrashed:
		return ErrAgentExecution
	case engine.ActionStatusDisqualified:
		return ErrAgentUnavailable
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
