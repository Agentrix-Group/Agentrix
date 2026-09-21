package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
)

const ExecutionSpecVersion = "agentrix-execution-spec/1"

// TickRate is the canonical simulation rate as an exact ratio of ticks per
// second (60/1 = 60 Hz). Every consumer (engine, replay, viewer, gym) derives
// its timestep from this value; no millisecond approximation is stored.
type TickRate struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

func (r TickRate) Valid() bool {
	return r.Numerator > 0 && r.Denominator > 0 && r.Numerator <= 1000 && r.Denominator <= 1000
}

type ExecutionLimits struct {
	MaxTicks      int `json:"max_ticks"`
	TurnTimeoutMs int `json:"turn_timeout_ms"`
	InitTimeoutMs int `json:"init_timeout_ms"`
	WallTimeMs    int `json:"wall_time_ms"`
	MemoryMB      int `json:"memory_mb"`
	CPUSeconds    int `json:"cpu_seconds"`
	MaxProcesses  int `json:"max_processes"`
	MaxOutputKB   int `json:"max_output_kb"`
	MaxFileSizeKB int `json:"max_file_size_kb"`
	MaxStderrKB   int `json:"max_stderr_kb"`
	MaxLineBytes  int `json:"max_line_bytes"`
}

type SpecGame struct {
	ID         string         `json:"id"`
	Version    string         `json:"version"`
	Config     map[string]any `json:"config"`
	ConfigHash string         `json:"config_hash"`
}

type SpecEngine struct {
	Version         string `json:"version"`
	SHA256          string `json:"sha256"`
	ProtocolVersion string `json:"protocol_version"`
}

type SpecRuntime struct {
	BotProtocolVersion string `json:"bot_protocol_version"`
	SandboxProfile     string `json:"sandbox_profile"`
}

