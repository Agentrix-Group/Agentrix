package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const (
	ExecutionSpecProtocolVersion = "agentrix-engine/2"
	DefaultRNGAlgorithm          = "xoshiro256starstar/1"
	DefaultFailurePolicyVersion  = "fail-closed/1"
	TierSameArtifactSameTarget   = "same_artifact_same_target"
	TierCertifiedTargetMatrix    = "certified_target_matrix"

	// Starfighter 0.4.0 (ADR-0013): Rapier, de 2 a 5 jugadores todos contra
	// todos. Los digests son los de agentrix_engine/schemas/games/starfighter/0.4.0.
	StarfighterGameID                  = "starfighter"
	StarfighterGameVersion             = "0.4.0"
	StarfighterEngineVersion           = "0.4.0"
	StarfighterMinPlayers              = 2
	StarfighterMaxPlayers              = 5
	StarfighterActionSchemaDigest      = "c97852810619ca799214c65f54acbdd1cd6e4ea98c55ea21dc7f21c73e004848"
	StarfighterObservationSchemaDigest = "fecb52b52fcb6d3af1bf69d2e83f4dcb300ac9dbbbb0f6fbaeeb3c310bee126d"
	StarfighterPublicSchemaDigest      = "a40acf5714e7f4571b82b8c67829987f98e80f23aa4c802190d44770d79581d9"
	StarfighterReplaySchemaDigest      = "0000000000000000000000000000000000000000000000000000000000000000"
)

// DeriveStarfighterGameDigest computes the canonical game manifest digest.
func DeriveStarfighterGameDigest() (string, error) {
	manifest := map[string]interface{}{
		"gameId":      StarfighterGameID,
		"gameVersion": StarfighterGameVersion,
		"schemas": map[string]interface{}{
			"action":      StarfighterActionSchemaDigest,
			"observation": StarfighterObservationSchemaDigest,
			"public":      StarfighterPublicSchemaDigest,
		},
	}
	return CanonicalJSONDigest("agentrix.game-manifest/1", manifest)
}

// StarfighterGameKey returns the validated canonical GameKey for Starfighter.
func StarfighterGameKey() (GameKey, error) {
	digest, err := DeriveStarfighterGameDigest()
	if err != nil {
		return GameKey{}, err
	}
	return GameKey{
		GameID:      StarfighterGameID,
		GameVersion: StarfighterGameVersion,
		GameDigest:  digest,
	}, nil
}

// StarfighterSchemaDigests returns the canonical schema digests for Starfighter.
func StarfighterSchemaDigests() SchemaDigests {
	return SchemaDigests{
		Action:      StarfighterActionSchemaDigest,
		Observation: StarfighterObservationSchemaDigest,
		Public:      StarfighterPublicSchemaDigest,
		Replay:      StarfighterReplaySchemaDigest,
	}
}

