package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubmissionJSONSerialization(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	sub := Submission{
		Id:        "sub-1",
		AgentId:   "agent-1",
		Version:   2,
		CodePath:  "/storage/submissions/agent-1-v2.py",
		Language:  "python",
		Status:    "ready",
		Active:    true,
		CreatedAt: now,
		Agent: &Agent{
			Id:   "agent-1",
			Name: "BotAlpha",
		},
	}

	data, err := json.Marshal(sub)
	r.NoError(err)
	r.Contains(string(data), `"version":2`)
	r.Contains(string(data), `"language":"python"`)
	r.Contains(string(data), `"status":"ready"`)

	var parsed Submission
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("sub-1", parsed.Id)
	r.Equal(2, parsed.Version)
	r.Equal("python", parsed.Language)
	r.Equal("ready", parsed.Status)
	r.NotNil(parsed.Agent)
	r.Equal("BotAlpha", parsed.Agent.Name)
}
