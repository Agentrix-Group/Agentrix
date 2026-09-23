package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
	"github.com/google/uuid"
)

var (
	ErrMatchNotFound          = errors.New("match not found")
	ErrInvalidMatchState      = errors.New("invalid match state for execution")
	ErrInvalidSubmissions     = errors.New("match has invalid or inactive submissions")
	ErrActiveMatchCannotRerun = errors.New("match is currently active or queued")
)

func (s *service) ListMatches(ctx context.Context) ([]model.Match, error) {
	return s.repo.ListMatches(ctx)
}

func (s *service) ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error) {
	return s.repo.ListMatchesByContest(ctx, contestId)
}

func (s *service) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	match, err := s.repo.GetMatch(ctx, id)
	if err != nil {
		return nil, err
	}

	// Fetch match results if available
	results, err := s.repo.ListResultsByMatch(ctx, id)
	if err == nil {
		match.Results = results
	}

	return match, nil
}

// validateMatchParticipants exige entre StarfighterMinPlayers y
// StarfighterMaxPlayers submissions distintas y no vacías. Antes una partida
// con más de 5 fallaba recién en el motor y una con submissions repetidas,
// al sellar el replay.
func validateMatchParticipants(submissionIds []string) error {
	if len(submissionIds) < model.StarfighterMinPlayers || len(submissionIds) > model.StarfighterMaxPlayers {
		return fmt.Errorf("%w: got %d", ErrInvalidParticipants, len(submissionIds))
	}
	seen := make(map[string]bool, len(submissionIds))
	for _, id := range submissionIds {
		if id == "" {
			return fmt.Errorf("%w: empty submission id", ErrInvalidParticipants)
		}
		if seen[id] {
			return fmt.Errorf("%w: submission %s appears twice", ErrInvalidParticipants, id)
		}
		seen[id] = true
	}
	return nil
}

func (s *service) CreateMatch(ctx context.Context, match *model.Match, submissionIds []string) (*model.MatchResponse, error) {
	if match.GameId != "starfighter" {
		return nil, ErrUnsupportedGame
	}
	if err := validateMatchParticipants(submissionIds); err != nil {
		return nil, err
	}
	if match.Id == "" {
		match.Id = uuid.New().String()
	}
	if match.Status == "" {
		match.Status = common.MatchStatusPending
	}
	if match.Seed == 0 {
		match.Seed = time.Now().UnixNano()
	}
	match.Active = true
	match.CreatedAt = time.Now().UTC()

	// Validate contest state if contestId is provided
	if match.ContestId != "" {
		if reader, ok := s.repo.(repository.ContestReader); ok {
			contest, err := reader.GetContest(ctx, match.ContestId)
			if err == nil && contest != nil {
				if contest.State == model.ContestStateDraft ||
					contest.State == model.ContestStateFinished ||
					contest.State == model.ContestStateCompleted ||
					contest.State == model.ContestStateArchived ||
					contest.State == model.ContestStateCancelled {
					return nil, fmt.Errorf("contest %s is in %s state and cannot schedule matches", match.ContestId, contest.State)
				}
			}
		}
	}

	// Build match slots and validate active contest entries
	var slots []model.MatchSlot
	now := time.Now().UTC()
	for i, subId := range submissionIds {
		slot := model.MatchSlot{
			Id:           uuid.New().String(),
			MatchId:      match.Id,
			SlotIndex:    i,
			SubmissionId: subId,
			CreatedAt:    now,
		}

		if match.ContestId != "" {
			if reader, ok := s.repo.(repository.ContestReader); ok {
				entry, err := reader.GetContestEntryBySubmission(ctx, match.ContestId, subId)
				if err == nil && entry != nil {
					if entry.Status != model.ContestEntryStatusEnrolled && entry.Status != model.ContestEntryStatusActive {
						return nil, fmt.Errorf("contest entry for submission %s is not active (%s)", subId, entry.Status)
					}
					entryID := entry.Id
					slot.ContestEntryId = &entryID
					if entry.Agent != nil {
						slot.AgentName = entry.Agent.Name
					}
					if entry.User != nil {
						slot.Username = entry.User.Username
					}
				}
			}
		}
		slots = append(slots, slot)
	}
	match.Slots = slots

	if err := s.repo.CreateMatch(ctx, match); err != nil {
		return nil, err
	}

	return &model.MatchResponse{
		HttpStatusCode: http.StatusCreated,
		Message:        fmt.Sprintf("Match %s scheduled successfully", match.Id),
		MatchId:        match.Id,
		Match:          match,
		Slots:          slots,
	}, nil
}

