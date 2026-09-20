package service

import (
	"context"
	"sort"
	"time"

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

func (s *service) CalculateRankings(ctx context.Context, contestId string) ([]model.Ranking, error) {
	matches, err := s.repo.ListMatchesByContest(ctx, contestId)
	if err != nil {
		return nil, err
	}

	type agentStats struct {
		AgentId       string
		UserId        string
		Score         int
		MatchesPlayed int
		Wins          int
		Losses        int
		Draws         int
	}

	statsMap := make(map[string]*agentStats)

	for _, m := range matches {
		results, err := s.repo.ListResultsByMatch(ctx, m.Id)
		if err != nil || len(results) == 0 {
			continue
		}

		for _, res := range results {
			sub, err := s.repo.GetSubmission(ctx, res.SubmissionId)
			if err != nil {
				continue
			}

			agent, err := s.repo.GetAgent(ctx, sub.AgentId)
			if err != nil {
				continue
			}

			st, exists := statsMap[agent.Id]
			if !exists {
				st = &agentStats{
					AgentId: agent.Id,
					UserId:  agent.OwnerUserId,
				}
				statsMap[agent.Id] = st
			}

			st.MatchesPlayed++
			st.Score += res.Score
			if res.Rank == 1 {
				st.Wins++
			} else if res.Rank == 2 && len(results) > 2 {
				st.Draws++
			} else {
				st.Losses++
			}
		}
	}

	// Sort agents by Score descending
	var sortedList []*agentStats
	for _, st := range statsMap {
		sortedList = append(sortedList, st)
	}

	sort.Slice(sortedList, func(i, j int) bool {
		if sortedList[i].Score == sortedList[j].Score {
			return sortedList[i].Wins > sortedList[j].Wins
		}
		return sortedList[i].Score > sortedList[j].Score
	})

	var rankings []model.Ranking
	now := time.Now().UTC()
	for i, st := range sortedList {
		ranking := model.Ranking{
			Id:            uuid.New().String(),
			ContestId:     contestId,
			AgentId:       st.AgentId,
			UserId:        st.UserId,
			Score:         st.Score,
			MatchesPlayed: st.MatchesPlayed,
			Wins:          st.Wins,
			Losses:        st.Losses,
			Draws:         st.Draws,
			Rank:          i + 1,
			UpdatedAt:     now,
		}

		if err := s.repo.UpsertRanking(ctx, &ranking); err != nil {
			continue
		}
		rankings = append(rankings, ranking)
	}

	return rankings, nil
}
