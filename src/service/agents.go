package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

func (s *Service) CreateAgent(ctx context.Context, p model.Principal, gameID, name, description string) (*model.Agent, error) {
	if err := requireCap(p, model.CapAgentsCreate); err != nil {
		return nil, err
	}
	if _, err := s.module(gameID); err != nil {
		return nil, err
	}
	name, description = strings.TrimSpace(name), strings.TrimSpace(description)
	if err := validateAgentText(name, description); err != nil {
		return nil, err
	}
	now := s.now()
	agent := &model.Agent{ID: s.newID(), OwnerUserID: p.UserID, GameID: gameID, Name: name, Description: description,
		Status: model.AgentActive, CreatedAt: now, UpdatedAt: now}
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		if err := q.CreateAgent(ctx, agent); err != nil {
			if model.KindOf(err) == model.KindConflict {
				return model.Conflict("agent_name_taken", "you already have an agent named %q", name)
			}
			return err
		}
		return q.Audit(ctx, p.UserID, "agent.created", "agent", agent.ID, map[string]any{"game_id": gameID}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetAgent(ctx, agent.ID)
}

func validateAgentText(name, description string) error {
	if n := utf8.RuneCountInString(name); n < 1 || n > 64 {
		return model.Validation("invalid_agent_name", "agent name must contain 1 to 64 characters")
	}
	if utf8.RuneCountInString(description) > 2000 {
		return model.Validation("invalid_agent_description", "description must not exceed 2000 characters")
	}
	return nil
}

// GetAgent enforces horizontal authorization: an agent of another user is
// reported as not found unless the principal holds agents:read:any.
func (s *Service) GetAgent(ctx context.Context, p model.Principal, id string) (*model.Agent, error) {
	agent, err := s.store.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}
	if !p.CanFor(agent.OwnerUserID, model.CapAgentsReadOwn, model.CapAgentsReadAny) {
		return nil, model.NotFound("agent_not_found", "agent not found")
	}
	return agent, nil
}

// ListAgents lists the principal's agents, or those of ownerID (any owner
// when ownerID is "*") with agents:read:any.
func (s *Service) ListAgents(ctx context.Context, p model.Principal, ownerID string) ([]model.Agent, error) {
	switch {
	case ownerID == "" || ownerID == p.UserID:
		if err := requireCap(p, model.CapAgentsReadOwn); err != nil {
			return nil, err
		}
		return s.store.ListAgents(ctx, p.UserID)
	case ownerID == "*":
		if err := requireCap(p, model.CapAgentsReadAny); err != nil {
			return nil, err
		}
		return s.store.ListAgents(ctx, "")
	default:
		if err := requireCap(p, model.CapAgentsReadAny); err != nil {
			return nil, err
		}
		return s.store.ListAgents(ctx, ownerID)
	}
}

type AgentPatch struct {
	Name        *string
	Description *string
}