func (s *service) RunMatch(ctx context.Context, matchId string, idempotencyKey ...string) (*model.RunMatchResponse, error) {
	idemKey := ""
	if len(idempotencyKey) > 0 {
		idemKey = idempotencyKey[0]
	}

	// 1. Check Idempotency Key if provided
	if idemKey != "" {
		if idemRepo, ok := s.repo.(repository.IdempotencyRepository); ok {
			rec, err := idemRepo.GetIdempotencyRecord(ctx, idemKey)
			if err == nil && rec != nil {
				var cachedResp model.RunMatchResponse
				if err := json.Unmarshal([]byte(rec.ResponseBody), &cachedResp); err == nil {
					cachedResp.HttpStatusCode = rec.StatusCode
					return &cachedResp, nil
				}
			}
		}
	}

	// 2. Load match
	match, err := s.repo.GetMatch(ctx, matchId)
	if err != nil {
		return nil, ErrMatchNotFound
	}
	if match.GameId != "starfighter" {
		return nil, ErrUnsupportedGame
	}

	// 3. Check match status
	if match.Status == common.MatchStatusFinished {
		return nil, fmt.Errorf("%w: match %s is finished and cannot be rerun", ErrInvalidMatchState, matchId)
	}

	// If match is already queued or running, return the active run/job idempotently
	if match.Status == common.MatchStatusQueued || match.Status == common.MatchStatusRunning {
		runId := ""
		jobId := ""
		if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
			if latestRun, err := runRepo.GetLatestMatchRunByMatch(ctx, matchId); err == nil && latestRun != nil {
				runId = latestRun.Id
			}
		}
		if s.queue != nil {
			if job, err := s.queue.GetJobByMatch(ctx, matchId); err == nil && job != nil {
				jobId = job.JobId
				if runId == "" {
					runId = job.RunId
				}
			}
		}
		return &model.RunMatchResponse{
			HttpStatusCode: http.StatusAccepted,
			MatchId:        match.Id,
			RunId:          runId,
			JobId:          jobId,
			Status:         string(match.Status),
		}, nil
	}

	// Valid states to initiate execution: scheduled, pending, or failed (rerun)
	if match.Status != "scheduled" && match.Status != "pending" && match.Status != common.MatchStatusFailed {
		return nil, fmt.Errorf("%w: cannot run match in status %q", ErrInvalidMatchState, match.Status)
	}

	// 4. Validate that all slots have valid and active submissions
	slots := match.Slots
	if len(slots) == 0 {
		if slotRepo, ok := s.repo.(repository.MatchSlotRepository); ok {
			slots, _ = slotRepo.ListMatchSlots(ctx, matchId)
		}
	}
	if len(slots) == 0 {
		return nil, fmt.Errorf("%w: match has no slots configured", ErrInvalidSubmissions)
	}

	subMap := make(map[string]*model.Submission)
	var submissionIds []string
	for _, slot := range slots {
		if slot.SubmissionId == "" {
			return nil, fmt.Errorf("%w: slot %d has no submission", ErrInvalidSubmissions, slot.SlotIndex)
		}
		sub, err := s.repo.GetSubmission(ctx, slot.SubmissionId)
		if err != nil || sub == nil {
			return nil, fmt.Errorf("%w: slot %d submission %s not found", ErrInvalidSubmissions, slot.SlotIndex, slot.SubmissionId)
		}
		if !sub.Active {
			return nil, fmt.Errorf("%w: slot %d submission %s is inactive", ErrInvalidSubmissions, slot.SlotIndex, slot.SubmissionId)
		}
		submissionIds = append(submissionIds, slot.SubmissionId)
		subMap[slot.SubmissionId] = sub
	}

	// 5. Determine attempt and run_id
	attempt := 1
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		runs, err := runRepo.ListMatchRunsByMatch(ctx, matchId)
		if err == nil && len(runs) > 0 {
			attempt = len(runs) + 1
		}
	}
	runId := fmt.Sprintf("%s-run-%d", match.Id, attempt)

	// Build execution slot specs and prepare ExecutionSpec BEFORE CAS
	var slotSpecs []model.ExecutionSlotSpec
	for _, slot := range slots {
		artifactDigest := ""
		if sub, ok := subMap[slot.SubmissionId]; ok && sub != nil {
			if sub.CodePath != "" {
				// Bot v2: digest del paquete completo (ADR-0014).
				if d, err := model.BotArtifactDigest(sub.CodePath); err == nil {
					artifactDigest = d
				}
			}
			if artifactDigest == "" {
				h := sha256.Sum256([]byte("agentrix.submission/" + sub.Id))
				artifactDigest = hex.EncodeToString(h[:])
			}
		} else {
			h := sha256.Sum256([]byte("agentrix.submission/" + slot.SubmissionId))
			artifactDigest = hex.EncodeToString(h[:])
		}
		slotSpecs = append(slotSpecs, model.ExecutionSlotSpec{
			SlotID:         slot.SubmissionId,
			ArtifactDigest: artifactDigest,
		})
	}

	maxTicks := 3600
	tickHz := 60.0
	gameVersion := model.StarfighterGameVersion
	engineVersion := model.StarfighterEngineVersion
	engineDigest := ""

	config := make(map[string]interface{})
	binPath := os.Getenv("AGENTRIX_ENGINE_BIN")
	if manifest := game.GetRegistry().GetManifest(match.GameId); manifest != nil {
		if manifest.MaxTicks > 0 {
			maxTicks = manifest.MaxTicks
		}
		if manifest.TickHz > 0 {
			tickHz = manifest.TickHz
		}
		if manifest.Version != "" {
			gameVersion = manifest.Version
		}
		for k, v := range manifest.Settings {
			if k == "tick_hz" {
				continue
			}
			if k == "asteroid_count" {
				if u, err := strconv.ParseUint(v, 10, 32); err == nil {
					config[k] = u
					continue
				}
			}
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				config[k] = f
				continue
			}
			config[k] = v
		}
		if _, ok := config["asteroid_count"]; !ok {
			config["asteroid_count"] = uint64(5)
		}
		if binPath == "" {
			binPath = manifest.BinaryPath
		}
	}
	if match.GameId == model.StarfighterGameID && len(config) == 0 {
		config = map[string]interface{}{
			"arena_width":                 2000.0,
			"arena_height":                1000.0,
			"ship_max_health":             100.0,
			"ship_max_energy":             100.0,
			"bullet_damage":               25.0,
			"asteroid_damage":             100.0,
			"shoot_energy_cost":           15.0,
			"shield_energy_cost_per_tick": 1.0,
			"shield_damage_reduction":     0.7,
			"energy_regen_per_tick":       0.5,
			"radar_range":                 800.0,
			"asteroid_count":              uint64(5),
		}
	}
	if binPath != "" {
		if _, err := os.Stat(binPath); err != nil {
			if _, err2 := os.Stat(filepath.Join("../../", binPath)); err2 == nil {
				binPath = filepath.Join("../../", binPath)
			}
		}
		if d, err := model.ComputeFileSHA256(binPath); err == nil {
			engineDigest = d
		}
	}
	if envDigest := os.Getenv("AGENTRIX_ENGINE_DIGEST"); envDigest != "" {
		engineDigest = envDigest
	}
	if engineDigest == "" {
		return nil, fmt.Errorf("%w: engine binary digest is required (fail closed)", model.ErrInvalidExecutionSpec)
	}

	tickRateNumerator := uint32(tickHz)
	if tickRateNumerator == 0 {
		tickRateNumerator = 60
	}
	tickRate, err := model.NewTickRate(tickRateNumerator, 1)
	if err != nil {
		return nil, fmt.Errorf("invalid tick rate: %w", err)
	}

	gameKey, err := model.StarfighterGameKey()
	if err != nil {
		return nil, fmt.Errorf("derive starfighter game key: %w", err)
	}
	if match.GameId != model.StarfighterGameID {
		gameKey = model.GameKey{
			GameID:      match.GameId,
			GameVersion: gameVersion,
			GameDigest:  strings.Repeat("0", 64),
		}
	}

	schemaDigests := model.SchemaDigests{
		Action:      model.StarfighterActionSchemaDigest,
		Observation: model.StarfighterObservationSchemaDigest,
		Public:      model.StarfighterPublicSchemaDigest,
		Replay:      model.StarfighterReplaySchemaDigest,
	}

	limits := model.ExecutionLimits{
		MaxTicks:        uint64(maxTicks),
		MaxPlayers:      uint32(len(slotSpecs)),
		MaxEntities:     10000,
		MaxMessageBytes: 1048576,
	}

	execSpec := &model.ExecutionSpec{
		ProtocolVersion:      model.ExecutionSpecProtocolVersion,
		RunID:                runId,
		MatchID:              match.Id,
		EngineVersion:        engineVersion,
		EngineDigest:         engineDigest,
		BuildIdentity:        fmt.Sprintf("agentrix-engine-%s", engineVersion),
		Target:               "x86_64-unknown-linux-gnu",
		Game:                 gameKey,
		SchemaDigests:        schemaDigests,
		Config:               config,
		TickRate:             tickRate,
		Seed:                 uint64(match.Seed),
		Slots:                slotSpecs,
		Limits:               limits,
		FailurePolicyVersion: model.DefaultFailurePolicyVersion,
		DeterminismTier:      model.TierSameArtifactSameTarget,
		RNGAlgorithm:         model.DefaultRNGAlgorithm,
	}
	if err := execSpec.Seal(); err != nil {
		return nil, fmt.Errorf("failed to seal execution spec: %w", err)
	}
	specJSON, err := execSpec.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize execution spec: %w", err)
	}

	// 6. Atomically transition match to queued using CAS
	var updated bool
	if casRepo, ok := s.repo.(repository.MatchCASRepository); ok {
		updated, err = casRepo.UpdateMatchStatusCAS(ctx, matchId, model.MatchStatus(match.Status), model.MatchStatusQueued)
		if err != nil {
			return nil, fmt.Errorf("failed to transition match status to queued: %w", err)
		}
	} else {
		match.Status = common.MatchStatusQueued
		if err := s.repo.UpdateMatch(ctx, match); err != nil {
			return nil, fmt.Errorf("failed to transition match status to queued: %w", err)
		}
		updated = true
	}

	if !updated {
		// Concurrent request updated the match; return active run/job
		freshMatch, err := s.repo.GetMatch(ctx, matchId)
		if err == nil && (freshMatch.Status == common.MatchStatusQueued || freshMatch.Status == common.MatchStatusRunning) {
			existingRunId := ""
			existingJobId := ""
			for retry := 0; retry < 15 && (existingRunId == "" || existingJobId == ""); retry++ {
				if retry > 0 {
					time.Sleep(5 * time.Millisecond)
				}
				if runRepo, ok := s.repo.(repository.MatchRunRepository); ok && existingRunId == "" {
					if latestRun, err := runRepo.GetLatestMatchRunByMatch(ctx, matchId); err == nil && latestRun != nil {
						existingRunId = latestRun.Id
					}
				}
				if s.queue != nil && existingJobId == "" {
					if job, err := s.queue.GetJobByMatch(ctx, matchId); err == nil && job != nil {
						existingJobId = job.JobId
						if existingRunId == "" {
							existingRunId = job.RunId
						}
					}
				}
			}
			return &model.RunMatchResponse{
				HttpStatusCode: http.StatusAccepted,
				MatchId:        match.Id,
				RunId:          existingRunId,
				JobId:          existingJobId,
				Status:         string(freshMatch.Status),
			}, nil
		}
		return nil, fmt.Errorf("%w: match status changed concurrently", ErrInvalidMatchState)
	}

	// 7. Create MatchRun in 'created' status with ExecutionSpec
	run := &model.MatchRun{
		Id:            runId,
		MatchId:       match.Id,
		Attempt:       attempt,
		WorkerId:      "unassigned",
		FencingToken:  0,
		Status:        model.MatchRunStatusCreated,
		ExecutionSpec: specJSON,
		StartedAt:     time.Now().UTC(),
	}
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		if err := runRepo.CreateMatchRun(ctx, run); err != nil {
			if casRepo, ok := s.repo.(repository.MatchCASRepository); ok {
				_, _ = casRepo.UpdateMatchStatusCAS(ctx, matchId, model.MatchStatusQueued, model.MatchStatus(match.Status))
			}
			return nil, fmt.Errorf("failed to create match run: %w", err)
		}
	}

	// 8. Enqueue MatchJob
	jobId := uuid.New().String()
	if s.queue != nil {
		job := &connection.MatchJob{
			JobId:         jobId,
			Attempt:       0,
			MatchId:       match.Id,
			RunId:         runId,
			ContestId:     match.ContestId,
			GameId:        match.GameId,
			GameVersion:   gameVersion,
			EngineDigest:  engineDigest,
			ConfigHash:    execSpec.ConfigDigest,
			SubmissionIds: submissionIds,
			Seed:          match.Seed,
		}
		if err := s.queue.Enqueue(ctx, job); err != nil {
			tracer.WarnEvent(ctx, tracer.ScopeQueue, "match.enqueue.degraded", "Partida en cola degradada", tracer.Err(err))
			return nil, fmt.Errorf("failed to enqueue match job: %w", err)
		}
		tracer.InfoEvent(ctx, tracer.ScopeQueue, "match.queued", "Partida en cola",
			tracer.String("job_id", job.JobId), tracer.String("match_id", job.MatchId), tracer.String("run_id", runId))
	}

	resp := &model.RunMatchResponse{
		HttpStatusCode: http.StatusAccepted,
		MatchId:        match.Id,
		RunId:          runId,
		JobId:          jobId,
		Status:         "queued",
	}

	// 9. Persist Idempotency Record if key was provided
	if idemKey != "" {
		if idemRepo, ok := s.repo.(repository.IdempotencyRepository); ok {
			bodyBytes, _ := json.Marshal(resp)
			_ = idemRepo.SaveIdempotencyRecord(ctx, &model.IdempotencyRecord{
				Key:          idemKey,
				ResourceType: "match_run",
				ResourceId:   runId,
				ResponseBody: string(bodyBytes),
				StatusCode:   http.StatusAccepted,
			})
		}
	}

	return resp, nil
}

