package engine

import (
	"encoding/json"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

const (
	ProtocolVersion = "agentrix-engine/2"

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

// Envelope represents the outer wrapper for all JSON Lines protocol messages in agentrix-engine/2.
type Envelope struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Type            string                 `json:"type"`
	MatchID         string                 `json:"matchId"`
	RunID           string                 `json:"runId,omitempty"`
	Sequence        uint64                 `json:"sequence"`
	Payload         map[string]interface{} `json:"payload"`
}

// HostCommitments contains the SHA-256 cryptographic commitments emitted by the engine host.
type HostCommitments struct {
	ExecutionSpecDigest          string `json:"executionSpecDigest"`
	ActionBatchDigest            string `json:"actionBatchDigest"`
	AuthoritativeStateCommitment string `json:"authoritativeStateCommitment"`
	PublicSnapshotHash           string `json:"publicSnapshotHash"`
	ReplayChainDigest            string `json:"replayChainDigest"`
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
	RunID                      string                 `json:"runId,omitempty"`
	GameID                     string                 `json:"gameId,omitempty"`
	ExpectedEngineVersion      string                 `json:"expectedEngineVersion,omitempty"`
	Seed                       int64                  `json:"seed"`
	FixedTimestepMs            int                    `json:"fixedTimestepMs,omitempty"`
	TickHz                     float64                `json:"tickHz,omitempty"`
	MaxTicks                   int                    `json:"maxTicks"`
	Players                    []string               `json:"players,omitempty"`
	Config                     map[string]interface{} `json:"config,omitempty"`
	ParticipantArtifactDigests map[string]string      `json:"participantArtifactDigests,omitempty"`
	Spec                       *model.ExecutionSpec   `json:"spec,omitempty"`
}

// StarfighterConfig represents the authoritative game configuration for Starfighter.
type StarfighterConfig struct {
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
	MatchID        string                     `json:"matchId,omitempty"`
	Tick           int                        `json:"tick"`
	InitialTick    int                        `json:"initialTick,omitempty"`
	StateHash      string                     `json:"stateHash,omitempty"`
	PublicSnapshot json.RawMessage            `json:"publicSnapshot"`
	Perceptions    map[string]json.RawMessage `json:"perceptions,omitempty"`
	Observations   map[string]json.RawMessage `json:"observations,omitempty"`
	Events         []string                   `json:"events"`
	Terminal       bool                       `json:"terminal,omitempty"`
	Result         map[string]interface{}     `json:"result,omitempty"`
	Commitments    HostCommitments            `json:"commitments,omitempty"`
}

func (m *MatchInitializedResult) UnmarshalJSON(data []byte) error {
	type rawInitialized struct {
		MatchID        string                     `json:"matchId,omitempty"`
		Tick           int                        `json:"tick"`
		InitialTick    int                        `json:"initialTick,omitempty"`
		StateHash      string                     `json:"stateHash,omitempty"`
		PublicSnapshot json.RawMessage            `json:"publicSnapshot"`
		Perceptions    map[string]json.RawMessage `json:"perceptions,omitempty"`
		Observations   map[string]json.RawMessage `json:"observations,omitempty"`
		Events         []json.RawMessage          `json:"events"`
		Terminal       bool                       `json:"terminal,omitempty"`
		Result         map[string]interface{}     `json:"result,omitempty"`
		Commitments    HostCommitments            `json:"commitments,omitempty"`
	}
	var aux rawInitialized
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.MatchID = aux.MatchID
	m.Tick = aux.Tick
	m.InitialTick = aux.InitialTick
	m.StateHash = aux.StateHash
	m.PublicSnapshot = aux.PublicSnapshot
	m.Perceptions = aux.Perceptions
	m.Observations = aux.Observations
	m.Terminal = aux.Terminal
	m.Result = aux.Result
	m.Commitments = aux.Commitments
	if m.Perceptions == nil && m.Observations != nil {
		m.Perceptions = m.Observations
	}
	if m.Observations == nil && m.Perceptions != nil {
		m.Observations = m.Perceptions
	}
	if m.StateHash == "" && m.Commitments.AuthoritativeStateCommitment != "" {
		m.StateHash = m.Commitments.AuthoritativeStateCommitment
	}
	m.Events = make([]string, 0, len(aux.Events))
	for _, raw := range aux.Events {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			m.Events = append(m.Events, s)
		} else {
			m.Events = append(m.Events, string(raw))
		}
	}
	return nil
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
	StateHash      string                     `json:"stateHash,omitempty"`
	IsOver         bool                       `json:"isOver,omitempty"`
	Terminal       bool                       `json:"terminal,omitempty"`
	Winner         string                     `json:"winner,omitempty"`
	Result         map[string]interface{}     `json:"result,omitempty"`
	PublicSnapshot json.RawMessage            `json:"publicSnapshot"`
	Perceptions    map[string]json.RawMessage `json:"perceptions,omitempty"`
	Observations   map[string]json.RawMessage `json:"observations,omitempty"`
	Commitments    HostCommitments            `json:"commitments,omitempty"`
}

