package server

import (
	"encoding/json"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
)

// Response DTOs. They are the only shapes the API serializes; persistence
// models (password hashes, artifact keys, raw specs) never reach the wire.

type UserDTO struct {
	ID           string           `json:"id"`
	Username     string           `json:"username"`
	Email        string           `json:"email,omitempty"`
	Status       model.UserStatus `json:"status"`
	Roles        []string         `json:"roles"`
	Capabilities []string         `json:"capabilities,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
}

func userDTO(u *model.User, withEmail bool) UserDTO {
	dto := UserDTO{ID: u.ID, Username: u.Username, Status: u.Status, Roles: nonNil(u.Roles), CreatedAt: u.CreatedAt}
	if withEmail {
		dto.Email = u.Email
	}
	return dto
}

type SessionDTO struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	ExpiresIn   int64     `json:"expires_in"`
	User        UserDTO   `json:"user"`
}

func sessionDTO(s *service.Session, now time.Time) SessionDTO {
	user := userDTO(s.User, true)
	for _, c := range s.Principal.Capabilities {
		user.Capabilities = append(user.Capabilities, string(c))
	}
	return SessionDTO{AccessToken: s.AccessToken, TokenType: "Bearer", ExpiresAt: s.AccessExpiresAt,
		ExpiresIn: int64(s.AccessExpiresAt.Sub(now).Seconds()), User: user}
}

type GameDTO struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Version        string         `json:"version"`
	Description    string         `json:"description"`
	MinPlayers     int            `json:"min_players"`
	MaxPlayers     int            `json:"max_players"`
	TickRate       model.TickRate `json:"tick_rate"`
	MaxTicks       int            `json:"max_ticks"`
	Viewer         string         `json:"viewer"`
	EngineProtocol string         `json:"engine_protocol"`
	BotProtocol    string         `json:"bot_protocol"`
	ReplayFormat   string         `json:"replay_format"`
	Runtimes       []string       `json:"runtimes"`
}

func gameDTO(m *game.Module) GameDTO {
	return GameDTO{ID: m.ID, Name: m.Name, Version: m.Version, Description: m.Description, MinPlayers: m.Players.Min,
		MaxPlayers: m.Players.Max, TickRate: m.TickRate, MaxTicks: m.MaxTicks, Viewer: m.Viewer,
		EngineProtocol: m.EngineProtocol, BotProtocol: m.BotProtocol, ReplayFormat: m.ReplayFormat, Runtimes: m.Runtimes}
}

type AgentDTO struct {
	ID            string            `json:"id"`
	OwnerUserID   string            `json:"owner_user_id"`
	OwnerUsername string            `json:"owner_username"`
	GameID        string            `json:"game_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Status        model.AgentStatus `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

func agentDTO(a *model.Agent) AgentDTO {
	return AgentDTO{ID: a.ID, OwnerUserID: a.OwnerUserID, OwnerUsername: a.OwnerName, GameID: a.GameID, Name: a.Name,
		Description: a.Description, Status: a.Status, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
}

type SubmissionDTO struct {
	ID             string                 `json:"id"`
	AgentID        string                 `json:"agent_id"`
	Version        int                    `json:"version"`
	Status         model.SubmissionStatus `json:"status"`
	Runtime        string                 `json:"runtime"`
	Entrypoint     string                 `json:"entrypoint"`
	ArtifactSHA256 string                 `json:"artifact_sha256"`
	SizeBytes      int64                  `json:"size_bytes"`
	Manifest       json.RawMessage        `json:"manifest"`
	AdmissionError string                 `json:"admission_error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func submissionDTO(s *model.Submission) SubmissionDTO {
	return SubmissionDTO{ID: s.ID, AgentID: s.AgentID, Version: s.Version, Status: s.Status, Runtime: s.Runtime,
		Entrypoint: s.Entrypoint, ArtifactSHA256: s.ArtifactSHA256, SizeBytes: s.SizeBytes, Manifest: s.Manifest,
		AdmissionError: s.AdmissionError, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt}
}

type ContestDTO struct {
	ID                 string               `json:"id"`
	GameID             string               `json:"game_id"`
	Name               string               `json:"name"`
	Description        string               `json:"description"`
	State              model.ContestState   `json:"state"`
	AllowedTransitions []model.ContestState `json:"allowed_transitions"`
	StartsAt           *time.Time           `json:"starts_at"`
	EndsAt             *time.Time           `json:"ends_at"`
	ScoringPolicy      model.ScoringPolicy  `json:"scoring_policy"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

func contestDTO(c *model.Contest) ContestDTO {
	return ContestDTO{ID: c.ID, GameID: c.GameID, Name: c.Name, Description: c.Description, State: c.State,
		AllowedTransitions: nonNil(model.ContestTransitions[c.State]), StartsAt: c.StartsAt, EndsAt: c.EndsAt,
		ScoringPolicy: c.ScoringPolicy, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

type EntryDTO struct {
	ID                string            `json:"id"`
	ContestID         string            `json:"contest_id"`
	AgentID           string            `json:"agent_id"`
	AgentName         string            `json:"agent_name"`
	UserID            string            `json:"user_id"`
	Username          string            `json:"username"`
	SubmissionID      string            `json:"submission_id"`
	SubmissionVersion int               `json:"submission_version"`
	Status            model.EntryStatus `json:"status"`
	StatusReason      string            `json:"status_reason,omitempty"`
	StatusChangedAt   *time.Time        `json:"status_changed_at,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
}

func entryDTO(e *model.ContestEntry) EntryDTO {
	return EntryDTO{ID: e.ID, ContestID: e.ContestID, AgentID: e.AgentID, AgentName: e.AgentName, UserID: e.UserID,
		Username: e.Username, SubmissionID: e.SubmissionID, SubmissionVersion: e.SubmissionVer, Status: e.Status,
		StatusReason: e.StatusReason, StatusChangedAt: e.StatusChangedAt, CreatedAt: e.CreatedAt}
}

type SlotDTO struct {
	ID             string `json:"id"`
	SlotIndex      int    `json:"slot_index"`
	AgentID        string `json:"agent_id"`
	SubmissionID   string `json:"submission_id"`
	ContestEntryID string `json:"contest_entry_id,omitempty"`
	DisplayName    string `json:"display_name"`
}

type MatchDTO struct {
	ID             string           `json:"id"`
	ContestID      string           `json:"contest_id,omitempty"`
	GameID         string           `json:"game_id"`
	Mode           model.MatchMode  `json:"mode"`
	State          model.MatchState `json:"state"`
	Seed           int64            `json:"seed"`
	CommittedRunID string           `json:"committed_run_id,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	FinishedAt     *time.Time       `json:"finished_at,omitempty"`
	Slots          []SlotDTO        `json:"slots"`
}

func matchDTO(m *model.Match) MatchDTO {
	dto := MatchDTO{ID: m.ID, ContestID: m.ContestID, GameID: m.GameID, Mode: m.Mode, State: m.State, Seed: m.Seed,
		CommittedRunID: m.CommittedRunID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt, FinishedAt: m.FinishedAt,
		Slots: []SlotDTO{}}
	for _, s := range m.Slots {
		dto.Slots = append(dto.Slots, SlotDTO{ID: s.ID, SlotIndex: s.SlotIndex, AgentID: s.AgentID, SubmissionID: s.SubmissionID,
			ContestEntryID: s.ContestEntryID, DisplayName: s.DisplayName})
	}
	return dto
}

type RunDTO struct {
	ID                string           `json:"id"`
	Attempt           int              `json:"attempt"`
	State             model.RunState   `json:"state"`
	ExecutionSpecHash string           `json:"execution_spec_hash"`
	EngineSHA256      string           `json:"engine_sha256"`
	EngineVersion     string           `json:"engine_version"`
	TickRate          model.TickRate   `json:"tick_rate"`
	CreatedAt         time.Time        `json:"created_at"`
	ReservedAt        *time.Time       `json:"reserved_at,omitempty"`
	StartedAt         *time.Time       `json:"started_at,omitempty"`
	FinishedAt        *time.Time       `json:"finished_at,omitempty"`
	ErrorClass        model.ErrorClass `json:"error_class,omitempty"`
	ErrorMessage      string           `json:"error_message,omitempty"`
}

type ResultDTO struct {
	SlotID      string        `json:"slot_id"`
	SlotIndex   int           `json:"slot_index"`
	DisplayName string        `json:"display_name"`
	Score       int           `json:"score"`
	Rank        int           `json:"rank"`
	Outcome     model.Outcome `json:"outcome"`
}

type ReplayDTO struct {
	ID            string                  `json:"id"`
	MatchID       string                  `json:"match_id"`
	MatchRunID    string                  `json:"match_run_id"`
	FormatVersion string                  `json:"format_version"`
	Compression   string                  `json:"compression"`
	SHA256        string                  `json:"sha256"`
	SizeBytes     int64                   `json:"size_bytes"`
	FrameCount    int                     `json:"frame_count"`
	Publication   model.ReplayPublication `json:"publication_state"`
	PublishedAt   *time.Time              `json:"published_at,omitempty"`
	Available     bool                    `json:"available"`
}

func replayDTO(r *model.Replay) ReplayDTO {
	return ReplayDTO{ID: r.ID, MatchID: r.MatchID, MatchRunID: r.MatchRunID, FormatVersion: r.FormatVersion,
		Compression: r.Compression, SHA256: r.SHA256, SizeBytes: r.SizeBytes, FrameCount: r.FrameCount,
		Publication: r.Publication, PublishedAt: r.PublishedAt, Available: r.Publication == model.ReplayPublished}
}

type MatchDetailDTO struct {
	MatchDTO
	Runs    []RunDTO    `json:"runs"`
	Results []ResultDTO `json:"results"`
	Replay  *ReplayDTO  `json:"replay"`
}

func matchDetailDTO(d *service.MatchDetail) MatchDetailDTO {
	dto := MatchDetailDTO{MatchDTO: matchDTO(d.Match), Runs: []RunDTO{}, Results: []ResultDTO{}}
	for _, r := range d.Runs {
		dto.Runs = append(dto.Runs, RunDTO{ID: r.ID, Attempt: r.Attempt, State: r.State, ExecutionSpecHash: r.ExecutionSpecHash,
			EngineSHA256: r.EngineSHA256, EngineVersion: r.ExecutionSpec.Engine.Version, TickRate: r.ExecutionSpec.TickRate,
			CreatedAt: r.CreatedAt, ReservedAt: r.ReservedAt, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
			ErrorClass: r.ErrorClass, ErrorMessage: r.ErrorMessage})
	}
	slots := map[string]model.MatchSlot{}
	for _, s := range d.Match.Slots {
		slots[s.ID] = s
	}
	for _, r := range d.Results {
		slot := slots[r.SlotID]
		dto.Results = append(dto.Results, ResultDTO{SlotID: r.SlotID, SlotIndex: slot.SlotIndex, DisplayName: slot.DisplayName,
			Score: r.Score, Rank: r.Rank, Outcome: r.Outcome})
	}
	if d.Replay != nil {
		replay := replayDTO(d.Replay)
		dto.Replay = &replay
	}
	return dto
}

type RankingDTO struct {
	Rank              int    `json:"rank"`
	EntryID           string `json:"entry_id"`
	AgentID           string `json:"agent_id"`
	AgentName         string `json:"agent_name"`
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	Points            int    `json:"points"`
	MatchesPlayed     int    `json:"matches_played"`
	Wins              int    `json:"wins"`
	Draws             int    `json:"draws"`
	Losses            int    `json:"losses"`
	Disqualifications int    `json:"disqualifications"`
	ScoreFor          int    `json:"score_for"`
	ScoreAgainst      int    `json:"score_against"`
}

func rankingDTOs(rows []model.Ranking) []RankingDTO {
	out := make([]RankingDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, RankingDTO(r.SnapshotRow()))
	}
	return out
}

type RankingsDTO struct {
	ContestID         string       `json:"contest_id"`
	Rankings          []RankingDTO `json:"rankings"`
	Stale             bool         `json:"stale"`
	ComputedAt        *time.Time   `json:"computed_at"`
	AppliedRunsCount  int          `json:"applied_runs_count"`
	AppliedRunsDigest string       `json:"applied_runs_digest"`
}

type SnapshotDTO struct {
	ID                string       `json:"id"`
	ContestID         string       `json:"contest_id"`
	Version           int          `json:"version"`
	Rankings          []RankingDTO `json:"rankings"`
	RankingsSHA256    string       `json:"rankings_sha256"`
	AppliedRunsDigest string       `json:"applied_runs_digest"`
	AppliedRunsCount  int          `json:"applied_runs_count"`
	PublishedBy       string       `json:"published_by"`
	PublishedAt       time.Time    `json:"published_at"`
}

func snapshotDTO(s *model.RankingSnapshot) SnapshotDTO {
	return SnapshotDTO{ID: s.ID, ContestID: s.ContestID, Version: s.Version, Rankings: rankingDTOs(s.Rankings),
		RankingsSHA256: s.RankingsSHA256, AppliedRunsDigest: s.AppliedRunsDigest, AppliedRunsCount: s.AppliedRunsCount,
		PublishedBy: s.PublishedBy, PublishedAt: s.PublishedAt}
}

type listDTO[T any] struct {
	Items []T `json:"items"`
}

func list[T any, S any](items []S, convert func(*S) T) listDTO[T] {
	out := listDTO[T]{Items: make([]T, 0, len(items))}
	for i := range items {
		out.Items = append(out.Items, convert(&items[i]))
	}
	return out
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
