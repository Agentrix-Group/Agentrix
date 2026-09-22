package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/google/uuid"
)

func (s *service) ListRankings(ctx context.Context) ([]model.Ranking, error) {
	return s.repo.ListRankings(ctx)
}

func (s *service) ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error) {
	return s.repo.ListRankingsByContest(ctx, contestId)
}

func (s *service) GetRanking(ctx context.Context, id string) (*model.Ranking, error) {
	return s.repo.GetRanking(ctx, id)
}

// CalculateRankings delegates directly to deterministic recalculation.
func (s *service) CalculateRankings(ctx context.Context, contestId string) ([]model.Ranking, error) {
	return s.RecalculateContestRankings(ctx, contestId)
}

type agentStats struct {
	AgentId           string
	UserId            string
	Points            int
	Score             int
	MatchesPlayed     int
	Wins              int
	Losses            int
	Draws             int
	Disqualifications int
	ScoreDiff         int
	H2HPoints         map[string]int
}

func normalizeResultOutcome(res model.Result, allMatchResults []model.Result) string {
	switch res.Status {
	case "disqualified", "dq":
		return "disqualified"
	case "no_contest", "cancelled", "aborted":
		return "no_contest"
	case "win", "victory":
		return "win"
	case "loss", "defeat":
		return "loss"
	case "draw", "tie":
		return "draw"
	}

	minRank := 999999
	for _, r := range allMatchResults {
		if r.Status == "disqualified" || r.Status == "no_contest" {
			continue
		}
		if r.Rank < minRank {
			minRank = r.Rank
		}
	}

	rank1Count := 0
	for _, r := range allMatchResults {
		if r.Status == "disqualified" || r.Status == "no_contest" {
			continue
		}
		if r.Rank == minRank {
			rank1Count++
		}
	}

	if res.Rank == minRank {
		if rank1Count > 1 {
			return "draw"
		}
		return "win"
	}
	return "loss"
}

// matchPoints devuelve los puntos de torneo de un participante en una
// partida según el modo de la política. En modo por posición un
// descalificado recibe los puntos de su puesto menos la penalización, y una
// partida sin resultado no reparte puntos.
func matchPoints(policy model.ScoringPolicy, outcome string, rank, players, tied int) int {
	if policy.EffectiveMode() != model.ScoringModePlacement {
		return policy.PointsForResult(outcome)
	}
	switch outcome {
	case "no_contest":
		return 0
	case "disqualified":
		return model.PlacementPoints(players, rank, tied) - policy.DisqualificationPenalty
	default:
		return model.PlacementPoints(players, rank, tied)
	}
}

