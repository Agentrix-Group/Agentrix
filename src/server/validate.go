package server

import (
	"errors"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func validateUser(u *model.User) error {
	if u == nil {
		return errors.New("user data is required")
	}
	if u.Username == "" || u.Email == "" {
		return errors.New("username and email are required")
	}
	return nil
}

func validateParticipant(p *model.Participant) error {
	return validateUser(p)
}

func validateLoginRequest(username, password string) error {
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}
	return nil
}

func validateContest(c *model.Contest) error {
	if c == nil {
		return errors.New("contest data is required")
	}
	if c.Name == "" || c.GameId == "" {
		return errors.New("contest name and game_id are required")
	}
	if c.GameId != "starfighter" {
		return errors.New("game_id must be starfighter")
	}
	return nil
}

func validateCategory(c *model.Category) error {
	if c == nil {
		return errors.New("category data is required")
	}
	if c.Id == "" || c.Description == "" {
		return errors.New("category id and description are required")
	}
	return nil
}

func validateAgent(a *model.Agent) error {
	if a == nil {
		return errors.New("agent data is required")
	}
	if a.Name == "" || a.GameId == "" {
		return errors.New("agent name and game_id are required")
	}
	if a.GameId != "starfighter" {
		return errors.New("game_id must be starfighter")
	}
	return nil
}

func validateMatch(m *model.Match) error {
	if m == nil {
		return errors.New("match data is required")
	}
	if m.GameId == "" {
		return errors.New("game_id is required")
	}
	if m.GameId != "starfighter" {
		return errors.New("game_id must be starfighter")
	}
	return nil
}

func validateResult(r *model.Result) error {
	if r == nil {
		return errors.New("result data is required")
	}
	if r.MatchId == "" || r.SubmissionId == "" {
		return errors.New("match_id and submission_id are required")
	}
	return nil
}
