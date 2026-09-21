package model

import (
	"encoding/json"
	"time"
)

type AgentStatus string

const (
	AgentActive   AgentStatus = "active"
	AgentDisabled AgentStatus = "disabled"
)

func (s AgentStatus) Valid() bool { return s == AgentActive || s == AgentDisabled }

// Agent is the logical competitor project. It never holds executable code.
type Agent struct {
	ID          string
	OwnerUserID string
	OwnerName   string
	GameID      string
	Name        string
	Description string
	Status      AgentStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Submission is an immutable, versioned artifact of an Agent.
type Submission struct {
	ID             string
	AgentID        string
	Version        int
	Status         SubmissionStatus
	Runtime        string
	Entrypoint     string
	ArtifactKey    string
	ArtifactSHA256 string
	SizeBytes      int64
	Manifest       json.RawMessage
	AdmissionError string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Game struct {
	ID         string
	Name       string
	Version    string
	MinPlayers int
	MaxPlayers int
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Contest struct {
	ID            string
	GameID        string
	Name          string
	Description   string
	State         ContestState
	StartsAt      *time.Time
	EndsAt        *time.Time
	ScoringPolicy ScoringPolicy
	CreatedBy     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ContestEntry locks an exact submission of an agent into a contest.
type ContestEntry struct {
	ID              string
	ContestID       string
	GameID          string
	AgentID         string
	AgentName       string
	UserID          string
	Username        string
	SubmissionID    string
	SubmissionVer   int
	Status          EntryStatus
	StatusReason    string
	StatusChangedBy string
	StatusChangedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Match struct {
	ID             string
	ContestID      string // empty for exhibition/demo matches without contest
	GameID         string
	Mode           MatchMode
	State          MatchState
	Seed           int64
	CommittedRunID string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	FinishedAt     *time.Time
	Slots          []MatchSlot
}

type MatchSlot struct {
	ID             string
	MatchID        string
	SlotIndex      int
	AgentID        string
	SubmissionID   string
	ContestEntryID string
	DisplayName    string
	CreatedAt      time.Time
}

type ErrorClass string

const (
	ErrorInfrastructure ErrorClass = "infrastructure"
	ErrorSandbox        ErrorClass = "sandbox"
	ErrorEngine         ErrorClass = "engine"
	ErrorArtifact       ErrorClass = "artifact"
	ErrorLeaseLost      ErrorClass = "lease_lost"
	ErrorCancelled      ErrorClass = "cancelled"
	ErrorSpec           ErrorClass = "spec"
)

// Retryable reports whether a failure of this class schedules a new attempt
// automatically. Engine, artifact and spec failures are deterministic and
// would fail again.
func (c ErrorClass) Retryable() bool {
	return c == ErrorInfrastructure || c == ErrorSandbox || c == ErrorLeaseLost
}

// MatchRun is one physical execution attempt. Terminal runs are immutable.
type MatchRun struct {
	ID                string
	MatchID           string
	Attempt           int
	State             RunState
	ExecutionSpec     ExecutionSpec
	ExecutionSpecHash string
	EngineSHA256      string
	WorkerID          string
	SandboxRuntime    string
	FencingToken      int64
	CreatedAt         time.Time
	ReservedAt        *time.Time
	StartedAt         *time.Time
	HeartbeatAt       *time.Time
	FinishedAt        *time.Time
	ErrorClass        ErrorClass
	ErrorMessage      string
}

type MatchJob struct {
	ID           string
	MatchID      string
	RunID        string
	EngineSHA256 string
	State        JobState
	AvailableAt  time.Time
	ReservedBy   string
	ReservedAt   *time.Time
	LeaseUntil   *time.Time
	FencingToken int64
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Reservation is what a worker owns after reserving a job: the fenced
// identity it must present on heartbeat, commit and failure.
type Reservation struct {
	JobID        string
	RunID        string
	MatchID      string
	ContestID    string
	Attempt      int
	WorkerID     string
	FencingToken int64
	LeaseUntil   time.Time
	Spec         ExecutionSpec
}

type Outcome string

const (
	OutcomeWin          Outcome = "win"
	OutcomeLoss         Outcome = "loss"
	OutcomeDraw         Outcome = "draw"
	OutcomeDisqualified Outcome = "disqualified"
)

type Result struct {
	ID           string
	MatchRunID   string
	MatchID      string
	SlotID       string
	SubmissionID string
	Score        int
	Rank         int
	Outcome      Outcome
	Details      json.RawMessage
	CreatedAt    time.Time
}

type Replay struct {
	ID               string
	MatchRunID       string
	MatchID          string
	FormatVersion    string
	Compression      string
	StagingKey       string
	StorageKey       string
	SHA256           string
	SizeBytes        int64
	FrameCount       int
	Publication      ReplayPublication
	PublishAttempts  int
	LastPublishError string
	NextAttemptAt    time.Time
	PublishedAt      *time.Time
	CreatedAt        time.Time
}

// SlotOutcome is what the executor reports for one slot of a finished run.
type SlotOutcome struct {
	SlotID       string
	Score        int
	Rank         int
	Disqualified bool
	Details      map[string]any
}

// RunCompletion is the executor's report of a successfully simulated run.
// The server rebuilds match/submission identities from the slots; the
// worker only reports slot IDs.
type RunCompletion struct {
	Reservation       Reservation
	TerminationReason string
	FinalTick         int
	FinalStateHash    string
	Slots             []SlotOutcome
	Replay            ReplayArtifact
}

// ReplayArtifact describes a sealed replay file in staging storage.
type ReplayArtifact struct {
	ID            string
	FormatVersion string
	Compression   string
	StagingKey    string
	StorageKey    string
	SHA256        string
	SizeBytes     int64
	FrameCount    int
}

type Ranking struct {
	ContestID         string
	EntryID           string
	AgentID           string
	AgentName         string
	UserID            string
	Username          string
	Rank              int
	Points            int
	MatchesPlayed     int
	Wins              int
	Draws             int
	Losses            int
	Disqualifications int
	ScoreFor          int
	ScoreAgainst      int
}

type RankingState struct {
	ContestID         string
	DirtyVersion      int64
	ComputedVersion   int64
	AppliedRunsCount  int
	AppliedRunsDigest string
	ComputedAt        *time.Time
}

func (s RankingState) Stale() bool { return s.ComputedVersion < s.DirtyVersion }

type RankingSnapshot struct {
	ID                string
	ContestID         string
	Version           int
	Rankings          []Ranking
	RankingsSHA256    string
	AppliedRunsDigest string
	AppliedRunsCount  int
	PublishedBy       string
	PublishedAt       time.Time
}

type WorkerInfo struct {
	ID             string
	EngineSHA256   string
	EngineVersion  string
	SandboxRuntime string
	StartedAt      time.Time
	LastSeenAt     time.Time
}

type EngineArtifact struct {
	SHA256          string
	GameID          string
	GameVersion     string
	EngineVersion   string
	ProtocolVersion string
	SourceCommit    string
}
