package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type TiebreakerRule string

const (
	TiebreakerScoreDiff     TiebreakerRule = "score_diff"
	TiebreakerHeadToHead    TiebreakerRule = "head_to_head"
	TiebreakerWins          TiebreakerRule = "wins"
	TiebreakerRivalSurvival TiebreakerRule = "rival_survival"
)

var (
	ErrInvalidScoringPolicy = errors.New("invalid scoring policy")
)

// ScoringPolicy defines the deterministic points and tiebreaker evaluation criteria for a tournament.
type ScoringPolicy struct {
	WinPoints               int              `json:"win_points"`
	DrawPoints              int              `json:"draw_points"`
	LossPoints              int              `json:"loss_points"`
	DisqualificationPenalty int              `json:"disqualification_penalty"`
	Tiebreakers             []TiebreakerRule `json:"tiebreakers"`
}

// DefaultScoringPolicy provides the canonical standard configuration.
func DefaultScoringPolicy() ScoringPolicy {
	return ScoringPolicy{
		WinPoints:               3,
		DrawPoints:              1,
		LossPoints:              0,
		DisqualificationPenalty: 0,
		Tiebreakers: []TiebreakerRule{
			TiebreakerScoreDiff,
			TiebreakerHeadToHead,
			TiebreakerWins,
		},
	}
}

// Validate verifies that the scoring policy contains coherent and valid tiebreaker rules.
func (sp *ScoringPolicy) Validate() error {
	if sp.WinPoints < sp.DrawPoints {
		return fmt.Errorf("%w: win_points (%d) cannot be less than draw_points (%d)", ErrInvalidScoringPolicy, sp.WinPoints, sp.DrawPoints)
	}
	if sp.DrawPoints < sp.LossPoints {
		return fmt.Errorf("%w: draw_points (%d) cannot be less than loss_points (%d)", ErrInvalidScoringPolicy, sp.DrawPoints, sp.LossPoints)
	}
	if sp.DisqualificationPenalty < 0 {
		return fmt.Errorf("%w: disqualification_penalty (%d) cannot be negative", ErrInvalidScoringPolicy, sp.DisqualificationPenalty)
	}
	for _, tb := range sp.Tiebreakers {
		switch tb {
		case TiebreakerScoreDiff, TiebreakerHeadToHead, TiebreakerWins, TiebreakerRivalSurvival:
		default:
			return fmt.Errorf("%w: unknown tiebreaker rule %q", ErrInvalidScoringPolicy, tb)
		}
	}
	return nil
}

// PointsForResult calculates awarded tournament points given a normalized result outcome.
func (sp *ScoringPolicy) PointsForResult(status string) int {
	switch status {
	case "win", "1":
		return sp.WinPoints
	case "draw":
		return sp.DrawPoints
	case "loss", "2":
		return sp.LossPoints
	case "disqualified":
		return sp.LossPoints - sp.DisqualificationPenalty
	case "no_contest":
		return 0
	default:
		return 0
	}
}

// Value implements the database/sql driver.Valuer interface for JSONB persistence.
func (sp ScoringPolicy) Value() (driver.Value, error) {
	return json.Marshal(sp)
}

// Scan implements the database/sql Scanner interface for JSONB retrieval.
func (sp *ScoringPolicy) Scan(value interface{}) error {
	if value == nil {
		*sp = DefaultScoringPolicy()
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into ScoringPolicy", value)
	}

	if len(bytes) == 0 {
		*sp = DefaultScoringPolicy()
		return nil
	}

	return json.Unmarshal(bytes, sp)
}
