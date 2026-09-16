package replay

import (
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestReplayProjector(t *testing.T) {
	r := require.New(t)

	p := NewProjector()
	r.NotNil(p)

	// nil data
	r.Nil(p.ProjectTimeline(nil))

	// valid data
	data := &model.ReplayData{
		MatchId:  "match-999",
		GameId:   "arena-basica",
		Players:  []string{"bot-1", "bot-2"},
		Winner:   "bot-1",
		Scores:   map[string]int{"bot-1": 150, "bot-2": 40},
		MaxTicks: 100,
		Frames: []model.ReplayFrame{
			{
				Tick:   1,
				Events: []string{"spawn:bot-1", "spawn:bot-2"},
			},
			{
				Tick:   2,
				Events: []string{"bot-1 struck bot-2 for 25 damage"},
			},
			{
				Tick:   3,
				Events: []string{"bot-1 defeated bot-2!"},
			},
		},
	}

	proj := p.ProjectTimeline(data)
	r.NotNil(proj)
	r.Equal("match-999", proj.MatchId)
	r.Equal("arena-basica", proj.GameId)
	r.Equal(3, proj.TotalTicks)
	r.Equal("bot-1", proj.Winner)
	r.Equal(150, proj.FinalScores["bot-1"])
	r.Len(proj.Highlights, 4)
	r.Contains(proj.Highlights, "bot-1 defeated bot-2!")
}
