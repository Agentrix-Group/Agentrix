package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

type Tiebreaker string

const (
	TiebreakScoreDiff  Tiebreaker = "score_diff"
	TiebreakWins       Tiebreaker = "wins"
	TiebreakHeadToHead Tiebreaker = "head_to_head"
	TiebreakScoreFor   Tiebreaker = "score_for"
)

var Tiebreakers = []Tiebreaker{TiebreakScoreDiff, TiebreakWins, TiebreakHeadToHead, TiebreakScoreFor}

// ScoringPolicy converts committed results into ranking points. After all
// declared tiebreakers the order falls back to enrollment order (entry
// creation time, then entry ID), which keeps the projection deterministic.
type ScoringPolicy struct {
	WinPoints               int          `json:"win_points"`
	DrawPoints              int          `json:"draw_points"`
	LossPoints              int          `json:"loss_points"`
	DisqualificationPenalty int          `json:"disqualification_penalty"`
	Tiebreakers             []Tiebreaker `json:"tiebreakers"`
}

func DefaultScoringPolicy() ScoringPolicy {
	return ScoringPolicy{
		WinPoints: 3, DrawPoints: 1, LossPoints: 0, DisqualificationPenalty: 0,
		Tiebreakers: []Tiebreaker{TiebreakScoreDiff, TiebreakHeadToHead, TiebreakWins},
	}
}

func (p ScoringPolicy) Validate() error {
	if p.WinPoints < p.DrawPoints || p.DrawPoints < p.LossPoints {
		return Validation("invalid_scoring_policy", "points must satisfy win >= draw >= loss")
	}
	if p.WinPoints > 1000 || p.LossPoints < -1000 {
		return Validation("invalid_scoring_policy", "points must be within [-1000, 1000]")
	}
	if p.DisqualificationPenalty < 0 || p.DisqualificationPenalty > 1000 {
		return Validation("invalid_scoring_policy", "disqualification_penalty must be within [0, 1000]")
	}
	seen := map[Tiebreaker]bool{}
	for _, tb := range p.Tiebreakers {
		if !contains(Tiebreakers, tb) {
			return Validation("invalid_scoring_policy", "unknown tiebreaker %q", tb)
		}
		if seen[tb] {
			return Validation("invalid_scoring_policy", "duplicated tiebreaker %q", tb)
		}
		seen[tb] = true
	}
	return nil
}

func (p ScoringPolicy) Points(o Outcome) int {
	switch o {
	case OutcomeWin:
		return p.WinPoints
	case OutcomeDraw:
		return p.DrawPoints
	case OutcomeLoss:
		return p.LossPoints
	case OutcomeDisqualified:
		return p.LossPoints - p.DisqualificationPenalty
	}
	return 0
}

// ParseScoringPolicy decodes a policy with a closed schema: unknown fields,
// trailing data and missing tiebreaker lists are rejected.
func ParseScoringPolicy(raw []byte) (ScoringPolicy, error) {
	var policy ScoringPolicy
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return ScoringPolicy{}, Validation("invalid_scoring_policy", "scoring policy is not valid: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ScoringPolicy{}, Validation("invalid_scoring_policy", "unexpected data after scoring policy")
	}
	if policy.Tiebreakers == nil {
		return ScoringPolicy{}, Validation("invalid_scoring_policy", "tiebreakers must be declared (use [] for none)")
	}
	return policy, policy.Validate()
}
