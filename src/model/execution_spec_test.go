package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validTestSpec() ExecutionSpec {
	return ExecutionSpec{
		ProtocolVersion: ExecutionSpecProtocolVersion,
		RunID:           "run-test-1",
		MatchID:         "match-test-1",
		EngineVersion:   "0.3.0",
		EngineDigest:    strings.Repeat("a", 64),
		BuildIdentity:   "test-build",
		Target:          "x86_64-unknown-linux-gnu",
		Game: GameKey{
			GameID:      "test",
			GameVersion: "1",
			GameDigest:  strings.Repeat("b", 64),
		},
		SchemaDigests: SchemaDigests{
			Action:      strings.Repeat("c", 64),
			Observation: strings.Repeat("d", 64),
			Public:      strings.Repeat("e", 64),
			Replay:      strings.Repeat("f", 64),
		},
		Config: map[string]interface{}{
			"difficulty": 2,
			"nested": map[string]interface{}{
				"enabled": true,
			},
		},
		TickRate: TickRate{
			Numerator:   60,
			Denominator: 1,
		},
		Seed: 7,
		Slots: []ExecutionSlotSpec{
			{SlotID: "alpha", ArtifactDigest: strings.Repeat("1", 64)},
			{SlotID: "beta", ArtifactDigest: strings.Repeat("2", 64)},
		},
		Limits: ExecutionLimits{
			MaxTicks:        10,
			MaxPlayers:      2,
			MaxEntities:     100,
			MaxMessageBytes: 65536,
		},
		FailurePolicyVersion: "failure-policy/1",
		DeterminismTier:      TierSameArtifactSameTarget,
		RNGAlgorithm:         DefaultRNGAlgorithm,
	}
}

func TestExecutionSpec_GoldenFixtureDigest(t *testing.T) {
	r := require.New(t)

	// Locate the golden fixture relative to repo root or package dir
	fixturePath := filepath.Join("..", "..", "contracts", "fixtures", "execution_spec_golden.json")
	if _, err := os.Stat(fixturePath); err != nil {
		fixturePath = filepath.Join("contracts", "fixtures", "execution_spec_golden.json")
	}

	data, err := os.ReadFile(fixturePath)
	r.NoError(err, "golden fixture must exist and be readable")

	var spec ExecutionSpec
	r.NoError(json.Unmarshal(data, &spec))
	r.NoError(spec.Validate(), "golden fixture must be valid according to ExecutionSpec rules")

	digest, err := spec.ComputeDigest()
	r.NoError(err)

	expectedGoldenDigest := "18568c8b3fab8f4d45a9d87f2ff39df18871d37c75117f011f51aadf879d4e72"
	r.Equal(expectedGoldenDigest, digest, "Go canonical execution_spec digest must match Rust golden digest exactly")
}

func TestExecutionSpec_SealAndValidate(t *testing.T) {
	r := require.New(t)
	spec := validTestSpec()

	r.NoError(spec.Seal())
	r.NotEmpty(spec.ConfigDigest)
	r.NoError(spec.Validate())

	// Round-trip serialization
	jsonStr, err := spec.ToJSON()
	r.NoError(err)

	parsed, err := ParseExecutionSpec(jsonStr)
	r.NoError(err)
	r.Equal(spec.ConfigDigest, parsed.ConfigDigest)
	r.Equal(spec.Seed, parsed.Seed)
	r.Equal(2, len(parsed.Slots))

	origDigest, err := spec.ComputeDigest()
	r.NoError(err)
	parsedDigest, err := parsed.ComputeDigest()
	r.NoError(err)
	r.Equal(origDigest, parsedDigest)
}

func TestExecutionSpec_HashDeterminism(t *testing.T) {
	assert := assert.New(t)

	spec1 := validTestSpec()
	require.NoError(t, spec1.Seal())

	spec2 := validTestSpec()
	require.NoError(t, spec2.Seal())

	assert.Equal(spec1.ConfigDigest, spec2.ConfigDigest)

	d1, err := spec1.ComputeDigest()
	require.NoError(t, err)
	d2, err := spec2.ComputeDigest()
	require.NoError(t, err)
	assert.Equal(d1, d2)
}

