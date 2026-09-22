package executor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActivePerceptionsDropsEliminatedSlots(t *testing.T) {
	r := require.New(t)
	tick3 := json.RawMessage(`{"tick":3}`)

	active, err := activePerceptions(map[string]json.RawMessage{"a": tick3, "c": tick3}, []string{"a", "b", "c"}, 3)
	r.NoError(err)
	r.Equal([]string{"a", "c"}, active, "a slot without perception was eliminated")

	_, err = activePerceptions(map[string]json.RawMessage{"a": tick3, "b": tick3}, []string{"a"}, 3)
	r.Error(err, "an eliminated slot must never receive a perception again")

	_, err = activePerceptions(map[string]json.RawMessage{"a": json.RawMessage(`{"tick":2}`)}, []string{"a"}, 3)
	r.Error(err, "a perception from another tick is an engine error")
}
