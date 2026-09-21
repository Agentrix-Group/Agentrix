package engine

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// The Go request and the protocol schema declare the same fields; all of
// them are required (the engine never fills defaults).
func TestInitializeMatchMatchesSchema(t *testing.T) {
	raw, err := os.ReadFile("../../protocol/engine/v1/initialize-match.schema.json")
	require.NoError(t, err)
	var schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(raw, &schema))
	encoded, err := json.Marshal(InitializeMatchRequest{})
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(encoded, &fields))
	var goFields, props []string
	for k := range fields {
		goFields = append(goFields, k)
	}
	for k := range schema.Properties {
		props = append(props, k)
	}
	sort.Strings(goFields)
	sort.Strings(props)
	sort.Strings(schema.Required)
	require.Equal(t, props, goFields)
	require.Equal(t, props, schema.Required)
}