func TestExecutionSpec_TamperDetection(t *testing.T) {
	r := require.New(t)

	// Tampered config after seal
	spec := validTestSpec()
	r.NoError(spec.Seal())
	spec.Config["difficulty"] = 99
	r.ErrorIs(spec.Validate(), ErrInvalidExecutionSpec)

	// Tampered config digest
	spec = validTestSpec()
	r.NoError(spec.Seal())
	spec.ConfigDigest = strings.Repeat("0", 64)
	r.ErrorIs(spec.Validate(), ErrInvalidExecutionSpec)

	// Tampered seed after seal changes spec digest
	spec = validTestSpec()
	r.NoError(spec.Seal())
	d1, err := spec.ComputeDigest()
	r.NoError(err)

	spec.Seed = 999
	d2, err := spec.ComputeDigest()
	r.NoError(err)
	r.NotEqual(d1, d2)
}

func TestExecutionSpec_ValidationRules(t *testing.T) {
	r := require.New(t)

	// Missing protocol
	s := validTestSpec()
	r.NoError(s.Seal())
	s.ProtocolVersion = ""
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Invalid protocol
	s = validTestSpec()
	s.ProtocolVersion = "agentrix-engine/1"
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Missing runID
	s = validTestSpec()
	r.NoError(s.Seal())
	s.RunID = ""
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Missing matchID
	s = validTestSpec()
	r.NoError(s.Seal())
	s.MatchID = ""
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Bad engine digest
	s = validTestSpec()
	r.NoError(s.Seal())
	s.EngineDigest = "short"
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Bad game digest
	s = validTestSpec()
	r.NoError(s.Seal())
	s.Game.GameDigest = "not-hex"
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Bad schema digests
	s = validTestSpec()
	r.NoError(s.Seal())
	s.SchemaDigests.Action = "bad"
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Invalid TickRate
	s = validTestSpec()
	r.NoError(s.Seal())
	s.TickRate = TickRate{Numerator: 0, Denominator: 1}
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// TickRate not reduced
	s = validTestSpec()
	r.NoError(s.Seal())
	s.TickRate = TickRate{Numerator: 120, Denominator: 2}
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Empty slots
	s = validTestSpec()
	r.NoError(s.Seal())
	s.Slots = nil
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Slot with empty ID
	s = validTestSpec()
	r.NoError(s.Seal())
	s.Slots[0].SlotID = ""
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Slot with bad digest
	s = validTestSpec()
	r.NoError(s.Seal())
	s.Slots[0].ArtifactDigest = "bad"
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Invalid MaxTicks
	s = validTestSpec()
	r.NoError(s.Seal())
	s.Limits.MaxTicks = 0
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Invalid MaxPlayers
	s = validTestSpec()
	r.NoError(s.Seal())
	s.Limits.MaxPlayers = 0
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Invalid MaxEntities
	s = validTestSpec()
	r.NoError(s.Seal())
	s.Limits.MaxEntities = 0
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Invalid MaxMessageBytes
	s = validTestSpec()
	r.NoError(s.Seal())
	s.Limits.MaxMessageBytes = 512
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Unsupported DeterminismTier
	s = validTestSpec()
	r.NoError(s.Seal())
	s.DeterminismTier = "unknown_tier"
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Unsupported RNG
	s = validTestSpec()
	r.NoError(s.Seal())
	s.RNGAlgorithm = "sha256_rng"
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)
}

func TestComputeFileSHA256(t *testing.T) {
	r := require.New(t)
	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "test.bin")
	r.NoError(os.WriteFile(fpath, []byte("agentrix-execution-spec-deterministic-data"), 0644))

	digest, err := ComputeFileSHA256(fpath)
	r.NoError(err)
	r.Len(digest, 64)

	// Digest should be reproducible
	digest2, err := ComputeFileSHA256(fpath)
	r.NoError(err)
	r.Equal(digest, digest2)
}
