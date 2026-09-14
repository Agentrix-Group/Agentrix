package replay

import (
	"github.com/F4nk1/Agentrix/src/model"
)

type Projector interface {
	ProjectTimeline(data *model.ReplayData) *TimelineProjection
}

type TimelineProjection struct {
	MatchId     string              `json:"match_id"`
	GameId      string              `json:"game_id"`
	TotalTicks  int                 `json:"total_ticks"`
	Winner      string              `json:"winner"`
	FinalScores map[string]int      `json:"final_scores"`
	Players     []string            `json:"players"`
	Highlights  []string            `json:"highlights"`
	Frames      []model.ReplayFrame `json:"frames"`
}

type replayProjector struct{}

func NewProjector() Projector {
	return &replayProjector{}
}

func (p *replayProjector) ProjectTimeline(data *model.ReplayData) *TimelineProjection {
	if data == nil {
		return nil
	}

	highlights := make([]string, 0)
	for _, frame := range data.Frames {
		for _, ev := range frame.Events {
			if len(ev) > 0 {
				highlights = append(highlights, ev)
			}
		}
	}

	return &TimelineProjection{
		MatchId:     data.MatchId,
		GameId:      data.GameId,
		TotalTicks:  len(data.Frames),
		Winner:      data.Winner,
		FinalScores: data.Scores,
		Players:     data.Players,
		Highlights:  highlights,
		Frames:      data.Frames,
	}
}
