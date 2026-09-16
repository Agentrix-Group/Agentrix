package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMatchJSONSerialization(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	match := Match{
		Id:        "match-1",
		ContestId: "contest-1",
		GameId:    "arena-basica",
		Status:    "finished",
		Seed:      12345,
		ReplayId:  "replay-1",
		Active:    true,
		CreatedAt: now,
		Results: []Result{
			{
				Id:           "res-1",
				MatchId:      "match-1",
				SubmissionId: "sub-1",
				Score:        100,
				Rank:         1,
				Status:       "winner",
			},
		},
		Submissions: []Submission{
			{
				Id:      "sub-1",
				AgentId: "agent-1",
				Version: 1,
			},
		},
	}

	data, err := json.Marshal(match)
	r.NoError(err)
	r.Contains(string(data), `"id":"match-1"`)
	r.Contains(string(data), `"status":"finished"`)
	r.Contains(string(data), `"seed":12345`)
	r.Contains(string(data), `"results":`)

	var parsed Match
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("match-1", parsed.Id)
	r.Equal(int64(12345), parsed.Seed)
	r.Len(parsed.Results, 1)
	r.Equal(100, parsed.Results[0].Score)
}