var (
	ErrInvalidExecutionSpec = errors.New("invalid execution spec")
	ErrEngineDigestMismatch = errors.New("engine binary digest mismatch")
	ErrIncompatibleProtocol = errors.New("incompatible engine protocol version")

	sha256Regex = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func IsValidSHA256(s string) bool {
	return sha256Regex.MatchString(s)
}

func isValidSHA256(s string) bool {
	return IsValidSHA256(s)
}

func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// TickRate represents the simulation update frequency as a reduced fraction.
type TickRate struct {
	Numerator   uint32 `json:"numerator"`
	Denominator uint32 `json:"denominator"`
}

func NewTickRate(numerator, denominator uint32) (TickRate, error) {
	if numerator == 0 || denominator == 0 {
		return TickRate{}, fmt.Errorf("%w: tick rate numerator and denominator must be positive", ErrInvalidExecutionSpec)
	}
	if numerator > 10000 || denominator > 10000 {
		return TickRate{}, fmt.Errorf("%w: tick rate numerator and denominator must not exceed 10000", ErrInvalidExecutionSpec)
	}
	g := gcd(numerator, denominator)
	return TickRate{
		Numerator:   numerator / g,
		Denominator: denominator / g,
	}, nil
}

func (tr TickRate) SecondsPerTick() float64 {
	return float64(tr.Denominator) / float64(tr.Numerator)
}

// GameKey identifies the game implementation and version.
type GameKey struct {
	GameID      string `json:"gameId"`
	GameVersion string `json:"gameVersion"`
	GameDigest  string `json:"gameDigest"`
}

func (g GameKey) Validate() error {
	if strings.TrimSpace(g.GameID) == "" {
		return fmt.Errorf("%w: gameId must not be empty", ErrInvalidExecutionSpec)
	}
	if strings.TrimSpace(g.GameVersion) == "" {
		return fmt.Errorf("%w: gameVersion must not be empty", ErrInvalidExecutionSpec)
	}
	if !isValidSHA256(g.GameDigest) {
		return fmt.Errorf("%w: gameDigest must be a 64-character lowercase hex sha256", ErrInvalidExecutionSpec)
	}
	return nil
}

// SchemaDigests contains canonical SHA-256 hashes of the schemas.
type SchemaDigests struct {
	Action      string `json:"action"`
	Observation string `json:"observation"`
	Public      string `json:"public"`
	Replay      string `json:"replay"`
}

func (s SchemaDigests) Validate() error {
	for name, digest := range map[string]string{
		"action":      s.Action,
		"observation": s.Observation,
		"public":      s.Public,
		"replay":      s.Replay,
	} {
		if !isValidSHA256(digest) {
			return fmt.Errorf("%w: %s schema digest must be a 64-character lowercase hex sha256", ErrInvalidExecutionSpec, name)
		}
	}
	return nil
}

// ExecutionSlotSpec defines an immutable participant slot in a match execution.
type ExecutionSlotSpec struct {
	SlotID         string `json:"slotId"`
	ArtifactDigest string `json:"artifactDigest"`
}

func (s ExecutionSlotSpec) Validate() error {
	if strings.TrimSpace(s.SlotID) == "" {
		return fmt.Errorf("%w: slotId must not be empty", ErrInvalidExecutionSpec)
	}
	if !isValidSHA256(s.ArtifactDigest) {
		return fmt.Errorf("%w: artifactDigest must be a 64-character lowercase hex sha256", ErrInvalidExecutionSpec)
	}
	return nil
}

// ExecutionLimits defines simulation boundaries and rate configurations.
type ExecutionLimits struct {
	MaxTicks        uint64 `json:"maxTicks"`
	MaxPlayers      uint32 `json:"maxPlayers"`
	MaxEntities     uint32 `json:"maxEntities"`
	MaxMessageBytes uint32 `json:"maxMessageBytes"`
}

func (l ExecutionLimits) Validate() error {
	if l.MaxTicks == 0 {
		return fmt.Errorf("%w: maxTicks must be > 0", ErrInvalidExecutionSpec)
	}
	if l.MaxPlayers == 0 || l.MaxPlayers > 64 {
		return fmt.Errorf("%w: maxPlayers must be between 1 and 64", ErrInvalidExecutionSpec)
	}
	if l.MaxEntities == 0 {
		return fmt.Errorf("%w: maxEntities must be > 0", ErrInvalidExecutionSpec)
	}
	if l.MaxMessageBytes < 1024 {
		return fmt.Errorf("%w: maxMessageBytes must be >= 1024", ErrInvalidExecutionSpec)
	}
	return nil
}

// ExecutionSpec is an immutable, serializable, and hashable specification of a match execution.
// It establishes the complete reproducible envelope conforming to agentrix-engine/2.
type ExecutionSpec struct {
	ProtocolVersion      string                 `json:"protocolVersion"`
	RunID                string                 `json:"runId"`
	MatchID              string                 `json:"matchId"`
	EngineVersion        string                 `json:"engineVersion"`
	EngineDigest         string                 `json:"engineDigest"`
	BuildIdentity        string                 `json:"buildIdentity"`
	Target               string                 `json:"target"`
	Game                 GameKey                `json:"game"`
	SchemaDigests        SchemaDigests          `json:"schemaDigests"`
	Config               map[string]interface{} `json:"config"`
	ConfigDigest         string                 `json:"configDigest"`
	TickRate             TickRate               `json:"tickRate"`
	Seed                 uint64                 `json:"seed"`
	Slots                []ExecutionSlotSpec    `json:"slots"`
	Limits               ExecutionLimits        `json:"limits"`
	FailurePolicyVersion string                 `json:"failurePolicyVersion"`
	DeterminismTier      string                 `json:"determinismTier"`
	RNGAlgorithm         string                 `json:"rngAlgorithm"`
}

// ComputeConfigDigest computes a deterministic canonical SHA256 digest of the game configuration.
func (s *ExecutionSpec) ComputeConfigDigest() (string, error) {
	domain := "starfighter-config"
	if s.Game.GameID != "" && s.Game.GameID != "starfighter" {
		domain = s.Game.GameID + "-config"
	}
	cfg := s.Config
	if cfg == nil {
		cfg = make(map[string]interface{})
	}
	return CanonicalJSONDigest(domain, cfg)
}

// ComputeDigest computes the deterministic canonical SHA256 digest of the execution specification.
func (s *ExecutionSpec) ComputeDigest() (string, error) {
	return CanonicalJSONDigest("agentrix.execution-spec/2", s)
}

// Seal computes and validates ConfigDigest.
func (s *ExecutionSpec) Seal() error {
	cfgDigest, err := s.ComputeConfigDigest()
	if err != nil {
		return fmt.Errorf("%w: compute config digest: %v", ErrInvalidExecutionSpec, err)
	}
	s.ConfigDigest = cfgDigest
	return s.Validate()
}

// Validate ensures all required fields are populated, slots are valid, and hashes match.
func (s *ExecutionSpec) Validate() error {
	if s.ProtocolVersion != ExecutionSpecProtocolVersion {
		return fmt.Errorf("%w: expected protocolVersion %s, got %s", ErrInvalidExecutionSpec, ExecutionSpecProtocolVersion, s.ProtocolVersion)
	}
	if strings.TrimSpace(s.RunID) == "" {
		return fmt.Errorf("%w: runId must not be empty", ErrInvalidExecutionSpec)
	}
	if strings.TrimSpace(s.MatchID) == "" {
		return fmt.Errorf("%w: matchId must not be empty", ErrInvalidExecutionSpec)
	}
	if strings.TrimSpace(s.EngineVersion) == "" {
		return fmt.Errorf("%w: engineVersion must not be empty", ErrInvalidExecutionSpec)
	}
	if !isValidSHA256(s.EngineDigest) {
		return fmt.Errorf("%w: engineDigest must be a 64-character lowercase hex sha256", ErrInvalidExecutionSpec)
	}
	if strings.TrimSpace(s.BuildIdentity) == "" {
		return fmt.Errorf("%w: buildIdentity must not be empty", ErrInvalidExecutionSpec)
	}
	if strings.TrimSpace(s.Target) == "" {
		return fmt.Errorf("%w: target must not be empty", ErrInvalidExecutionSpec)
	}
	if err := s.Game.Validate(); err != nil {
		return err
	}
	if err := s.SchemaDigests.Validate(); err != nil {
		return err
	}
	if !isValidSHA256(s.ConfigDigest) {
		return fmt.Errorf("%w: configDigest must be a 64-character lowercase hex sha256", ErrInvalidExecutionSpec)
	}
	if s.TickRate.Numerator == 0 || s.TickRate.Denominator == 0 {
		return fmt.Errorf("%w: tickRate numerator and denominator must be positive", ErrInvalidExecutionSpec)
	}
	if gcd(s.TickRate.Numerator, s.TickRate.Denominator) != 1 {
		return fmt.Errorf("%w: tickRate must be in lowest terms", ErrInvalidExecutionSpec)
	}
	if len(s.Slots) == 0 || len(s.Slots) > 64 {
		return fmt.Errorf("%w: slots count must be between 1 and 64", ErrInvalidExecutionSpec)
	}
	for _, slot := range s.Slots {
		if err := slot.Validate(); err != nil {
			return err
		}
	}
	if err := s.Limits.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(s.FailurePolicyVersion) == "" {
		return fmt.Errorf("%w: failurePolicyVersion must not be empty", ErrInvalidExecutionSpec)
	}
	if s.DeterminismTier != TierSameArtifactSameTarget && s.DeterminismTier != TierCertifiedTargetMatrix {
		return fmt.Errorf("%w: unsupported determinismTier: %s", ErrInvalidExecutionSpec, s.DeterminismTier)
	}
	if s.RNGAlgorithm != DefaultRNGAlgorithm {
		return fmt.Errorf("%w: expected rngAlgorithm %s, got %s", ErrInvalidExecutionSpec, DefaultRNGAlgorithm, s.RNGAlgorithm)
	}

	expectedConfigDigest, err := s.ComputeConfigDigest()
	if err != nil {
		return fmt.Errorf("%w: failed to compute config digest: %v", ErrInvalidExecutionSpec, err)
	}
	if s.ConfigDigest != expectedConfigDigest {
		return fmt.Errorf("%w: configDigest mismatch (expected %s, got %s)", ErrInvalidExecutionSpec, expectedConfigDigest, s.ConfigDigest)
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
