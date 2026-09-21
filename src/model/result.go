package model

import "time"

type Result struct {
	Id           string      `json:"id,omitempty" db:"id"`
	MatchId      string      `json:"match_id,omitempty" db:"match_id"`
	MatchRunId   *string     `json:"match_run_id,omitempty" db:"match_run_id"`
	SlotId       *string     `json:"slot_id,omitempty" db:"slot_id"`
	SubmissionId string      `json:"submission_id,omitempty" db:"submission_id"`
	Score        int         `json:"score,omitempty" db:"score"`
	Rank         int         `json:"rank,omitempty" db:"rank"`
	Status       string      `json:"status,omitempty" db:"status"`
	Details      string      `json:"details,omitempty" db:"details"`
	CreatedAt    time.Time   `json:"created_at,omitempty" db:"created_at"`
	Submission   *Submission `json:"submission,omitempty"`
}
