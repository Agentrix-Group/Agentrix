package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateStateHash(t *testing.T) {
	r := require.New(t)

	scores1 := map[string]int{"bot-1": 10, "bot-2": 20}
	scores2 := map[string]int{"bot-1": 10, "bot-2": 20}

	h1 := calculateStateHash(12345, 1, scores1)
	h2 := calculateStateHash(12345, 1, scores2)
	r.NotEmpty(h1)
	r.Equal(h1, h2, "State hash must be deterministic")

	// Different seed
	hDiffSeed := calculateStateHash(99999, 1, scores1)
	r.NotEqual(h1, hDiffSeed)

	// Different tick
	hDiffTick := calculateStateHash(12345, 2, scores1)
	r.NotEqual(h1, hDiffTick)

	// Different score
	scoresDiff := map[string]int{"bot-1": 15, "bot-2": 20}
	hDiffScore := calculateStateHash(12345, 1, scoresDiff)
	r.NotEqual(h1, hDiffScore)
}