func (t *TickResult) UnmarshalJSON(data []byte) error {
	type rawTick struct {
		Tick           int                        `json:"tick"`
		Events         []json.RawMessage          `json:"events"`
		StateHash      string                     `json:"stateHash,omitempty"`
		IsOver         bool                       `json:"isOver,omitempty"`
		Terminal       bool                       `json:"terminal,omitempty"`
		Winner         string                     `json:"winner,omitempty"`
		Result         map[string]interface{}     `json:"result,omitempty"`
		PublicSnapshot json.RawMessage            `json:"publicSnapshot"`
		Perceptions    map[string]json.RawMessage `json:"perceptions,omitempty"`
		Observations   map[string]json.RawMessage `json:"observations,omitempty"`
		Commitments    HostCommitments            `json:"commitments,omitempty"`
	}
	var aux rawTick
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.Tick = aux.Tick
	t.StateHash = aux.StateHash
	t.IsOver = aux.IsOver
	t.Terminal = aux.Terminal
	t.Winner = aux.Winner
	t.Result = aux.Result
	t.PublicSnapshot = aux.PublicSnapshot
	t.Perceptions = aux.Perceptions
	t.Observations = aux.Observations
	t.Commitments = aux.Commitments
	if t.Perceptions == nil && t.Observations != nil {
		t.Perceptions = t.Observations
	}
	if t.Observations == nil && t.Perceptions != nil {
		t.Observations = t.Perceptions
	}
	if t.Terminal {
		t.IsOver = true
	}
	if t.StateHash == "" && t.Commitments.AuthoritativeStateCommitment != "" {
		t.StateHash = t.Commitments.AuthoritativeStateCommitment
	}
	t.Events = make([]string, 0, len(aux.Events))
	for _, raw := range aux.Events {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			t.Events = append(t.Events, s)
		} else {
			t.Events = append(t.Events, string(raw))
		}
	}
	return nil
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
	Scores         map[string]int         `json:"scores,omitempty"`
	Rankings       []PlayerRank           `json:"rankings,omitempty"`
	FinalStateHash string                 `json:"finalStateHash,omitempty"`
	Lifecycle      string                 `json:"lifecycle,omitempty"`
	Commitments    HostCommitments        `json:"commitments,omitempty"`
	ReplayMetadata map[string]interface{} `json:"replayMetadata,omitempty"`
}

func (m *MatchResult) UnmarshalJSON(data []byte) error {
	type Alias MatchResult
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*m = MatchResult(aux)
	if m.FinalStateHash == "" && m.Commitments.AuthoritativeStateCommitment != "" {
		m.FinalStateHash = m.Commitments.AuthoritativeStateCommitment
	}
	return nil
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
