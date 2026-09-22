package model_test

import (
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestScoringPolicy_DefaultAndValidation(t *testing.T) {
	r := require.New(t)

	policy := model.DefaultScoringPolicy()
	r.NoError(policy.Validate())
	r.Equal(3, policy.WinPoints)
	r.Equal(1, policy.DrawPoints)
	r.Equal(0, policy.LossPoints)
	r.Equal(0, policy.DisqualificationPenalty)
	r.Equal(model.ScoringModePlacement, policy.EffectiveMode())
	r.Equal([]model.TiebreakerRule{
		model.TiebreakerKills, model.TiebreakerScoreDiff, model.TiebreakerHeadToHead, model.TiebreakerWins,
	}, policy.Tiebreakers)

	// Points for result
	r.Equal(3, policy.PointsForResult("win"))
	r.Equal(1, policy.PointsForResult("draw"))
	r.Equal(0, policy.PointsForResult("loss"))
	r.Equal(0, policy.PointsForResult("disqualified"))
	r.Equal(0, policy.PointsForResult("no_contest"))

	// Custom penalty
	policy.DisqualificationPenalty = 2
	r.Equal(-2, policy.PointsForResult("disqualified"))

	// Invalid policies
	invalidPolicy1 := policy
	invalidPolicy1.WinPoints = 0
	r.Error(invalidPolicy1.Validate())

	invalidPolicy2 := policy
	invalidPolicy2.DrawPoints = -1
	r.Error(invalidPolicy2.Validate())

	invalidPolicy3 := policy
	invalidPolicy3.DisqualificationPenalty = -1
	r.Error(invalidPolicy3.Validate())

	invalidPolicy4 := policy
	invalidPolicy4.Tiebreakers = []model.TiebreakerRule{"unknown_rule"}
	r.Error(invalidPolicy4.Validate())

	invalidPolicy5 := policy
	invalidPolicy5.Mode = "lottery"
	r.Error(invalidPolicy5.Validate())
}

func TestPlacementPoints(t *testing.T) {
	r := require.New(t)
	// 5 jugadores sin empates: 8, 6, 4, 2, 0.
	for rank, want := range map[int]int{1: 8, 2: 6, 3: 4, 4: 2, 5: 0} {
		r.Equal(want, model.PlacementPoints(5, rank, 1), "rank %d", rank)
	}
	r.Equal(7, model.PlacementPoints(5, 1, 2), "two tied first share (8+6)/2")
	r.Equal(6, model.PlacementPoints(5, 1, 3), "three tied first share (8+6+4)/3")
	r.Equal(1, model.PlacementPoints(5, 4, 2), "two tied fourth share (2+0)/2")
	// 1 contra 1: gana 2, empate 1, pierde 0 -- un empate nunca vale como ganar.
	r.Equal(2, model.PlacementPoints(2, 1, 1))
	r.Equal(1, model.PlacementPoints(2, 1, 2))
	r.Equal(0, model.PlacementPoints(2, 2, 1))
	// Fuera de rango.
	r.Equal(0, model.PlacementPoints(5, 6, 1))
	r.Equal(0, model.PlacementPoints(5, 5, 2))
	r.Equal(0, model.PlacementPoints(0, 1, 1))
}

func TestScoringPolicy_StoredPolicyWithoutModeIsWinDrawLoss(t *testing.T) {
	r := require.New(t)
	var stored model.ScoringPolicy
	r.NoError(stored.Scan([]byte(`{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["score_diff","head_to_head","wins"]}`)))
	r.Equal(model.ScoringModeWinDrawLoss, stored.EffectiveMode())
	r.NoError(stored.Validate())
}

func TestScoringPolicy_JSONBScanAndValue(t *testing.T) {
	r := require.New(t)

	policy := model.DefaultScoringPolicy()
	val, err := policy.Value()
	r.NoError(err)
	r.NotEmpty(val)

	var scanned model.ScoringPolicy
	r.NoError(scanned.Scan(val))
	r.Equal(policy.WinPoints, scanned.WinPoints)
	r.Equal(policy.Tiebreakers, scanned.Tiebreakers)

	// Scanning nil should yield default policy
	var defaultScanned model.ScoringPolicy
	r.NoError(defaultScanned.Scan(nil))
	r.Equal(3, defaultScanned.WinPoints)
}
