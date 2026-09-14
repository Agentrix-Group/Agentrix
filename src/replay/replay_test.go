package replay

import (
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestReplaySerializeDeserialize(t *testing.T) {
	r := require.New(t)

	data := &model.ReplayData{
		GameId:   "arena-basica",
		MatchId:  "match-test-1",
		Seed:     999,
		Players:  []string{"p1", "p2"},
		MaxTicks: 50,
		Winner:   "p1",
		Scores:   map[string]int{"p1": 150, "p2": 40},
		Frames: []model.ReplayFrame{
			{
				Tick:   0,
				Events: []string{"Game started"},
				State:  map[string]interface{}{"tick": 0},
			},
			{
				Tick:   1,
				Events: []string{"p1 moved"},
				State:  map[string]interface{}{"tick": 1},
			},
		},
	}

	bytes, err := Serialize(data)
	r.NoError(err)
	r.NotEmpty(bytes)

	parsed, err := Deserialize(bytes)
	r.NoError(err)
	r.Equal("match-test-1", parsed.MatchId)
	r.Equal("p1", parsed.Winner)
	r.Equal(2, len(parsed.Frames))

	r.NoError(Validate(parsed))

	projector := NewProjector()
	projection := projector.ProjectTimeline(parsed)
	r.NotNil(projection)
	r.Equal("match-test-1", projection.MatchId)
	r.Equal(2, projection.TotalTicks)
	r.Equal(2, len(projection.Highlights))
}
