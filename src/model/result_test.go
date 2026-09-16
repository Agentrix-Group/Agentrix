package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResultJSONSerialization(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	res := Result{
		Id:           "res-1",
		MatchId:      "match-1",
		SubmissionId: "sub-1",
		Score:        85,
		Rank:         2,
		Status:       "completed",
		Details:      "Survived 180 ticks",
		CreatedAt:    now,
	}

	data, err := json.Marshal(res)
	r.NoError(err)
	r.Contains(string(data), `"score":85`)
	r.Contains(string(data), `"rank":2`)
	r.Contains(string(data), `"details":"Survived 180 ticks"`)

	var parsed Result
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("res-1", parsed.Id)
	r.Equal(85, parsed.Score)
	r.Equal(2, parsed.Rank)
	r.Equal("completed", parsed.Status)
}
