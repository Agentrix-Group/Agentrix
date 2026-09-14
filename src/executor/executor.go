package executor

import (
	"context"
	"encoding/json"
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
	tracer.Infof(ctx, "Starting execution for match %s (game: %s)", job.MatchId, job.GameId)

	match, err := e.svc.GetMatch(ctx, job.MatchId)
	if err != nil {
		tracer.Errorf(ctx, "Failed to get match %s: %s", job.MatchId, err)
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

	// If fewer than 2 submissions provided, create dummy bots for basic simulation
	if len(submissions) < 2 {
		for i := len(submissions); i < 2; i++ {
			dummySub := &model.Submission{
				Id:       fmt.Sprintf("bot-ref-%d", i+1),
				AgentId:  fmt.Sprintf("reference-agent-%d", i+1),
				Language: "python",
				Status:   common.SubmissionStatusReady,
				Active:   true,
			}
			submissions = append(submissions, dummySub)
		}
	}

	engine, err := game.GetRegistry().CreateEngine(job.GameId)
	if err != nil {
		tracer.Errorf(ctx, "Failed to create game engine for %s: %s", job.GameId, err)
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
		tracer.Errorf(ctx, "Failed to init game engine: %s", err)
		match.Status = common.MatchStatusFailed
		_ = e.svc.UpdateMatch(ctx, match)
		return err
	}

	replayFrames := make([]model.ReplayFrame, 0)

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
				action = game.Action{Type: game.ActionRest}
			}
			actions[pID] = action
		}

		state, err = engine.Step(actions)
		if err != nil {
			tracer.Errorf(ctx, "Engine step error at tick %d: %s", state.Tick, err)
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
		tracer.Errorf(ctx, "Failed to save replay for match %s: %s", job.MatchId, err)
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
			tracer.Warnf(ctx, "Failed to persist result for submission %s: %s", item.SubID, err)
		}
	}

	// Mark match finished
	now := time.Now().UTC()
	match.Status = common.MatchStatusFinished
	match.ReplayId = replayID
	match.FinishedAt = &now
	if err := e.svc.UpdateMatch(ctx, match); err != nil {
		tracer.Errorf(ctx, "Failed to update finished status for match %s: %s", match.Id, err)
	}

	// Update rankings for contest if contest is set
	if job.ContestId != "" {
		_, _ = e.svc.CalculateRankings(ctx, job.ContestId)
	}

	tracer.Infof(ctx, "Successfully completed match %s (winner: %s, duration: %d ticks)", job.MatchId, state.Winner, len(replayFrames))
	return nil
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
