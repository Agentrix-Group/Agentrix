package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContestStateValidation(t *testing.T) {
	r := require.New(t)

	validStates := []ContestState{
		ContestStateDraft,
		ContestStatePublished,
		ContestStateRegistrationOpen,
		ContestStatePreparation,
		ContestStateInProgress,
		ContestStateFinalSelection,
		ContestStateLiveFinal,
		ContestStateFinished,
		ContestStateArchived,
		ContestStateSuspended,
		ContestStateCancelled,
	}

	for _, s := range validStates {
		r.True(s.IsValid(), "state %s should be valid", s)
	}

	r.False(ContestState("unknown").IsValid())
	r.False(ContestState("").IsValid())
	r.False(ContestState("deleted").IsValid())

	// Public check
	r.False(ContestStateDraft.IsPublic(), "draft should not be public")
	r.False(ContestState("invalid").IsPublic(), "invalid state should not be public")
	r.True(ContestStatePublished.IsPublic(), "published should be public")
	r.True(ContestStateRegistrationOpen.IsPublic(), "registration_open should be public")
	r.True(ContestStateFinished.IsPublic(), "finished should be public")
}

func TestContestJSONSerialization(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	contest := Contest{
		Id:          "contest-1",
		Name:        "Grand Tournament",
		Description: "Official season tournament",
		GameId:      "arena-basica",
		CategoryId:  "cat-strat",
		StartDate:   now,
		EndDate:     now.Add(24 * time.Hour),
		State:       ContestStateRegistrationOpen,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(contest)
	r.NoError(err)
	r.Contains(string(data), `"state":"registration_open"`)
	r.Contains(string(data), `"name":"Grand Tournament"`)

	var parsed Contest
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal(ContestStateRegistrationOpen, parsed.State)
	r.Equal("Grand Tournament", parsed.Name)
}

func TestEnrollAgentStructures(t *testing.T) {
	r := require.New(t)

	req := EnrollAgentRequest{AgentId: "agent-123"}
	data, err := json.Marshal(req)
	r.NoError(err)
	r.Contains(string(data), `"agent_id":"agent-123"`)

	resp := EnrollAgentResponse{
		HttpStatusCode: 200,
		Message:        "Enrolled successfully",
		Ranking: &Ranking{
			Id:      "rank-1",
			AgentId: "agent-123",
			Score:   1000,
		},
	}
	respData, err := json.Marshal(resp)
	r.NoError(err)
	r.Contains(string(respData), `"httpStatusCode":200`)
	r.Contains(string(respData), `"score":1000`)
}
