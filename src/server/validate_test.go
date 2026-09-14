package server

import (
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
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
	r.NoError(validateContest(&model.Contest{Name: "Cup 2026", GameId: "arena-basica"}))
}

func TestValidateGame(t *testing.T) {
	r := require.New(t)

	r.Error(validateGame(nil))
	r.Error(validateGame(&model.Game{}))
	r.NoError(validateGame(&model.Game{Name: "Arena Basica"}))
}

func TestValidateAgent(t *testing.T) {
	r := require.New(t)

	r.Error(validateAgent(nil))
	r.Error(validateAgent(&model.Agent{}))
	r.Error(validateAgent(&model.Agent{Name: "HunterBot"}))
	r.NoError(validateAgent(&model.Agent{Name: "HunterBot", GameId: "arena-basica"}))
}
