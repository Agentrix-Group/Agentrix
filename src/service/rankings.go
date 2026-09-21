package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

// recomputeLocked rebuilds the ranking projection of a contest from the
// committed runs only. It must run inside a transaction that holds the
// contest ranking lock.
func (s *Service) recomputeLocked(ctx context.Context, q *repository.Queries, contest *model.Contest) ([]model.Ranking, model.RankingState, error) {
	state, err := q.GetRankingState(ctx, contest.ID)
	if err != nil {
		return nil, state, err
	}
	entries, err := q.RankingEntries(ctx, contest.ID)
	if err != nil {
		return nil, state, err
	}
	results, err := q.CommittedResults(ctx, contest.ID)
	if err != nil {
		return nil, state, err
	}
	rankings := model.ComputeRankings(contest.ID, entries, results, contest.ScoringPolicy)
	applied := map[string]string{}
	for _, r := range results {
		applied[r.RunID] = r.MatchID
	}
	lines := make([]string, 0, len(applied))
	for run, match := range applied {
		lines = append(lines, match+":"+run)
	}
	sort.Strings(lines)
	digest := model.SHA256Hex([]byte(strings.Join(lines, "\n")))
	now := s.now()
	if err := q.ReplaceRankings(ctx, contest.ID, rankings, applied, digest, state.DirtyVersion, now); err != nil {
		return nil, state, err
	}
	state.ComputedVersion = state.DirtyVersion
	state.AppliedRunsCount = len(applied)
	state.AppliedRunsDigest = digest
	state.ComputedAt = &now
	return rankings, state, nil
}

// RecomputeRankings refreshes the projection of one contest (worker
// reconciler and explicit recalculation).
func (s *Service) RecomputeRankings(ctx context.Context, contestID string) error {
	return s.store.Tx(ctx, func(q *repository.Queries) error {
		if err := q.LockContestRankings(ctx, contestID); err != nil {
			return err
		}
		contest, err := q.GetContest(ctx, contestID)
		if err != nil {
			return err
		}
		_, _, err = s.recomputeLocked(ctx, q, contest)
		return err
	})
}

func (s *Service) RecalculateRankings(ctx context.Context, p model.Principal, contestID string) error {
	if err := requireCap(p, model.CapRankingsPublish); err != nil {
		return err
	}
	if _, err := s.visibleContest(ctx, p, contestID); err != nil {
		return err
	}
	return s.RecomputeRankings(ctx, contestID)
}

// RecomputeDirtyRankings refreshes contests whose projection is behind.
func (s *Service) RecomputeDirtyRankings(ctx context.Context, limit int) (int, error) {
	ids, err := s.store.DirtyRankingContests(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := s.RecomputeRankings(ctx, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

type RankingView struct {
	Rankings []model.Ranking
	State    model.RankingState
}

func (s *Service) GetRankings(ctx context.Context, p model.Principal, contestID string) (*RankingView, error) {
	if err := requireCap(p, model.CapRankingsView); err != nil {
		return nil, err
	}
	if _, err := s.visibleContest(ctx, p, contestID); err != nil {
		return nil, err
	}
	rankings, err := s.store.ListRankings(ctx, contestID)
	if err != nil {
		return nil, err
	}
	state, err := s.store.GetRankingState(ctx, contestID)
	if err != nil {
		return nil, err
	}
	return &RankingView{Rankings: rankings, State: state}, nil
}

// PublishSnapshot freezes the current standings as a new immutable version.
// The projection is recomputed inside the same transaction, so the snapshot
// always covers every committed run.
func (s *Service) PublishSnapshot(ctx context.Context, p model.Principal, contestID string) (*model.RankingSnapshot, error) {
	if err := requireCap(p, model.CapRankingsPublish); err != nil {
		return nil, err
	}
	var snapshot *model.RankingSnapshot
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		contest, err := q.LockContest(ctx, contestID)
		if err != nil {
			return err
		}
		if contest.State != model.ContestRunning && contest.State != model.ContestFinished {
			return model.Conflict("snapshot_not_allowed", "rankings can be published while running or finished (state %s)", contest.State)
		}
		if err := q.LockContestRankings(ctx, contestID); err != nil {
			return err
		}
		rankings, state, err := s.recomputeLocked(ctx, q, contest)
		if err != nil {
			return err
		}
		version, err := q.NextSnapshotVersion(ctx, contestID)
		if err != nil {
			return err
		}
		rows := make([]model.SnapshotRow, 0, len(rankings))
		for _, r := range rankings {
			rows = append(rows, r.SnapshotRow())
		}
		raw, err := model.CanonicalJSON(rows)
		if err != nil {
			return err
		}
		now := s.now()
		snapshot = &model.RankingSnapshot{ID: s.newID(), ContestID: contestID, Version: version, Rankings: rankings,
			RankingsSHA256: model.SHA256Hex(raw), AppliedRunsDigest: state.AppliedRunsDigest, AppliedRunsCount: state.AppliedRunsCount,
			PublishedBy: p.UserID, PublishedAt: now}
		if err := q.InsertSnapshot(ctx, snapshot, raw); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "rankings.snapshot_published", "contest", contestID,
			map[string]any{"version": version, "sha256": snapshot.RankingsSHA256}, now)
	})
	return snapshot, err
}

func (s *Service) ListSnapshots(ctx context.Context, p model.Principal, contestID string) ([]model.RankingSnapshot, error) {
	if err := requireCap(p, model.CapRankingsView); err != nil {
		return nil, err
	}
	if _, err := s.visibleContest(ctx, p, contestID); err != nil {
		return nil, err
	}
	return s.store.ListSnapshots(ctx, contestID)
}

func (s *Service) GetSnapshot(ctx context.Context, p model.Principal, contestID string, version int) (*model.RankingSnapshot, error) {
	if err := requireCap(p, model.CapRankingsView); err != nil {
		return nil, err
	}
	if _, err := s.visibleContest(ctx, p, contestID); err != nil {
		return nil, err
	}
	return s.store.GetSnapshot(ctx, contestID, version)
}