func (s *Service) UpdateAgent(ctx context.Context, p model.Principal, id string, patch AgentPatch) (*model.Agent, error) {
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		agent, err := q.LockAgent(ctx, id)
		if err != nil {
			return err
		}
		if !p.CanFor(agent.OwnerUserID, model.CapAgentsUpdateOwn, model.CapAgentsUpdateAny) {
			if p.CanFor(agent.OwnerUserID, model.CapAgentsReadOwn, model.CapAgentsReadAny) {
				return model.Forbidden("agent_not_owned", "you cannot modify this agent")
			}
			return model.NotFound("agent_not_found", "agent not found")
		}
		name, description := agent.Name, agent.Description
		if patch.Name != nil {
			name = strings.TrimSpace(*patch.Name)
		}
		if patch.Description != nil {
			description = strings.TrimSpace(*patch.Description)
		}
		if err := validateAgentText(name, description); err != nil {
			return err
		}
		now := s.now()
		if err := q.UpdateAgentDetails(ctx, id, name, description, now); err != nil {
			if model.KindOf(err) == model.KindConflict {
				return model.Conflict("agent_name_taken", "you already have an agent named %q", name)
			}
			return err
		}
		return q.Audit(ctx, p.UserID, "agent.updated", "agent", id, nil, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetAgent(ctx, id)
}

func (s *Service) SetAgentStatus(ctx context.Context, p model.Principal, id string, status model.AgentStatus) (*model.Agent, error) {
	if !status.Valid() {
		return nil, model.Validation("invalid_status", "agent status must be active or disabled")
	}
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		agent, err := q.LockAgent(ctx, id)
		if err != nil {
			return err
		}
		if !p.CanFor(agent.OwnerUserID, model.CapAgentsUpdateOwn, model.CapAgentsUpdateAny) {
			return model.NotFound("agent_not_found", "agent not found")
		}
		if agent.Status == status {
			return nil
		}
		now := s.now()
		if err := q.UpdateAgentStatus(ctx, id, status, now); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "agent.status_changed", "agent", id,
			map[string]any{"before": agent.Status, "after": status}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetAgent(ctx, id)
}

// ---------------------------------------------------------------------------
// Submissions
// ---------------------------------------------------------------------------

const (
	MaxBundleBytes       = 2 << 20
	MaxBundleEntryBytes  = 1 << 20
	MaxBundleUncompBytes = 2 << 20
	MaxBundleEntries     = 8
	BotProtocolVersion   = "1.0"
)

// BotManifest is the agentrix.json file of a bot bundle.
type BotManifest struct {
	Name            string `json:"name"`
	Entrypoint      string `json:"entrypoint"`
	ProtocolVersion string `json:"protocol_version"`
}

type bundle struct {
	manifest   BotManifest
	normalized []byte
	code       []byte
}

// parseBundle validates the ZIP strictly: exactly agentrix.json and the
// declared entrypoint, safe names, bounded sizes, no directories or links.
func parseBundle(archive []byte) (*bundle, error) {
	invalid := func(format string, args ...any) error {
		return model.Validation("invalid_bundle", format, args...)
	}
	if len(archive) == 0 || len(archive) > MaxBundleBytes {
		return nil, invalid("bundle must be a ZIP between 1 byte and %d bytes", MaxBundleBytes)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, invalid("bundle is not a valid ZIP archive")
	}
	if len(reader.File) > MaxBundleEntries {
		return nil, invalid("bundle contains too many entries")
	}
	files := map[string][]byte{}
	var total uint64
	for _, item := range reader.File {
		name := item.Name
		if item.FileInfo().IsDir() {
			return nil, invalid("bundle must not contain directories")
		}
		if !item.Mode().IsRegular() {
			return nil, invalid("bundle entry %q is not a regular file", name)
		}
		if name != path.Base(name) || strings.ContainsAny(name, `\/:`) || strings.HasPrefix(name, ".") {
			return nil, invalid("bundle entry %q must be a plain file name at the archive root", name)
		}
		if _, dup := files[name]; dup {
			return nil, invalid("bundle entry %q is duplicated", name)
		}
		total += item.UncompressedSize64
		if item.UncompressedSize64 > MaxBundleEntryBytes || total > MaxBundleUncompBytes {
			return nil, invalid("bundle content exceeds the size limit")
		}
		stream, err := item.Open()
		if err != nil {
			return nil, invalid("bundle entry %q cannot be read", name)
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, MaxBundleEntryBytes+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil {
			return nil, invalid("bundle entry %q is corrupt", name)
		}
		if len(content) > MaxBundleEntryBytes {
			return nil, invalid("bundle entry %q exceeds the size limit", name)
		}
		files[name] = content
	}
	rawManifest, ok := files["agentrix.json"]
	if !ok {
		return nil, invalid("agentrix.json is required")
	}
	var manifest BotManifest
	decoder := json.NewDecoder(bytes.NewReader(rawManifest))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, invalid("agentrix.json is not valid: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, invalid("agentrix.json has trailing data")
	}
	if n := utf8.RuneCountInString(manifest.Name); n < 1 || n > 80 {
		return nil, invalid("manifest name must contain 1 to 80 characters")
	}
	if manifest.ProtocolVersion != BotProtocolVersion {
		return nil, invalid("protocol_version must be %q", BotProtocolVersion)
	}
	if manifest.Entrypoint != "bot.py" {
		return nil, invalid("entrypoint must be bot.py")
	}
	code, ok := files[manifest.Entrypoint]
	if !ok || len(code) == 0 {
		return nil, invalid("entrypoint %s is missing or empty", manifest.Entrypoint)
	}
	if !utf8.Valid(code) {
		return nil, invalid("entrypoint must be UTF-8 source code")
	}
	if len(files) != 2 {
		return nil, invalid("bundle must contain only agentrix.json and %s", manifest.Entrypoint)
	}
	normalized, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	return &bundle{manifest: manifest, normalized: normalized, code: code}, nil
}

// SubmissionArtifactKey is the content-addressed location of a bot file.
func SubmissionArtifactKey(sha string) string { return "submissions/sha256/" + sha + ".py" }

// UploadSubmission stores the bot content-addressed (idempotent write),
// then creates the next version in "validating" inside a transaction that
// locks the agent. A worker performs the sandboxed admission test. If the
// database step fails, the stored blob is unreferenced and the reconciler
// removes it once it is old enough.
func (s *Service) UploadSubmission(ctx context.Context, p model.Principal, agentID string, archive []byte) (*model.Submission, error) {
	if err := requireCap(p, model.CapSubmissionsCreate); err != nil {
		return nil, err
	}
	agent, err := s.store.GetAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent.OwnerUserID != p.UserID {
		// submissions are authored by the owner only; "any" is not granted for writes.
		if p.Can(model.CapAgentsReadAny) {
			return nil, model.Forbidden("agent_not_owned", "only the owner can upload submissions")
		}
		return nil, model.NotFound("agent_not_found", "agent not found")
	}
	if agent.Status != model.AgentActive {
		return nil, model.Conflict("agent_disabled", "agent is disabled")
	}
	module, err := s.module(agent.GameID)
	if err != nil {
		return nil, err
	}
	b, err := parseBundle(archive)
	if err != nil {
		return nil, err
	}
	sha := connection.SHA256Bytes(b.code)
	key := SubmissionArtifactKey(sha)
	if exists, err := s.artifacts.Exists(key); err != nil {
		return nil, err
	} else if !exists {
		if _, err := s.artifacts.Put(key, b.code); err != nil {
			return nil, fmt.Errorf("store submission artifact: %w", err)
		}
	} else if err := s.artifacts.Verify(key, sha); err != nil {
		return nil, fmt.Errorf("existing submission artifact is corrupt: %w", err)
	}
	now := s.now()
	sub := &model.Submission{ID: s.newID(), AgentID: agentID, Status: model.SubmissionValidating, Runtime: module.Runtimes[0],
		Entrypoint: b.manifest.Entrypoint, ArtifactKey: key, ArtifactSHA256: sha, SizeBytes: int64(len(b.code)),
		Manifest: b.normalized, CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now}
	err = s.store.Tx(ctx, func(q *repository.Queries) error {
		locked, err := q.LockAgent(ctx, agentID)
		if err != nil {
			return err
		}
		if locked.Status != model.AgentActive {
			return model.Conflict("agent_disabled", "agent is disabled")
		}
		if sub.Version, err = q.NextSubmissionVersion(ctx, agentID); err != nil {
			return err
		}
		if err := q.CreateSubmission(ctx, sub); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "submission.uploaded", "submission", sub.ID,
			map[string]any{"agent_id": agentID, "version": sub.Version, "sha256": sha}, now)
	})
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Service) submissionOwner(ctx context.Context, sub *model.Submission) (string, error) {
	agent, err := s.store.GetAgent(ctx, sub.AgentID)
	if err != nil {
		return "", err
	}
	return agent.OwnerUserID, nil
}

func (s *Service) GetSubmission(ctx context.Context, p model.Principal, id string) (*model.Submission, error) {
	sub, err := s.store.GetSubmission(ctx, id)
	if err != nil {
		return nil, err
	}
	owner, err := s.submissionOwner(ctx, sub)
	if err != nil {
		return nil, err
	}
	if !p.CanFor(owner, model.CapSubmissionsReadOwn, model.CapSubmissionsReadAny) {
		return nil, model.NotFound("submission_not_found", "submission not found")
	}
	return sub, nil
}

func (s *Service) ListSubmissions(ctx context.Context, p model.Principal, agentID string) ([]model.Submission, error) {
	agent, err := s.store.GetAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if !p.CanFor(agent.OwnerUserID, model.CapSubmissionsReadOwn, model.CapSubmissionsReadAny) {
		return nil, model.NotFound("agent_not_found", "agent not found")
	}
	return s.store.ListSubmissions(ctx, agentID)
}

// DisableSubmission retires a ready submission. Entries that locked it keep
// it (history is immutable) but it can no longer be enrolled or scheduled.
func (s *Service) DisableSubmission(ctx context.Context, p model.Principal, id string) (*model.Submission, error) {
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		sub, err := q.LockSubmission(ctx, id)
		if err != nil {
			return err
		}
		agent, err := q.GetAgent(ctx, sub.AgentID)
		if err != nil {
			return err
		}
		if !p.CanFor(agent.OwnerUserID, model.CapAgentsUpdateOwn, model.CapAgentsUpdateAny) {
			return model.NotFound("submission_not_found", "submission not found")
		}
		if !sub.Status.CanTransitionTo(model.SubmissionDisabled) {
			return model.InvalidTransition("submission", sub.Status, model.SubmissionDisabled)
		}
		now := s.now()
		if err := q.DisableSubmission(ctx, id, now); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "submission.disabled", "submission", id, nil, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetSubmission(ctx, id)
}
