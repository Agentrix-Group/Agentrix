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
	// ActionStatusInactive marks a player that received no perception for
	// the tick (out of the simulation); the engine applies no action.
	ActionStatusInactive = "inactive"
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

// TickRate is the exact simulation rate (ticks per second as a ratio).
type TickRate struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

// InitializeMatchRequest contains game startup parameters sent to the engine.
// Config is the opaque game configuration sealed in the ExecutionSpec; the
// platform never interprets it.
type InitializeMatchRequest struct {
	MatchID                    string                 `json:"matchId"`
	GameID                     string                 `json:"gameId"`
	GameVersion                string                 `json:"gameVersion"`
	ExpectedEngineVersion      string                 `json:"expectedEngineVersion"`
	Seed                       int64                  `json:"seed"`
	TickRate                   TickRate               `json:"tickRate"`
	MaxTicks                   int                    `json:"maxTicks"`
	Players                    []string               `json:"players"`
	Config                     map[string]interface{} `json:"config"`
	ParticipantArtifactDigests map[string]string      `json:"participantArtifactDigests"`
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
