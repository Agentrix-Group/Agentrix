package server

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
)

// ---------------------------------------------------------------------------
// Games
// ---------------------------------------------------------------------------

func (s *Server) listGames(w http.ResponseWriter, r *http.Request) {
	modules := s.svc.Games().List()
	out := listDTO[GameDTO]{Items: make([]GameDTO, 0, len(modules))}
	for _, m := range modules {
		out.Items = append(out.Items, gameDTO(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getGame(w http.ResponseWriter, r *http.Request) {
	m, ok := s.svc.Games().Get(r.PathValue("id"))
	if !ok {
		writeError(w, r, model.NotFound("game_not_found", "game not found"))
		return
	}
	writeJSON(w, http.StatusOK, gameDTO(m))
}

// ---------------------------------------------------------------------------
// Agents and submissions
// ---------------------------------------------------------------------------

type createAgentRequest struct {
	GameID      string `json:"game_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var req createAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	agent, err := s.svc.CreateAgent(r.Context(), principalFrom(r), req.GameID, req.Name, req.Description)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, agentDTO(agent))
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.svc.ListAgents(r.Context(), principalFrom(r), r.URL.Query().Get("owner"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list(agents, agentDTO))
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	agent, err := s.svc.GetAgent(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, agentDTO(agent))
}

type updateAgentRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	var req updateAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	agent, err := s.svc.UpdateAgent(r.Context(), principalFrom(r), r.PathValue("id"),
		service.AgentPatch{Name: req.Name, Description: req.Description})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, agentDTO(agent))
}

type agentStatusRequest struct {
	Status model.AgentStatus `json:"status"`
}

func (s *Server) setAgentStatus(w http.ResponseWriter, r *http.Request) {
	var req agentStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	agent, err := s.svc.SetAgentStatus(r.Context(), principalFrom(r), r.PathValue("id"), req.Status)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, agentDTO(agent))
}

func (s *Server) listSubmissions(w http.ResponseWriter, r *http.Request) {
	subs, err := s.svc.ListSubmissions(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list(subs, submissionDTO))
}

// uploadSubmission accepts multipart/form-data with a "bundle" ZIP file.
func (s *Server) uploadSubmission(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxBundleBytes+64<<10)
	if err := r.ParseMultipartForm(service.MaxBundleBytes + 64<<10); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, r, model.Validation("bundle_too_large", "bundle exceeds %d bytes", service.MaxBundleBytes))
			return
		}
		writeError(w, r, model.BadRequest("invalid_multipart", "expected multipart/form-data with a 'bundle' file"))
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	file, _, err := r.FormFile("bundle")
	if err != nil {
		writeError(w, r, model.BadRequest("bundle_required", "the 'bundle' file field is required"))
		return
	}
	defer file.Close()
	archive, err := io.ReadAll(io.LimitReader(file, service.MaxBundleBytes+1))
	if err != nil {
		writeError(w, r, err)
		return
	}
	sub, err := s.svc.UploadSubmission(r.Context(), principalFrom(r), r.PathValue("id"), archive)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, submissionDTO(sub))
}

func (s *Server) getSubmission(w http.ResponseWriter, r *http.Request) {
	sub, err := s.svc.GetSubmission(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, submissionDTO(sub))
}

func (s *Server) disableSubmission(w http.ResponseWriter, r *http.Request) {
	sub, err := s.svc.DisableSubmission(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, submissionDTO(sub))
}

// ---------------------------------------------------------------------------
// Contests and entries
// ---------------------------------------------------------------------------

type contestRequest struct {
	GameID        string               `json:"game_id"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	StartsAt      *time.Time           `json:"starts_at,omitempty"`
	EndsAt        *time.Time           `json:"ends_at,omitempty"`
	ScoringPolicy *model.ScoringPolicy `json:"scoring_policy,omitempty"`
}

func (s *Server) createContest(w http.ResponseWriter, r *http.Request) {
	var req contestRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.svc.CreateContest(r.Context(), principalFrom(r), service.ContestInput{GameID: req.GameID, Name: req.Name,
		Description: req.Description, StartsAt: req.StartsAt, EndsAt: req.EndsAt, ScoringPolicy: req.ScoringPolicy})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, contestDTO(c))
}

type contestPatchRequest struct {
	Name          *string              `json:"name,omitempty"`
	Description   *string              `json:"description,omitempty"`
	StartsAt      optionalTime         `json:"starts_at"`
	EndsAt        optionalTime         `json:"ends_at"`
	ScoringPolicy *model.ScoringPolicy `json:"scoring_policy,omitempty"`
}

// optionalTime distinguishes an absent field from an explicit null.
type optionalTime struct {
	Set   bool
	Value *time.Time
}

func (o *optionalTime) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var t time.Time
	if err := t.UnmarshalJSON(b); err != nil {
		return err
	}
	o.Value = &t
	return nil
}

func (s *Server) updateContest(w http.ResponseWriter, r *http.Request) {
	var req contestPatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	patch := service.ContestPatch{Name: req.Name, Description: req.Description, ScoringPolicy: req.ScoringPolicy}
	if req.StartsAt.Set {
		patch.StartsAt = &req.StartsAt.Value
	}
	if req.EndsAt.Set {
		patch.EndsAt = &req.EndsAt.Value
	}
	c, err := s.svc.UpdateContest(r.Context(), principalFrom(r), r.PathValue("id"), patch)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, contestDTO(c))
}

type transitionRequest struct {
	State  model.ContestState `json:"state"`
	Reason string             `json:"reason,omitempty"`
}

func (s *Server) transitionContest(w http.ResponseWriter, r *http.Request) {
	var req transitionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.svc.TransitionContest(r.Context(), principalFrom(r), r.PathValue("id"), req.State, req.Reason)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, contestDTO(c))
}

