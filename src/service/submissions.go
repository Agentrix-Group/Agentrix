package service

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/google/uuid"
)

func (s *service) ListSubmissions(ctx context.Context) ([]model.Submission, error) {
	return s.repo.ListSubmissions(ctx)
}

func (s *service) ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error) {
	return s.repo.ListSubmissionsByAgent(ctx, agentId)
}

func (s *service) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	return s.repo.GetSubmission(ctx, id)
}

var (
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrAgentNotOwned      = ErrUnauthorizedAgent
	ErrInvalidBotBundle   = errors.New("invalid bot bundle")
	ErrAdmissionFailed    = errors.New("bot admission failed")
)

const (
	maxBundleBytes       = 2 << 20
	maxBotBytes          = 1 << 20
	agentProtocolVersion = "1.0"
)

func (s *service) CreateSubmissionBundle(ctx context.Context, participantId, roleId, agentId string, archive []byte) (*model.Submission, error) {
	if len(archive) == 0 || len(archive) > maxBundleBytes {
		return nil, fmt.Errorf("%w: ZIP must be between 1 byte and 2 MiB", ErrInvalidBotBundle)
	}
	agent, err := s.repo.GetAgent(ctx, agentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	if roleId != common.RoleAdmin && agent.ParticipantId != participantId {
		return nil, ErrAgentNotOwned
	}
	if agent.GameId != "starfighter" {
		return nil, fmt.Errorf("%w: agent must target starfighter", ErrInvalidBotBundle)
	}

	manifestBytes, botBytes, manifest, err := readBotBundle(archive)
	if err != nil {
		return nil, err
	}
	if manifest.Entrypoint != "bot.py" || manifest.ProtocolVersion != agentProtocolVersion {
		return nil, fmt.Errorf("%w: agentrix.json must declare bot.py and protocol 1.0", ErrInvalidBotBundle)
	}
	if s.validator == nil {
		return nil, fmt.Errorf("%w: validator unavailable", ErrAdmissionFailed)
	}
	tempDir, err := os.MkdirTemp("", "agentrix-admission-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)
	tempBot := filepath.Join(tempDir, "bot.py")
	if err := os.WriteFile(tempBot, botBytes, 0o600); err != nil {
		return nil, err
	}
	if err := s.validator.ValidateBot(ctx, tempBot); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAdmissionFailed, err)
	}

	version := 1
	if existing, listErr := s.repo.ListSubmissionsByAgent(ctx, agentId); listErr == nil {
		version = len(existing) + 1
	}
	submission := &model.Submission{
		Id: uuid.New().String(), AgentId: agentId, Version: version,
		Language: "python", Status: common.SubmissionStatusReady,
		Active: true, CreatedAt: time.Now().UTC(),
	}
	basePath := fmt.Sprintf("submissions/%s/v%d", agentId, version)
	codePath, err := s.artifacts.Save(ctx, basePath+"/bot.py", botBytes)
	if err != nil {
		return nil, err
	}
	if _, err := s.artifacts.Save(ctx, basePath+"/agentrix.json", manifestBytes); err != nil {
		return nil, err
	}
	if _, err := s.artifacts.Save(ctx, basePath+"/bundle.zip", archive); err != nil {
		return nil, err
	}
	submission.CodePath = codePath
	if err := s.repo.CreateSubmission(ctx, submission); err != nil {
		return nil, err
	}
	return submission, nil
}

func readBotBundle(archive []byte) ([]byte, []byte, model.AgentPackageManifest, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: malformed ZIP", ErrInvalidBotBundle)
	}
	files := make(map[string][]byte, 2)
	var total uint64
	for _, item := range reader.File {
		if item.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(filepath.Clean(item.Name))
		if name != item.Name || name == "." || name == ".." || filepath.IsAbs(item.Name) {
			return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: unsafe archive path", ErrInvalidBotBundle)
		}
		if name != "agentrix.json" && name != "bot.py" {
			return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: unexpected file %s", ErrInvalidBotBundle, name)
		}
		if _, duplicated := files[name]; duplicated {
			return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: duplicate file %s", ErrInvalidBotBundle, name)
		}
		total += item.UncompressedSize64
		if item.UncompressedSize64 > maxBotBytes || total > maxBundleBytes {
			return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: uncompressed content is too large", ErrInvalidBotBundle)
		}
		stream, err := item.Open()
		if err != nil {
			return nil, nil, model.AgentPackageManifest{}, err
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, maxBotBytes+1))
		closeErr := stream.Close()
		if readErr != nil {
			return nil, nil, model.AgentPackageManifest{}, readErr
		}
		if closeErr != nil {
			return nil, nil, model.AgentPackageManifest{}, closeErr
		}
		if len(content) > maxBotBytes {
			return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: file %s is too large", ErrInvalidBotBundle, name)
		}
		files[name] = content
	}
	manifestBytes, manifestOK := files["agentrix.json"]
	botBytes, botOK := files["bot.py"]
	if !manifestOK || !botOK || len(botBytes) == 0 {
		return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: agentrix.json and bot.py are required", ErrInvalidBotBundle)
	}
	var manifest model.AgentPackageManifest
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: invalid agentrix.json", ErrInvalidBotBundle)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: invalid agentrix.json", ErrInvalidBotBundle)
	}
	if manifest.Name == "" || len(manifest.Name) > 80 {
		return nil, nil, model.AgentPackageManifest{}, fmt.Errorf("%w: manifest name must contain 1 to 80 characters", ErrInvalidBotBundle)
	}
	return manifestBytes, botBytes, manifest, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}
