package model

import "time"

type ContestState string

const (
	ContestStateDraft            ContestState = "draft"
	ContestStatePublished        ContestState = "published"
	ContestStateRegistrationOpen ContestState = "registration_open"
	ContestStatePreparation      ContestState = "preparation"
	ContestStateInProgress       ContestState = "in_progress"
	ContestStateFinalSelection   ContestState = "final_selection"
	ContestStateLiveFinal        ContestState = "live_final"
	ContestStateFinished         ContestState = "finished"
	ContestStateArchived         ContestState = "archived"
	ContestStateSuspended        ContestState = "suspended"
	ContestStateCancelled        ContestState = "cancelled"
)

func (s ContestState) IsValid() bool {
	switch s {
	case ContestStateDraft,
		ContestStatePublished,
		ContestStateRegistrationOpen,
		ContestStatePreparation,
		ContestStateInProgress,
		ContestStateFinalSelection,
		ContestStateLiveFinal,
		ContestStateFinished,
		ContestStateArchived,
		ContestStateSuspended,
		ContestStateCancelled:
		return true
	}
	return false
}

func (s ContestState) IsPublic() bool {
	return s.IsValid() && s != ContestStateDraft
}

type Contest struct {
	Id          string       `json:"id,omitempty" db:"id"`
	Name        string       `json:"name,omitempty" db:"name"`
	Description string       `json:"description,omitempty" db:"description"`
	GameId      string       `json:"game_id,omitempty" db:"game_id"`
	CategoryId  string       `json:"category_id,omitempty" db:"category_id"`
	StartDate   time.Time    `json:"start_date,omitempty" db:"start_date"`
	EndDate     time.Time    `json:"end_date,omitempty" db:"end_date"`
	StartsAt    *time.Time   `json:"starts_at,omitempty" db:"starts_at"`
	EndsAt      *time.Time   `json:"ends_at,omitempty" db:"ends_at"`
	Status      string       `json:"status,omitempty" db:"status"`
	State       ContestState `json:"state,omitempty" db:"state"`
	Active      bool         `json:"active,omitempty" db:"active"`
	CreatedAt   time.Time    `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at,omitempty" db:"updated_at"`
}

type PublicContestSummary struct {
	Id          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	State       ContestState `json:"state"`
	StartsAt    *time.Time   `json:"starts_at"`
	EndsAt      *time.Time   `json:"ends_at"`
}

type PublicContestsFilter struct {
	State           ContestState
	IncludeArchived bool
}

type EnrollAgentRequest struct {
	AgentId string `json:"agent_id"`
}

type EnrollAgentResponse struct {
	HttpStatusCode int      `json:"httpStatusCode"`
	Message        string   `json:"message"`
	Ranking        *Ranking `json:"ranking,omitempty"`
}
