package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRankingJSONSerialization(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	ranking := Ranking{
		Id:            "rank-1",
		ContestId:     "contest-1",
		AgentId:       "agent-1",
		UserId:        "user-1",
		Score:         1250,
		MatchesPlayed: 10,
		Wins:          7,
		Losses:        2,
		Draws:         1,
		Rank:          1,
		UpdatedAt:     now,
	}

	data, err := json.Marshal(ranking)
	r.NoError(err)
	r.Contains(string(data), `"score":1250`)
	r.Contains(string(data), `"matches_played":10`)
	r.Contains(string(data), `"wins":7`)

	var parsed Ranking
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("rank-1", parsed.Id)
	r.Equal(1250, parsed.Score)
	r.Equal(7, parsed.Wins)
	r.Equal(2, parsed.Losses)
	r.Equal(1, parsed.Draws)
	r.Equal(1, parsed.Rank)
}
