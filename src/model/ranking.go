package model

import (
	"sort"
	"time"
)

// RankingEntry is an input row of the ranking projection.
type RankingEntry struct {
	EntryID   string
	AgentID   string
	AgentName string
	UserID    string
	Username  string
	Status    EntryStatus
	CreatedAt time.Time
}

// CommittedMatchResult is one slot result of the committed run of an
// official (competitive, finished) match.
type CommittedMatchResult struct {
	MatchID string
	RunID   string
	EntryID string
	Score   int
	Outcome Outcome
}

type rankingAccumulator struct {
	Ranking
	entry   RankingEntry
	matches map[string]bool
}

// ComputeRankings is the deterministic ranking projection. Its output only
// depends on its inputs: the same entries, results and policy always yield
// the same rows in the same order.
func ComputeRankings(contestID string, entries []RankingEntry, results []CommittedMatchResult, policy ScoringPolicy) []Ranking {
	acc := make(map[string]*rankingAccumulator, len(entries))
	for _, e := range entries {
		acc[e.EntryID] = &rankingAccumulator{
			Ranking: Ranking{ContestID: contestID, EntryID: e.EntryID, AgentID: e.AgentID, AgentName: e.AgentName,
				UserID: e.UserID, Username: e.Username},
			entry:   e,
			matches: map[string]bool{},
		}
	}

	byMatch := map[string][]CommittedMatchResult{}
	for _, r := range results {
		byMatch[r.MatchID] = append(byMatch[r.MatchID], r)
	}
	// headToHead[a][b] = points a earned in matches that b also played.
	headToHead := map[string]map[string]int{}
	for _, matchResults := range byMatch {
		matchTotal := 0
		for _, r := range matchResults {
			matchTotal += r.Score
		}
		for _, r := range matchResults {
			row := acc[r.EntryID]
			if row == nil {
				continue
			}
			points := policy.Points(r.Outcome)
			row.Points += points
			row.MatchesPlayed++
			row.ScoreFor += r.Score
			row.ScoreAgainst += matchTotal - r.Score
			switch r.Outcome {
			case OutcomeWin:
				row.Wins++
			case OutcomeDraw:
				row.Draws++
			case OutcomeLoss:
				row.Losses++
			case OutcomeDisqualified:
				row.Disqualifications++
			}
			for _, other := range matchResults {
				if other.EntryID == r.EntryID {
					continue
				}
				if headToHead[r.EntryID] == nil {
					headToHead[r.EntryID] = map[string]int{}
				}
				headToHead[r.EntryID][other.EntryID] += points
			}
		}
	}

	rows := make([]*rankingAccumulator, 0, len(acc))
	for _, row := range acc {
		if row.entry.Status == EntryWithdrawn && row.MatchesPlayed == 0 {
			continue
		}
		rows = append(rows, row)
	}

	// Fallback order first; every later stable sort refines it.
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].entry.CreatedAt.Equal(rows[j].entry.CreatedAt) {
			return rows[i].entry.CreatedAt.Before(rows[j].entry.CreatedAt)
		}
		return rows[i].EntryID < rows[j].EntryID
	})

	groups := [][]*rankingAccumulator{rows}
	keys := []func(group []*rankingAccumulator) func(*rankingAccumulator) int{
		func([]*rankingAccumulator) func(*rankingAccumulator) int {
			return func(r *rankingAccumulator) int {
				if r.entry.Status == EntryDisqualified {
					return 0
				}
				return 1
			}
		},
		func([]*rankingAccumulator) func(*rankingAccumulator) int {
			return func(r *rankingAccumulator) int { return r.Points }
		},
	}
	for _, tb := range policy.Tiebreakers {
		switch tb {
		case TiebreakScoreDiff:
			keys = append(keys, func([]*rankingAccumulator) func(*rankingAccumulator) int {
				return func(r *rankingAccumulator) int { return r.ScoreFor - r.ScoreAgainst }
			})
		case TiebreakWins:
			keys = append(keys, func([]*rankingAccumulator) func(*rankingAccumulator) int {
				return func(r *rankingAccumulator) int { return r.Wins }
			})
		case TiebreakScoreFor:
			keys = append(keys, func([]*rankingAccumulator) func(*rankingAccumulator) int {
				return func(r *rankingAccumulator) int { return r.ScoreFor }
			})
		case TiebreakHeadToHead:
			keys = append(keys, func(group []*rankingAccumulator) func(*rankingAccumulator) int {
				return func(r *rankingAccumulator) int {
					total := 0
					for _, other := range group {
						total += headToHead[r.EntryID][other.EntryID]
					}
					return total
				}
			})
		}
	}

	for _, key := range keys {
		var next [][]*rankingAccumulator
		for _, group := range groups {
			if len(group) < 2 {
				next = append(next, group)
				continue
			}
			value := key(group)
			values := make(map[string]int, len(group))
			for _, r := range group {
				values[r.EntryID] = value(r)
			}
			sort.SliceStable(group, func(i, j int) bool { return values[group[i].EntryID] > values[group[j].EntryID] })
			start := 0
			for i := 1; i <= len(group); i++ {
				if i == len(group) || values[group[i].EntryID] != values[group[start].EntryID] {
					next = append(next, group[start:i])
					start = i
				}
			}
		}
		groups = next
	}

	out := make([]Ranking, 0, len(rows))
	position := 1
	for _, group := range groups {
		for _, r := range group {
			r.Rank = position
			out = append(out, r.Ranking)
		}
		position += len(group)
	}
	return out
}