func (s *service) RecalculateContestRankings(ctx context.Context, contestId string) ([]model.Ranking, error) {
	contest, err := s.repo.GetContest(ctx, contestId)
	if err != nil {
		return nil, fmt.Errorf("get contest %s: %w", contestId, err)
	}
	if contest == nil {
		return nil, ErrContestNotFound
	}

	policy := model.DefaultScoringPolicy()
	if contest.ScoringPolicy != nil {
		policy = *contest.ScoringPolicy
	}

	statsMap := make(map[string]*agentStats)

	// Pre-populate with all enrolled contest entries
	entries, err := s.repo.ListContestEntries(ctx, contestId)
	if err == nil {
		for _, entry := range entries {
			statsMap[entry.AgentId] = &agentStats{
				AgentId:   entry.AgentId,
				UserId:    entry.UserId,
				H2HPoints: make(map[string]int),
			}
		}
	}

	matches, err := s.repo.ListMatchesByContest(ctx, contestId)
	if err != nil {
		return nil, fmt.Errorf("list contest matches: %w", err)
	}

	// Sort matches chronologically for determinism
	sort.Slice(matches, func(i, j int) bool {
		var tI, tJ time.Time
		if matches[i].FinishedAt != nil {
			tI = *matches[i].FinishedAt
		} else {
			tI = matches[i].CreatedAt
		}
		if matches[j].FinishedAt != nil {
			tJ = *matches[j].FinishedAt
		} else {
			tJ = matches[j].CreatedAt
		}
		if tI.Equal(tJ) {
			return matches[i].Id < matches[j].Id
		}
		return tI.Before(tJ)
	})

	_ = s.repo.ClearAppliedRuns(ctx, contestId)

	type matchParticipant struct {
		agentId string
		userId  string
		res     model.Result
		outcome string
	}

	for _, m := range matches {
		isCommitted := false
		runId := ""
		if m.CommittedRunId != nil && *m.CommittedRunId != "" {
			isCommitted = true
			runId = *m.CommittedRunId
		} else if m.Status == common.MatchStatusFinished {
			isCommitted = true
			runId = m.RunId
			if runId == "" {
				runId = m.Id + "-run"
			}
		}

		if !isCommitted {
			continue
		}

		results, err := s.repo.ListResultsByMatch(ctx, m.Id)
		if err != nil {
			return nil, fmt.Errorf("list results for match %s: %w", m.Id, err)
		}
		if len(results) == 0 {
			continue
		}

		// Puestos de competencia compartidos en esta partida (1, 2, 2, 4),
		// para repartir puntos por posición entre empatados.
		tiedAtRank := make(map[int]int, len(results))
		for _, res := range results {
			tiedAtRank[res.Rank]++
		}

		var participants []matchParticipant
		for _, res := range results {
			sub, err := s.repo.GetSubmission(ctx, res.SubmissionId)
			if err != nil {
				return nil, fmt.Errorf("get submission %s for match %s: %w", res.SubmissionId, m.Id, err)
			}
			agent, err := s.repo.GetAgent(ctx, sub.AgentId)
			if err != nil {
				return nil, fmt.Errorf("get agent %s for submission %s: %w", sub.AgentId, res.SubmissionId, err)
			}
			outcome := normalizeResultOutcome(res, results)
			participants = append(participants, matchParticipant{
				agentId: agent.Id,
				userId:  agent.OwnerUserId,
				res:     res,
				outcome: outcome,
			})
		}

		for i, p := range participants {
			st, exists := statsMap[p.agentId]
			if !exists {
				st = &agentStats{
					AgentId:   p.agentId,
					UserId:    p.userId,
					H2HPoints: make(map[string]int),
				}
				statsMap[p.agentId] = st
			}

			st.MatchesPlayed++
			st.Score += p.res.Score
			st.Points += matchPoints(policy, p.outcome, p.res.Rank, len(results), tiedAtRank[p.res.Rank])

			switch p.outcome {
			case "win":
				st.Wins++
			case "loss":
				st.Losses++
			case "draw":
				st.Draws++
			case "disqualified":
				st.Disqualifications++
			}

			for j, opp := range participants {
				if i == j {
					continue
				}
				st.ScoreDiff += (p.res.Score - opp.res.Score)
				// Cara a cara por puesto final (menor es mejor): el score
				// ya no indica quién le ganó a quién (son bajas, ADR-0013).
				if p.res.Rank < opp.res.Rank {
					st.H2HPoints[opp.agentId] += policy.WinPoints
				} else if p.res.Rank == opp.res.Rank {
					st.H2HPoints[opp.agentId] += policy.DrawPoints
				} else {
					st.H2HPoints[opp.agentId] += policy.LossPoints
				}
			}
		}

		_ = s.repo.RecordAppliedRun(ctx, contestId, runId, m.Id)
	}

	var sortedList []*agentStats
	for _, st := range statsMap {
		sortedList = append(sortedList, st)
	}

	sort.Slice(sortedList, func(i, j int) bool {
		a := sortedList[i]
		b := sortedList[j]

		if a.Points != b.Points {
			return a.Points > b.Points
		}

		for _, rule := range policy.Tiebreakers {
			switch rule {
			case model.TiebreakerScoreDiff:
				if a.ScoreDiff != b.ScoreDiff {
					return a.ScoreDiff > b.ScoreDiff
				}
			case model.TiebreakerHeadToHead:
				h2hA := a.H2HPoints[b.AgentId]
				h2hB := b.H2HPoints[a.AgentId]
				if h2hA != h2hB {
					return h2hA > h2hB
				}
			case model.TiebreakerWins:
				if a.Wins != b.Wins {
					return a.Wins > b.Wins
				}
			case model.TiebreakerRivalSurvival:
				if a.Score != b.Score {
					return a.Score > b.Score
				}
			case model.TiebreakerKills:
				if a.Score != b.Score {
					return a.Score > b.Score
				}
			}
		}

		return a.AgentId < b.AgentId
	})

	var rankings []model.Ranking
	now := time.Now().UTC()
	for idx, st := range sortedList {
		rk := model.Ranking{
			Id:                uuid.New().String(),
			ContestId:         contestId,
			AgentId:           st.AgentId,
			UserId:            st.UserId,
			Score:             st.Score,
			Points:            st.Points,
			MatchesPlayed:     st.MatchesPlayed,
			Wins:              st.Wins,
			Losses:            st.Losses,
			Draws:             st.Draws,
			Disqualifications: st.Disqualifications,
			TiebreakerScore:   float64(st.ScoreDiff),
			Rank:              idx + 1,
			UpdatedAt:         now,
		}

		if existing, err := s.repo.GetRankingByContestAndAgent(ctx, contestId, st.AgentId); err == nil && existing != nil {
			rk.Id = existing.Id
		}

		if err := s.repo.UpsertRanking(ctx, &rk); err != nil {
			return nil, fmt.Errorf("upsert ranking for agent %s: %w", st.AgentId, err)
		}
		rankings = append(rankings, rk)
	}

	return rankings, nil
}

