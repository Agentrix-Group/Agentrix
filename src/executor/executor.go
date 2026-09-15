package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/game"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
)

type MatchExecutor interface {
	Execute(ctx context.Context, job *connection.MatchJob) error
}

type matchExecutor struct {
	svc     service.Service
	sandbox Sandbox
}

func NewMatchExecutor(svc service.Service, sandbox Sandbox) MatchExecutor {
	if sandbox == nil {
		sandbox = NewSandbox(500 * time.Millisecond)
	}
	return &matchExecutor{
		svc:     svc,
		sandbox: sandbox,
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

	engine, err := game.GetRegistry().CreateEngine(job.GameId)
	if err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeMatch, "game.create.failed", "El juego no pudo iniciarse",
			tracer.Origin(tracer.OriginGame), tracer.Err(err))
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return err
	}

	var playerIDs []string
	subMap := make(map[string]*model.Submission)
	for _, sub := range submissions {
		playerIDs = append(playerIDs, sub.Id)
		subMap[sub.Id] = sub
	}

	state, err := engine.Init(playerIDs, job.Seed)
	if err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeMatch, "game.initialize.failed", "El juego rechazó la configuración inicial",
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
	initialStateMap, _ := structToMap(state)
	replayFrames = append(replayFrames, model.ReplayFrame{
		Tick:   0,
		Events: state.Events,
		State:  initialStateMap,
	})

	// Run game simulation loop
	for !engine.IsOver() {
		actions := make(map[string]game.Action)

		for _, pID := range playerIDs {
			sub := subMap[pID]
			codePath := ""
			if sub != nil {
				codePath = sub.CodePath
			}

			action, err := e.sandbox.ExecuteTurn(ctx, codePath, state, pID)
			if err != nil {
				recordAgentIssue(agentIssues, pID, err)
				action = game.Action{Type: game.ActionRest}
			}
			actions[pID] = action
		}

		state, err = engine.Step(actions)
		if err != nil {
			executionErr = err
			tracer.ErrorEvent(ctx, tracer.ScopeMatch, "game.tick.failed", "El juego falló durante la partida",
				tracer.Origin(tracer.OriginGame), tracer.Int("tick", state.Tick), tracer.Err(err))
			break
		}

		stateMap, _ := structToMap(state)
		actionMap := make(map[string]interface{})
		for pID, act := range actions {
			actionMap[pID] = act.Type
		}

		replayFrames = append(replayFrames, model.ReplayFrame{
			Tick:    state.Tick,
			Events:  state.Events,
			State:   stateMap,
			Actions: actionMap,
		})
	}

	// Collect final results
	scores := engine.GetResults()
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

	// Save match replay
	replayID := uuid.New().String()
	replayData := &model.ReplayData{
		GameId:   job.GameId,
		MatchId:  job.MatchId,
		Seed:     job.Seed,
		Players:  playerIDs,
		MaxTicks: engine.GetManifest().MaxTicks,
		Frames:   replayFrames,
		Winner:   state.Winner,
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

	// Save individual results
	for rankIdx, item := range rankedList {
		status := "finished"
		if item.Score <= 0 {
			status = "eliminated"
		}

		res := &model.Result{
			Id:           uuid.New().String(),
			MatchId:      job.MatchId,
			SubmissionId: item.SubID,
			Score:        item.Score,
			Rank:         rankIdx + 1,
			Status:       status,
			Details:      fmt.Sprintf("Score: %d, Rank: %d", item.Score, rankIdx+1),
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

	// Mark match finished
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
		tracer.String("winner_id", state.Winner),
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
