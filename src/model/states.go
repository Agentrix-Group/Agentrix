package model

// State machines. The database enforces the same transitions with triggers
// (00001_canonical.sql); these tables are the single Go-side definition and
// are exported for the documentation matrix and tests.

type ContestState string

const (
	ContestDraft              ContestState = "draft"
	ContestRegistrationOpen   ContestState = "registration_open"
	ContestRegistrationClosed ContestState = "registration_closed"
	ContestRunning            ContestState = "running"
	ContestFinished           ContestState = "finished"
	ContestCancelled          ContestState = "cancelled"
	ContestArchived           ContestState = "archived"
)

var ContestTransitions = map[ContestState][]ContestState{
	ContestDraft:              {ContestRegistrationOpen, ContestCancelled},
	ContestRegistrationOpen:   {ContestRegistrationClosed, ContestCancelled},
	ContestRegistrationClosed: {ContestRunning, ContestCancelled},
	ContestRunning:            {ContestFinished, ContestCancelled},
	ContestFinished:           {ContestArchived},
	ContestCancelled:          {ContestArchived},
	ContestArchived:           {},
}

func (s ContestState) Valid() bool { _, ok := ContestTransitions[s]; return ok }

func (s ContestState) CanTransitionTo(to ContestState) bool {
	return contains(ContestTransitions[s], to)
}

// Public reports whether contests in this state are visible to viewers
// without contests:manage.
func (s ContestState) Public() bool { return s.Valid() && s != ContestDraft }

type EntryStatus string

const (
	EntryEnrolled     EntryStatus = "enrolled"
	EntryWithdrawn    EntryStatus = "withdrawn"
	EntryDisqualified EntryStatus = "disqualified"
)

var EntryTransitions = map[EntryStatus][]EntryStatus{
	EntryEnrolled:     {EntryWithdrawn, EntryDisqualified},
	EntryWithdrawn:    {},
	EntryDisqualified: {},
}

func (s EntryStatus) CanTransitionTo(to EntryStatus) bool { return contains(EntryTransitions[s], to) }

type SubmissionStatus string

const (
	SubmissionValidating SubmissionStatus = "validating"
	SubmissionReady      SubmissionStatus = "ready"
	SubmissionRejected   SubmissionStatus = "rejected"
	SubmissionDisabled   SubmissionStatus = "disabled"
)

var SubmissionTransitions = map[SubmissionStatus][]SubmissionStatus{
	SubmissionValidating: {SubmissionReady, SubmissionRejected},
	SubmissionReady:      {SubmissionDisabled},
	SubmissionRejected:   {},
	SubmissionDisabled:   {},
}

func (s SubmissionStatus) CanTransitionTo(to SubmissionStatus) bool {
	return contains(SubmissionTransitions[s], to)
}

type MatchMode string

const (
	ModeCompetitive MatchMode = "competitive"
	ModeExhibition  MatchMode = "exhibition"
	ModeDemo        MatchMode = "demo"
)

func (m MatchMode) Valid() bool {
	return m == ModeCompetitive || m == ModeExhibition || m == ModeDemo
}

type MatchState string

const (
	MatchScheduled MatchState = "scheduled"
	MatchQueued    MatchState = "queued"
	MatchRunning   MatchState = "running"
	MatchFinished  MatchState = "finished"
	MatchFailed    MatchState = "failed"
	MatchCancelled MatchState = "cancelled"
)

var MatchTransitions = map[MatchState][]MatchState{
	MatchScheduled: {MatchQueued, MatchCancelled},
	MatchQueued:    {MatchRunning, MatchFailed, MatchCancelled},
	MatchRunning:   {MatchFinished, MatchFailed},
	MatchFailed:    {MatchQueued, MatchCancelled}, // a retry always creates a new run
	MatchFinished:  {},
	MatchCancelled: {},
}

func (s MatchState) Valid() bool { _, ok := MatchTransitions[s]; return ok }

func (s MatchState) CanTransitionTo(to MatchState) bool { return contains(MatchTransitions[s], to) }

type RunState string

const (
	RunCreated   RunState = "created"
	RunReserved  RunState = "reserved"
	RunRunning   RunState = "running"
	RunCommitted RunState = "committed"
	RunFailed    RunState = "failed"
	RunTimedOut  RunState = "timed_out"
	RunAborted   RunState = "aborted"
)

var RunTransitions = map[RunState][]RunState{
	RunCreated:   {RunReserved, RunAborted},
	RunReserved:  {RunRunning, RunFailed, RunTimedOut, RunAborted},
	RunRunning:   {RunCommitted, RunFailed, RunTimedOut, RunAborted},
	RunCommitted: {},
	RunFailed:    {},
	RunTimedOut:  {},
	RunAborted:   {},
}

func (s RunState) Terminal() bool { return len(RunTransitions[s]) == 0 }

func (s RunState) CanTransitionTo(to RunState) bool { return contains(RunTransitions[s], to) }

type JobState string

const (
	JobPending   JobState = "pending"
	JobReserved  JobState = "reserved"
	JobCompleted JobState = "completed"
	JobFailed    JobState = "failed"
	JobCancelled JobState = "cancelled"
)

var JobTransitions = map[JobState][]JobState{
	JobPending:   {JobReserved, JobCancelled},
	JobReserved:  {JobCompleted, JobFailed},
	JobCompleted: {},
	JobFailed:    {},
	JobCancelled: {},
}

func (s JobState) CanTransitionTo(to JobState) bool { return contains(JobTransitions[s], to) }

type ReplayPublication string

const (
	ReplayStaging       ReplayPublication = "staging"
	ReplayPublished     ReplayPublication = "published"
	ReplayPublishFailed ReplayPublication = "publish_failed"
)

func contains[T comparable](list []T, v T) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
