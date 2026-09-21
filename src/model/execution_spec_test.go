package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validTestSpec() ExecutionSpec {
	return ExecutionSpec{
		ProtocolVersion: "agentrix-engine/1",
		GameID:          "starfighter",
		GameVersion:     "1.0.0",
		EngineVersion:   "0.3.0",
		EngineDigest:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Config: map[string]interface{}{
			"tick_hz":      60.0,
			"arena_width":  2000.0,
			"arena_height": 1000.0,
		},
		Seed: 42,
		Limits: ExecutionLimits{
			MaxTicks:        100,
			FixedTimestepMs: 16,
			TickHz:          60.0,
			TimeoutMs:       500,
		},
		Slots: []ExecutionSlotSpec{
			{SlotIndex: 0, SubmissionID: "sub-1", AgentID: "agent-1"},
			{SlotIndex: 1, SubmissionID: "sub-2", AgentID: "agent-2"},
		},
		ReplayVersion: "agentrix-replay/1",
	}
}

func TestExecutionSpec_SealAndValidate(t *testing.T) {
	r := require.New(t)
	spec := validTestSpec()

	r.NoError(spec.Seal())
	r.NotEmpty(spec.ConfigHash)
	r.NotEmpty(spec.SpecHash)
	r.NoError(spec.Validate())

	// Round-trip serialization
	jsonStr, err := spec.ToJSON()
	r.NoError(err)

	parsed, err := ParseExecutionSpec(jsonStr)
	r.NoError(err)
	r.Equal(spec.SpecHash, parsed.SpecHash)
	r.Equal(spec.ConfigHash, parsed.ConfigHash)
	r.Equal(spec.Seed, parsed.Seed)
	r.Equal(2, len(parsed.Slots))
}

func TestExecutionSpec_HashDeterminism(t *testing.T) {
	assert := assert.New(t)

	spec1 := validTestSpec()
	require.NoError(t, spec1.Seal())

	spec2 := validTestSpec()
	require.NoError(t, spec2.Seal())

	assert.Equal(spec1.ConfigHash, spec2.ConfigHash)
	assert.Equal(spec1.SpecHash, spec2.SpecHash)
}

func TestExecutionSpec_TamperDetection(t *testing.T) {
	r := require.New(t)

	// Tampered config
	spec := validTestSpec()
	r.NoError(spec.Seal())
	spec.Config["arena_width"] = 9999.0
	r.ErrorIs(spec.Validate(), ErrInvalidExecutionSpec)

	// Tampered spec hash
	spec = validTestSpec()
	r.NoError(spec.Seal())
	spec.SpecHash = "badhash"
	r.ErrorIs(spec.Validate(), ErrInvalidExecutionSpec)

	// Tampered seed after seal
	spec = validTestSpec()
	r.NoError(spec.Seal())
	spec.Seed = 999
	r.ErrorIs(spec.Validate(), ErrInvalidExecutionSpec)
}

func TestExecutionSpec_ValidationRules(t *testing.T) {
	r := require.New(t)

	// Missing protocol
	s := validTestSpec()
	s.ProtocolVersion = ""
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Missing game
	s = validTestSpec()
	s.GameID = ""
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Invalid MaxTicks
	s = validTestSpec()
	s.Limits.MaxTicks = 0
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Invalid TickHz
	s = validTestSpec()
	s.Limits.TickHz = -1
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Fewer than 2 slots
	s = validTestSpec()
	s.Slots = s.Slots[:1]
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Slots out of order
	s = validTestSpec()
	s.Slots[0].SlotIndex = 1
	s.Slots[1].SlotIndex = 0
	r.ErrorIs(s.Validate(), ErrInvalidExecutionSpec)

	// Empty submission ID
	s = validTestSpec()
	s.Slots[0].SubmissionID = ""
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
