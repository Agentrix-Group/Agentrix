package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrInvalidExecutionSpec  = errors.New("invalid execution spec")
	ErrEngineDigestMismatch  = errors.New("engine binary digest mismatch")
	ErrIncompatibleProtocol  = errors.New("incompatible engine protocol version")
)

// ExecutionSlotSpec defines an immutable participant slot in a match execution.
type ExecutionSlotSpec struct {
	SlotIndex    int    `json:"slot_index"`
	SubmissionID string `json:"submission_id"`
	AgentID      string `json:"agent_id,omitempty"`
	Digest       string `json:"digest,omitempty"`
}

// ExecutionLimits defines simulation boundaries and rate configurations.
type ExecutionLimits struct {
	MaxTicks        int     `json:"max_ticks"`
	FixedTimestepMs int     `json:"fixed_timestep_ms"`
	TickHz          float64 `json:"tick_hz"`
	TimeoutMs       int     `json:"timeout_ms,omitempty"`
	MaxLineBytes    int     `json:"max_line_bytes,omitempty"`
}

// ExecutionSpec is an immutable, serializable, and hashable specification of a match execution.
// It establishes the complete reproducible envelope: protocol, engine identity, game config, limits, and participants.
type ExecutionSpec struct {
	ProtocolVersion string                 `json:"protocol_version"`
	GameID          string                 `json:"game_id"`
	GameVersion     string                 `json:"game_version"`
	EngineVersion   string                 `json:"engine_version"`
	EngineDigest    string                 `json:"engine_digest"`
	Config          map[string]interface{} `json:"config"`
	ConfigHash      string                 `json:"config_hash"`
	Seed            int64                  `json:"seed"`
	Limits          ExecutionLimits        `json:"limits"`
	Slots           []ExecutionSlotSpec    `json:"slots"`
	ReplayVersion   string                 `json:"replay_version"`
	SpecHash        string                 `json:"spec_hash,omitempty"`
}

// ComputeConfigHash computes a deterministic SHA256 digest of the game configuration.
func ComputeConfigHash(config map[string]interface{}) (string, error) {
	if config == nil {
		config = make(map[string]interface{})
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}

// ComputeConfigHash computes the SHA256 digest for the spec's Config.
func (s *ExecutionSpec) ComputeConfigHash() (string, error) {
	return ComputeConfigHash(s.Config)
}

// ComputeSpecHash computes the deterministic SHA256 digest of the execution specification,
// excluding the SpecHash field itself.
func (s *ExecutionSpec) ComputeSpecHash() (string, error) {
	copySpec := *s
	copySpec.SpecHash = ""
	raw, err := json.Marshal(copySpec)
	if err != nil {
		return "", fmt.Errorf("marshal execution spec: %w", err)
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}

// Seal computes and populates ConfigHash and SpecHash.
func (s *ExecutionSpec) Seal() error {
	cfgHash, err := s.ComputeConfigHash()
	if err != nil {
		return err
	}
	s.ConfigHash = cfgHash

	specHash, err := s.ComputeSpecHash()
	if err != nil {
		return err
	}
	s.SpecHash = specHash
	return nil
}

// Validate ensures all required fields are populated, slots are ordered, and hashes match.
func (s *ExecutionSpec) Validate() error {
	if s.ProtocolVersion == "" {
		return fmt.Errorf("%w: missing protocol_version", ErrInvalidExecutionSpec)
	}
	if s.GameID == "" {
		return fmt.Errorf("%w: missing game_id", ErrInvalidExecutionSpec)
	}
	if s.Limits.MaxTicks <= 0 {
		return fmt.Errorf("%w: max_ticks must be > 0", ErrInvalidExecutionSpec)
	}
	if s.Limits.TickHz <= 0 {
		return fmt.Errorf("%w: tick_hz must be > 0", ErrInvalidExecutionSpec)
	}
	if len(s.Slots) < 2 {
		return fmt.Errorf("%w: match must have at least 2 participant slots", ErrInvalidExecutionSpec)
	}
	for i, slot := range s.Slots {
		if slot.SlotIndex != i {
			return fmt.Errorf("%w: slot index %d does not match position %d", ErrInvalidExecutionSpec, slot.SlotIndex, i)
		}
		if slot.SubmissionID == "" {
			return fmt.Errorf("%w: slot %d has empty submission_id", ErrInvalidExecutionSpec, i)
		}
	}

	expectedConfigHash, err := s.ComputeConfigHash()
	if err != nil {
		return fmt.Errorf("%w: failed to compute config hash: %v", ErrInvalidExecutionSpec, err)
	}
	if s.ConfigHash != "" && s.ConfigHash != expectedConfigHash {
		return fmt.Errorf("%w: config_hash mismatch (expected %s, got %s)", ErrInvalidExecutionSpec, expectedConfigHash, s.ConfigHash)
	}

	if s.SpecHash != "" {
		expectedSpecHash, err := s.ComputeSpecHash()
		if err != nil {
			return fmt.Errorf("%w: failed to compute spec hash: %v", ErrInvalidExecutionSpec, err)
		}
		if s.SpecHash != expectedSpecHash {
			return fmt.Errorf("%w: spec_hash mismatch (expected %s, got %s)", ErrInvalidExecutionSpec, expectedSpecHash, s.SpecHash)
		}
	}

	return nil
}

// ToJSON serializes the ExecutionSpec to a JSON string.
func (s *ExecutionSpec) ToJSON() (string, error) {
	bytes, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ParseExecutionSpec deserializes a JSON string into an ExecutionSpec and validates it.
func ParseExecutionSpec(raw string) (*ExecutionSpec, error) {
	if raw == "" {
		return nil, fmt.Errorf("%w: empty execution spec string", ErrInvalidExecutionSpec)
	}
	var spec ExecutionSpec
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return nil, fmt.Errorf("%w: json unmarshal: %v", ErrInvalidExecutionSpec, err)
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return &spec, nil
}

// ComputeFileSHA256 calculates the SHA256 hexadecimal digest of any file on disk.
func ComputeFileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
