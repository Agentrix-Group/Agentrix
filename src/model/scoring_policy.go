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
	// TiebreakerKills compara las bajas acumuladas (score de Starfighter
	// 0.4.0). Es el primer desempate de la política por posición (ADR-0013).
	TiebreakerKills TiebreakerRule = "kills"
)

// ScoringMode elige cómo una partida reparte puntos de torneo.
type ScoringMode string

const (
	// ScoringModeWinDrawLoss es la política original de ADR-0012
	// (victoria/empate/derrota). Una política guardada sin `mode` la usa,
	// así los concursos existentes no reinterpretan sus resultados.
	ScoringModeWinDrawLoss ScoringMode = "win_draw_loss"
	// ScoringModePlacement reparte puntos por posición final (ADR-0013):
	// ver PlacementPoints.
	ScoringModePlacement ScoringMode = "placement"
)

var (
	ErrInvalidScoringPolicy = errors.New("invalid scoring policy")
)

// ScoringPolicy defines the deterministic points and tiebreaker evaluation criteria for a tournament.
type ScoringPolicy struct {
	Mode                    ScoringMode      `json:"mode,omitempty"`
	WinPoints               int              `json:"win_points"`
	DrawPoints              int              `json:"draw_points"`
	LossPoints              int              `json:"loss_points"`
	DisqualificationPenalty int              `json:"disqualification_penalty"`
	Tiebreakers             []TiebreakerRule `json:"tiebreakers"`
}

// DefaultScoringPolicy provides the canonical standard configuration for new
// contests: points by placement with kills as the first tiebreaker
// (ADR-0013). Win/draw/loss points are kept for head-to-head comparisons.
func DefaultScoringPolicy() ScoringPolicy {
	return ScoringPolicy{
		Mode:                    ScoringModePlacement,
		WinPoints:               3,
		DrawPoints:              1,
		LossPoints:              0,
		DisqualificationPenalty: 0,
		Tiebreakers: []TiebreakerRule{
			TiebreakerKills,
			TiebreakerScoreDiff,
			TiebreakerHeadToHead,
			TiebreakerWins,
		},
	}
}

// EffectiveMode devuelve el modo de la política; sin `mode` es la política
// original de victoria/empate/derrota.
func (sp *ScoringPolicy) EffectiveMode() ScoringMode {
	if sp.Mode == "" {
		return ScoringModeWinDrawLoss
	}
	return sp.Mode
}

// PlacementPoints devuelve los puntos de un puesto en una partida de
// `players` participantes, con `tied` participantes compartiendo ese puesto
// de competencia (1, 2, 2, 4): cada puesto p vale 2·(players − p) y los
// empatados se reparten el promedio de los puestos que ocupan. Con 5
// jugadores: 8, 6, 4, 2, 0; dos empatados primeros: 7 cada uno; en 1 contra
// 1: 2 / 1 / 0. Devuelve 0 para entradas fuera de rango.
func PlacementPoints(players, rank, tied int) int {
	if players < 1 || rank < 1 || tied < 1 || rank+tied-1 > players {
		return 0
	}
	return 2*(players-rank) - (tied - 1)
}

// Validate verifies that the scoring policy contains coherent and valid tiebreaker rules.
func (sp *ScoringPolicy) Validate() error {
	switch sp.EffectiveMode() {
	case ScoringModeWinDrawLoss, ScoringModePlacement:
	default:
		return fmt.Errorf("%w: unknown scoring mode %q", ErrInvalidScoringPolicy, sp.Mode)
	}
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
		case TiebreakerScoreDiff, TiebreakerHeadToHead, TiebreakerWins, TiebreakerRivalSurvival, TiebreakerKills:
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
