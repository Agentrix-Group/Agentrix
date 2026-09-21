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
	r.Len(policy.Tiebreakers, 3)

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
