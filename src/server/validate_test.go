package server

import (
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestValidateParticipant(t *testing.T) {
	r := require.New(t)

	r.Error(validateParticipant(nil))
	r.Error(validateParticipant(&model.Participant{}))
	r.Error(validateParticipant(&model.Participant{Username: "alice"}))
	r.NoError(validateParticipant(&model.Participant{Username: "alice", Email: "alice@test.com"}))
}

func TestValidateContest(t *testing.T) {
	r := require.New(t)

	r.Error(validateContest(nil))
	r.Error(validateContest(&model.Contest{}))
	r.Error(validateContest(&model.Contest{Name: "Cup 2026"}))
	r.NoError(validateContest(&model.Contest{Name: "Cup 2026", GameId: "starfighter"}))
}

func TestValidateAgent(t *testing.T) {
	r := require.New(t)

	r.Error(validateAgent(nil))
	r.Error(validateAgent(&model.Agent{}))
	r.Error(validateAgent(&model.Agent{Name: "HunterBot"}))
	r.NoError(validateAgent(&model.Agent{Name: "HunterBot", GameId: "starfighter"}))
}