func (s *service) UpdateMatch(ctx context.Context, match *model.Match) error {
	if match.GameId != "starfighter" {
		return ErrUnsupportedGame
	}
	return s.repo.UpdateMatch(ctx, match)
}

func (s *service) ActivateMatch(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateMatch(ctx, id, isActive)
}

func (s *service) CommitMatchResult(ctx context.Context, commit model.MatchResultCommit) error {
	if committer, ok := s.repo.(repository.MatchCommitter); ok {
		return committer.CommitMatchResult(ctx, commit)
	}

	// Fallback for mock repositories without transactional commit
	match, err := s.repo.GetMatch(ctx, commit.MatchID)
	if err != nil {
		return err
	}
	match.Status = commit.Status
	match.ReplayId = commit.ReplayID
	match.FinishedAt = &commit.FinishedAt
	if err := s.repo.UpdateMatch(ctx, match); err != nil {
		return err
	}
	for _, res := range commit.Results {
		rCopy := res
		_ = s.repo.CreateResult(ctx, &rCopy)
	}
	return nil
}

func (s *service) CreateMatchRun(ctx context.Context, run *model.MatchRun) error {
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		return runRepo.CreateMatchRun(ctx, run)
	}
	return nil
}

func (s *service) GetMatchRun(ctx context.Context, id string) (*model.MatchRun, error) {
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		return runRepo.GetMatchRun(ctx, id)
	}
	return nil, nil
}