func (s *Server) listContests(w http.ResponseWriter, r *http.Request) {
	contests, err := s.svc.ListContests(r.Context(), principalFrom(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list(contests, contestDTO))
}

func (s *Server) getContest(w http.ResponseWriter, r *http.Request) {
	c, err := s.svc.GetContest(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, contestDTO(c))
}

func (s *Server) listEntries(w http.ResponseWriter, r *http.Request) {
	entries, err := s.svc.ListEntries(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list(entries, entryDTO))
}

type enrollRequest struct {
	AgentID      string `json:"agent_id"`
	SubmissionID string `json:"submission_id"`
}

func (s *Server) enroll(w http.ResponseWriter, r *http.Request) {
	var req enrollRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	entry, err := s.svc.Enroll(r.Context(), principalFrom(r), r.PathValue("id"), req.AgentID, req.SubmissionID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, entryDTO(entry))
}

type resubmitRequest struct {
	SubmissionID string `json:"submission_id"`
}

func (s *Server) resubmitEntry(w http.ResponseWriter, r *http.Request) {
	var req resubmitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	entry, err := s.svc.ResubmitEntry(r.Context(), principalFrom(r), r.PathValue("id"), r.PathValue("entryId"), req.SubmissionID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, entryDTO(entry))
}

type reasonRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) entryStatus(to model.EntryStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req reasonRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		entry, err := s.svc.ChangeEntryStatus(r.Context(), principalFrom(r), r.PathValue("id"), r.PathValue("entryId"), to, req.Reason)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, entryDTO(entry))
	}
}

func (s *Server) withdrawEntry(w http.ResponseWriter, r *http.Request) {
	s.entryStatus(model.EntryWithdrawn)(w, r)
}

func (s *Server) disqualifyEntry(w http.ResponseWriter, r *http.Request) {
	s.entryStatus(model.EntryDisqualified)(w, r)
}

// ---------------------------------------------------------------------------
// Rankings
// ---------------------------------------------------------------------------