func (s *service) ApplyMatchResultIncremental(ctx context.Context, matchId string) error {
	match, err := s.repo.GetMatch(ctx, matchId)
	if err != nil {
		return fmt.Errorf("get match %s: %w", matchId, err)
	}
	if match.ContestId == "" {
		return nil
	}

	runId := ""
	if match.CommittedRunId != nil && *match.CommittedRunId != "" {
		runId = *match.CommittedRunId
	} else if match.RunId != "" {
		runId = match.RunId
	} else {
		runId = match.Id + "-run"
	}

	applied, err := s.repo.IsRunAppliedToRanking(ctx, match.ContestId, runId)
	if err != nil {
		return fmt.Errorf("check applied run: %w", err)
	}
	if applied {
		return nil
	}

	_, err = s.RecalculateContestRankings(ctx, match.ContestId)
	return err
}

func (s *service) PublishRankingSnapshot(ctx context.Context, contestId, publisherUserId string) (*model.RankingSnapshot, error) {
	contest, err := s.repo.GetContest(ctx, contestId)
	if err != nil {
		return nil, fmt.Errorf("get contest %s: %w", contestId, err)
	}
	if contest == nil {
		return nil, ErrContestNotFound
	}

	rankings, err := s.repo.ListRankingsByContest(ctx, contestId)
	if err != nil {
		return nil, fmt.Errorf("list rankings for snapshot: %w", err)
	}

	latest, err := s.repo.GetLatestRankingSnapshot(ctx, contestId)
	if err != nil {
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}

	version := 1
	if latest != nil {
		version = latest.Version + 1
	}

	var pubBy *string
	if publisherUserId != "" {
		pubBy = &publisherUserId
	}

	snapshot := &model.RankingSnapshot{
		Id:          uuid.New().String(),
		ContestId:   contestId,
		Version:     version,
		Rankings:    rankings,
		PublishedBy: pubBy,
		PublishedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateRankingSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("create ranking snapshot: %w", err)
	}

	return snapshot, nil
}

func (s *service) ListRankingSnapshots(ctx context.Context, contestId string) ([]model.RankingSnapshot, error) {
	return s.repo.ListRankingSnapshots(ctx, contestId)
}

func (s *service) GetPublishedRankings(ctx context.Context, contestId string, version ...int) (*model.RankingSnapshot, error) {
	if len(version) > 0 && version[0] > 0 {
		return s.repo.GetRankingSnapshotByVersion(ctx, contestId, version[0])
	}
	return s.repo.GetLatestRankingSnapshot(ctx, contestId)
}
