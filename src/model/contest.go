package model

import "time"

type ContestState string

const (
	ContestStateDraft            ContestState = "draft"
	ContestStatePublished        ContestState = "published"
	ContestStateRegistrationOpen ContestState = "registration_open"
	ContestStateOpen             ContestState = "open"
	ContestStatePreparation      ContestState = "preparation"
	ContestStateInProgress       ContestState = "in_progress"
	ContestStateLive             ContestState = "live"
	ContestStateFinalSelection   ContestState = "final_selection"
	ContestStateLiveFinal        ContestState = "live_final"
	ContestStateFinished         ContestState = "finished"
	ContestStateCompleted        ContestState = "completed"
	ContestStateArchived         ContestState = "archived"
	ContestStateSuspended        ContestState = "suspended"
	ContestStateCancelled        ContestState = "cancelled"
)

func (s ContestState) IsValid() bool {
	switch s {
	case ContestStateDraft,
		ContestStatePublished,
		ContestStateRegistrationOpen,
		ContestStateOpen,
		ContestStatePreparation,
		ContestStateInProgress,
		ContestStateLive,
		ContestStateFinalSelection,
		ContestStateLiveFinal,
		ContestStateFinished,
		ContestStateCompleted,
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
	State         ContestState   `json:"state,omitempty" db:"state"`
	Active        bool           `json:"active,omitempty" db:"active"`
	ScoringPolicy *ScoringPolicy `json:"scoring_policy,omitempty" db:"scoring_policy"`
	CreatedAt     time.Time      `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at,omitempty" db:"updated_at"`
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

type ContestEntryStatus string

const (
	ContestEntryStatusEnrolled     ContestEntryStatus = "enrolled"
	ContestEntryStatusActive       ContestEntryStatus = "active"
	ContestEntryStatusDisqualified ContestEntryStatus = "disqualified"
	ContestEntryStatusWithdrawn    ContestEntryStatus = "withdrawn"
)

type ContestEntry struct {
	Id           string             `json:"id" db:"id"`
	ContestId    string             `json:"contest_id" db:"contest_id"`
	AgentId      string             `json:"agent_id" db:"agent_id"`
	UserId       string             `json:"user_id" db:"user_id"`
	SubmissionId string             `json:"submission_id" db:"submission_id"`
	Status       ContestEntryStatus `json:"status" db:"status"`
	EnrolledAt   time.Time          `json:"enrolled_at" db:"enrolled_at"`
	Agent        *Agent      `json:"agent,omitempty"`
	User         *User       `json:"user,omitempty"`
	Submission   *Submission `json:"submission,omitempty"`
}

type EnrollAgentRequest struct {
	AgentId      string `json:"agent_id"`
	SubmissionId string `json:"submission_id,omitempty"`
}

type EnrollAgentResponse struct {
	HttpStatusCode int           `json:"httpStatusCode"`
	Message        string        `json:"message"`
	Entry          *ContestEntry `json:"entry,omitempty"`
	Ranking        *Ranking      `json:"ranking,omitempty"`
}