func (s *service) UpdateMatchRunStatusCAS(ctx context.Context, runId string, expectedStatus, newStatus model.MatchRunStatus) (bool, error) {
	if err := expectedStatus.ValidateTransition(newStatus); err != nil {
		return false, err
	}
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		return runRepo.UpdateMatchRunStatusCAS(ctx, runId, expectedStatus, newStatus)
	}
	return false, errors.New("match run repository does not support CAS")
}

func (s *service) GetLatestMatchRunByMatch(ctx context.Context, matchId string) (*model.MatchRun, error) {
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		return runRepo.GetLatestMatchRunByMatch(ctx, matchId)
	}
	return nil, nil
}

func (s *service) ListMatchRunsByMatch(ctx context.Context, matchId string) ([]*model.MatchRun, error) {
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		return runRepo.ListMatchRunsByMatch(ctx, matchId)
	}
	return nil, nil
}

func (s *service) StartMatchRun(ctx context.Context, runId, workerId string, fencingToken int64) error {
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		return runRepo.StartMatchRun(ctx, runId, workerId, fencingToken)
	}
	return nil
}

func (s *service) FailMatchRun(ctx context.Context, runId string, lastError string) error {
	if runRepo, ok := s.repo.(repository.MatchRunRepository); ok {
		return runRepo.FailMatchRun(ctx, runId, lastError)
	}
	return nil
}
