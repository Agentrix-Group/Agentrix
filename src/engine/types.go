package engine

import "encoding/json"

const (
	ProtocolVersion = "agentrix-engine/1"

	// Command message types (Agentrix -> Engine)
	TypeInitializeMatch = "initialize_match"
	TypeAdvanceTick     = "advance_tick"
	TypeFinishMatch     = "finish_match"
	TypeShutdown        = "shutdown"

	// Response message types (Engine -> Agentrix)
	TypeEngineReady      = "engine_ready"
	TypeMatchInitialized = "match_initialized"
	TypeTickCompleted    = "tick_completed"
	TypeMatchCompleted   = "match_completed"
	TypeEngineError      = "engine_error"
	TypeShutdownAck      = "shutdown_ack"

	// Action statuses from Agentrix to Engine
	ActionStatusValid         = "valid"
	ActionStatusTimeout       = "timeout"
	ActionStatusInvalidOutput = "invalid_output"
	ActionStatusCrashed       = "crashed"
	ActionStatusDisqualified  = "disqualified"
)

// Envelope represents the outer wrapper for all JSON Lines protocol messages.
type Envelope struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Type            string                 `json:"type"`
	MatchID         string                 `json:"matchId"`
	Sequence        uint64                 `json:"sequence"`
	Payload         map[string]interface{} `json:"payload"`
}

// PlayerActionInput describes a participant's turn evaluation sent to the engine.
type PlayerActionInput struct {
	Status       string          `json:"status"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	ErrorDetails string          `json:"errorDetails,omitempty"`
}

// InitializeMatchRequest contains game startup parameters sent to the engine.
type InitializeMatchRequest struct {
	MatchID                    string                 `json:"matchId"`
	GameID                     string                 `json:"gameId"`
	ExpectedEngineVersion      string                 `json:"expectedEngineVersion,omitempty"`
	Seed                       int64                  `json:"seed"`
	FixedTimestepMs            int                    `json:"fixedTimestepMs,omitempty"`
	TickHz                     float64                `json:"tickHz,omitempty"`
	MaxTicks                   int                    `json:"maxTicks"`
	Players                    []string               `json:"players"`
	Config                     map[string]interface{} `json:"config,omitempty"`
	ParticipantArtifactDigests map[string]string      `json:"participantArtifactDigests,omitempty"`
}

// StarfighterConfig represents the authoritative game configuration for Starfighter.
type StarfighterConfig struct {
	TickHz                  float64 `json:"tick_hz" yaml:"tick_hz"`
	ArenaWidth              float64 `json:"arena_width" yaml:"arena_width"`
	ArenaHeight             float64 `json:"arena_height" yaml:"arena_height"`
	ShipMaxHealth           float64 `json:"ship_max_health" yaml:"ship_max_health"`
	ShipMaxEnergy           float64 `json:"ship_max_energy" yaml:"ship_max_energy"`
	BulletDamage            float64 `json:"bullet_damage" yaml:"bullet_damage"`
	AsteroidDamage          float64 `json:"asteroid_damage" yaml:"asteroid_damage"`
	ShootEnergyCost         float64 `json:"shoot_energy_cost" yaml:"shoot_energy_cost"`
	ShieldEnergyCostPerTick float64 `json:"shield_energy_cost_per_tick" yaml:"shield_energy_cost_per_tick"`
	ShieldDamageReduction   float64 `json:"shield_damage_reduction" yaml:"shield_damage_reduction"`
	EnergyRegenPerTick      float64 `json:"energy_regen_per_tick" yaml:"energy_regen_per_tick"`
	RadarRange              float64 `json:"radar_range" yaml:"radar_range"`
	AsteroidCount           int     `json:"asteroid_count,omitempty" yaml:"asteroid_count,omitempty"`
}

func DefaultStarfighterConfig() StarfighterConfig {
	return StarfighterConfig{
		TickHz:                  60.0,
		ArenaWidth:              2000.0,
		ArenaHeight:             1000.0,
		ShipMaxHealth:           100.0,
		ShipMaxEnergy:           100.0,
		BulletDamage:            25.0,
		AsteroidDamage:          100.0,
		ShootEnergyCost:         15.0,
		ShieldEnergyCostPerTick: 1.0,
		ShieldDamageReduction:   0.7,
		EnergyRegenPerTick:      0.5,
		RadarRange:              800.0,
		AsteroidCount:           5,
	}
}

// MatchInitializedResult represents the engine's confirmation and initial perception state.
type MatchInitializedResult struct {
	MatchID        string                     `json:"matchId"`
	InitialTick    int                        `json:"initialTick"`
	StateHash      string                     `json:"stateHash"`
	PublicSnapshot json.RawMessage            `json:"publicSnapshot"`
	Perceptions    map[string]json.RawMessage `json:"perceptions"`
	Events         []string                   `json:"events"`
}

// AdvanceTickRequest supplies the next tick index and all agent actions to the engine.
type AdvanceTickRequest struct {
	Tick    int                          `json:"tick"`
	Actions map[string]PlayerActionInput `json:"actions"`
}

// TickResult represents the outcome of simulating one tick.
type TickResult struct {
	Tick           int                        `json:"tick"`
	Events         []string                   `json:"events"`
	StateHash      string                     `json:"stateHash"`
	IsOver         bool                       `json:"isOver"`
	Winner         string                     `json:"winner,omitempty"`
	PublicSnapshot json.RawMessage            `json:"publicSnapshot"`
	Perceptions    map[string]json.RawMessage `json:"perceptions,omitempty"`
}

// PlayerRank describes the final placement and score of a participant.
type PlayerRank struct {
	PlayerID string `json:"playerId"`
	Rank     int    `json:"rank"`
	Score    int    `json:"score"`
}

// MatchResult consolidates the final outcome of the match returned by the engine.
type MatchResult struct {
	FinalTick      int                    `json:"finalTick"`
	Reason         string                 `json:"reason"`
	Winner         string                 `json:"winner,omitempty"`
	Scores         map[string]int         `json:"scores"`
	Rankings       []PlayerRank           `json:"rankings"`
	FinalStateHash string                 `json:"finalStateHash"`
	ReplayMetadata map[string]interface{} `json:"replayMetadata,omitempty"`
}

// EngineReadyPayload defines the handshake payload sent by the engine upon process startup.
type EngineReadyPayload struct {
	EngineVersion      string                 `json:"engineVersion"`
	SupportedProtocols []string               `json:"supportedProtocols"`
	EngineDigest       string                 `json:"engineDigest,omitempty"`
	Capabilities       map[string]interface{} `json:"capabilities,omitempty"`
}

// EngineErrorPayload defines the payload sent by the engine when reporting an error.
type EngineErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Fatal   bool   `json:"fatal"`
}