type SpecSlot struct {
	Index          int    `json:"index"`
	SlotID         string `json:"slot_id"`
	ContestEntryID string `json:"contest_entry_id,omitempty"`
	AgentID        string `json:"agent_id"`
	SubmissionID   string `json:"submission_id"`
	ArtifactKey    string `json:"artifact_key"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	Runtime        string `json:"runtime"`
	Entrypoint     string `json:"entrypoint"`
}

// ExecutionSpec is the complete, immutable description of a run. The
// executor uses only this document; it never re-reads live manifests or
// submissions to rebuild parameters.
type ExecutionSpec struct {
	SpecVersion  string          `json:"spec_version"`
	MatchID      string          `json:"match_id"`
	Mode         MatchMode       `json:"mode"`
	Seed         int64           `json:"seed"`
	TickRate     TickRate        `json:"tick_rate"`
	Limits       ExecutionLimits `json:"limits"`
	Game         SpecGame        `json:"game"`
	Engine       SpecEngine      `json:"engine"`
	Runtime      SpecRuntime     `json:"runtime"`
	ReplayFormat string          `json:"replay_format"`
	Slots        []SpecSlot      `json:"slots"`
}

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func IsSHA256(value string) bool { return sha256Pattern.MatchString(value) }

// CanonicalJSON serializes a value deterministically: struct fields keep
// their declaration order and map keys are sorted by encoding/json.
func CanonicalJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ConfigHash hashes the canonical JSON of a game configuration.
func ConfigHash(config map[string]any) (string, error) {
	raw, err := CanonicalJSON(config)
	if err != nil {
		return "", fmt.Errorf("canonical game config: %w", err)
	}
	return SHA256Hex(raw), nil
}

// Seal fills the config hash and returns the canonical bytes and spec hash.
// Every error is returned; a spec that cannot be sealed must not be queued.
func (s *ExecutionSpec) Seal() ([]byte, string, error) {
	hash, err := ConfigHash(s.Game.Config)
	if err != nil {
		return nil, "", err
	}
	s.Game.ConfigHash = hash
	if err := s.Validate(); err != nil {
		return nil, "", err
	}
	raw, err := CanonicalJSON(s)
	if err != nil {
		return nil, "", fmt.Errorf("canonical execution spec: %w", err)
	}
	return raw, SHA256Hex(raw), nil
}

// Hash recomputes the spec hash from the canonical serialization.
func (s ExecutionSpec) Hash() (string, error) {
	raw, err := CanonicalJSON(s)
	if err != nil {
		return "", err
	}
	return SHA256Hex(raw), nil
}

func specError(format string, args ...any) error {
	return &Error{Kind: KindValidation, Code: "invalid_execution_spec", Message: fmt.Sprintf(format, args...)}
}

// Validate checks structural completeness. Competitive specs additionally
// require a contest entry for every slot and distinct submissions.
func (s ExecutionSpec) Validate() error {
	switch {
	case s.SpecVersion != ExecutionSpecVersion:
		return specError("unsupported spec_version %q", s.SpecVersion)
	case s.MatchID == "":
		return specError("match_id is required")
	case !s.Mode.Valid():
		return specError("invalid mode %q", s.Mode)
	case !s.TickRate.Valid():
		return specError("tick_rate must be a positive ratio")
	case s.Limits.MaxTicks <= 0 || s.Limits.TurnTimeoutMs <= 0 || s.Limits.InitTimeoutMs <= 0 || s.Limits.WallTimeMs <= 0:
		return specError("limits max_ticks, turn_timeout_ms, init_timeout_ms and wall_time_ms must be positive")
	case s.Limits.MemoryMB <= 0 || s.Limits.CPUSeconds <= 0 || s.Limits.MaxProcesses <= 0 ||
		s.Limits.MaxOutputKB <= 0 || s.Limits.MaxFileSizeKB <= 0 || s.Limits.MaxStderrKB <= 0 || s.Limits.MaxLineBytes <= 0:
		return specError("resource limits must be positive")
	case s.Game.ID == "" || s.Game.Version == "":
		return specError("game id and version are required")
	case !IsSHA256(s.Game.ConfigHash):
		return specError("game config_hash is missing")
	case s.Engine.Version == "" || s.Engine.ProtocolVersion == "":
		return specError("engine version and protocol are required")
	case !IsSHA256(s.Engine.SHA256):
		return specError("engine sha256 is required")
	case s.Runtime.BotProtocolVersion == "" || s.Runtime.SandboxProfile == "":
		return specError("runtime bot protocol and sandbox profile are required")
	case s.ReplayFormat == "":
		return specError("replay_format is required")
	case len(s.Slots) == 0:
		return specError("at least one slot is required")
	}
	if hash, err := ConfigHash(s.Game.Config); err != nil || hash != s.Game.ConfigHash {
		return specError("game config_hash does not match config")
	}
	seenSlots := map[string]bool{}
	seenSubmissions := map[string]bool{}
	for i, slot := range s.Slots {
		switch {
		case slot.Index != i:
			return specError("slot %d has index %d", i, slot.Index)
		case slot.SlotID == "" || slot.AgentID == "" || slot.SubmissionID == "":
			return specError("slot %d is missing identifiers", i)
		case seenSlots[slot.SlotID]:
			return specError("slot id %s is duplicated", slot.SlotID)
		case !IsSHA256(slot.ArtifactSHA256):
			return specError("slot %d has no artifact digest", i)
		case slot.ArtifactKey == "" || slot.Runtime == "" || slot.Entrypoint == "":
			return specError("slot %d is missing runtime metadata", i)
		}
		if s.Mode == ModeCompetitive {
			if slot.ContestEntryID == "" {
				return specError("competitive slot %d has no contest entry", i)
			}
			if seenSubmissions[slot.SubmissionID] {
				return specError("competitive match repeats submission %s", slot.SubmissionID)
			}
		}
		seenSlots[slot.SlotID] = true
		seenSubmissions[slot.SubmissionID] = true
	}
	return nil
}

// ParseExecutionSpec decodes a persisted spec strictly and verifies its hash.
func ParseExecutionSpec(raw []byte, expectedHash string) (ExecutionSpec, error) {
	var spec ExecutionSpec
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(&spec); err != nil {
		return ExecutionSpec{}, specError("execution spec is not valid JSON: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ExecutionSpec{}, specError("unexpected data after execution spec")
	}
	spec.Game.Config = normalizeNumbers(spec.Game.Config).(map[string]any)
	if err := spec.Validate(); err != nil {
		return ExecutionSpec{}, err
	}
	hash, err := spec.Hash()
	if err != nil {
		return ExecutionSpec{}, specError("execution spec cannot be serialized: %v", err)
	}
	if hash != expectedHash {
		return ExecutionSpec{}, specError("execution spec hash mismatch")
	}
	return spec, nil
}

// normalizeNumbers converts json.Number values (produced by UseNumber) back
// into the float64 representation used when the spec was sealed, so the
// canonical serialization round-trips byte for byte.
func normalizeNumbers(value any) any {
	switch v := value.(type) {
	case map[string]any:
		if v == nil {
			return map[string]any{}
		}
		for key, item := range v {
			v[key] = normalizeNumbers(item)
		}
		return v
	case []any:
		for i, item := range v {
			v[i] = normalizeNumbers(item)
		}
		return v
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return v.String()
		}
		return f
	case nil:
		return nil
	default:
		return v
	}
}