func (s *Server) getRankings(w http.ResponseWriter, r *http.Request) {
	view, err := s.svc.GetRankings(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, RankingsDTO{ContestID: r.PathValue("id"), Rankings: rankingDTOs(view.Rankings),
		Stale: view.State.Stale(), ComputedAt: view.State.ComputedAt, AppliedRunsCount: view.State.AppliedRunsCount,
		AppliedRunsDigest: view.State.AppliedRunsDigest})
}

func (s *Server) recalculateRankings(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.RecalculateRankings(r.Context(), principalFrom(r), r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	s.getRankings(w, r)
}

func (s *Server) listSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.svc.ListSnapshots(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list(snapshots, snapshotDTO))
}

func (s *Server) publishSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.svc.PublishSnapshot(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, snapshotDTO(snapshot))
}

func (s *Server) getSnapshot(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil || version < 1 {
		writeError(w, r, model.NotFound("snapshot_not_found", "ranking snapshot not found"))
		return
	}
	snapshot, err := s.svc.GetSnapshot(r.Context(), principalFrom(r), r.PathValue("id"), version)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotDTO(snapshot))
}

// ---------------------------------------------------------------------------
// Matches and replays
// ---------------------------------------------------------------------------

func (s *Server) listMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := s.svc.ListMatches(r.Context(), principalFrom(r), r.URL.Query().Get("contest_id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list(matches, matchDTO))
}

type createMatchRequest struct {
	ContestID     string          `json:"contest_id,omitempty"`
	Mode          model.MatchMode `json:"mode"`
	EntryIDs      []string        `json:"entry_ids,omitempty"`
	SubmissionIDs []string        `json:"submission_ids,omitempty"`
	Seed          *int64          `json:"seed,omitempty"`
}

func (s *Server) createMatch(w http.ResponseWriter, r *http.Request) {
	var req createMatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	match, err := s.svc.CreateMatch(r.Context(), principalFrom(r), service.MatchInput{ContestID: req.ContestID, Mode: req.Mode,
		EntryIDs: req.EntryIDs, SubmissionIDs: req.SubmissionIDs, Seed: req.Seed})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, matchDTO(match))
}

func (s *Server) getMatch(w http.ResponseWriter, r *http.Request) {
	detail, err := s.svc.GetMatchDetail(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, matchDetailDTO(detail))
}

// scheduleRun requires an Idempotency-Key header. Replays of the same key
// return the original ticket with Idempotency-Replayed: true.
func (s *Server) scheduleRun(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key != "" && (len(key) < 8 || len(key) > 128 || strings.ContainsFunc(key, func(c rune) bool {
		return !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-_.:", c))
	})) {
		writeError(w, r, model.BadRequest("invalid_idempotency_key", "Idempotency-Key must be 8-128 characters of [A-Za-z0-9_.:-]"))
		return
	}
	ticket, replayed, err := s.svc.ScheduleRun(r.Context(), principalFrom(r), r.PathValue("id"), key)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
	writeJSON(w, http.StatusAccepted, ticket)
}

func (s *Server) cancelMatch(w http.ResponseWriter, r *http.Request) {
	var req reasonRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	match, err := s.svc.CancelMatch(r.Context(), principalFrom(r), r.PathValue("id"), req.Reason)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, matchDTO(match))
}

func (s *Server) getReplay(w http.ResponseWriter, r *http.Request) {
	replay, err := s.svc.GetReplay(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, replayDTO(replay))
}

// streamReplay serves the verified NDJSON artifact. Gzip replays are sent
// with Content-Encoding: gzip so browsers decode them transparently.
func (s *Server) streamReplay(w http.ResponseWriter, r *http.Request) {
	replay, body, err := s.svc.OpenReplay(r.Context(), principalFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer body.Close()
	h := w.Header()
	h.Set("Content-Type", "application/x-ndjson")
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("ETag", `"`+replay.SHA256+`"`)
	h.Set("X-Replay-SHA256", replay.SHA256)
	if replay.Compression == "gzip" {
		h.Set("Content-Encoding", "gzip")
	}
	h.Set("Content-Length", strconv.FormatInt(replay.SizeBytes, 10))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}
