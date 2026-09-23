package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
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

const agentProtocolVersion = "1.0"

func (s *service) CreateSubmissionBundle(ctx context.Context, userId, roleId, agentId string, archive []byte) (*model.Submission, error) {
	if len(archive) == 0 || len(archive) > MaxBundleBytes {
		return nil, fmt.Errorf("%w: ZIP must be between 1 byte and %d MiB", ErrInvalidBotBundle, MaxBundleBytes>>20)
	}
	agent, err := s.repo.GetAgent(ctx, agentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	if roleId != common.RoleAdmin && agent.OwnerUserId != userId {
		return nil, ErrAgentNotOwned
	}
	if agent.GameId != "starfighter" {
		return nil, fmt.Errorf("%w: agent must target starfighter", ErrInvalidBotBundle)
	}

	bundle, err := readBotBundle(archive)
	if err != nil {
		return nil, err
	}
	manifest := bundle.Manifest
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
	// bundle-manifest.json: digest, runtime y SHA-256 por archivo. Es el
	// mismo en la admisión y en el almacenamiento definitivo.
	bundleManifest, err := json.MarshalIndent(map[string]any{
		"digest":  bundle.Digest,
		"runtime": manifest.Runtime,
		"files":   bundle.FileDigests,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	// La prueba de admisión usa la misma estructura en disco que la
	// ejecución (bundle/ + bundle-manifest.json): el sandbox monta el paquete
	// completo con el runtime que declara.
	admissionBundle := filepath.Join(tempDir, model.BotBundleDirName)
	if err := os.WriteFile(filepath.Join(tempDir, model.BotBundleManifestName), bundleManifest, 0o600); err != nil {
		return nil, err
	}
	for _, name := range bundle.SortedPaths() {
		target := filepath.Join(admissionBundle, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, bundle.Files[name], 0o600); err != nil {
			return nil, err
		}
	}
	tempBot := filepath.Join(admissionBundle, "bot.py")
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
	// El paquete se guarda desplegado en bundle/, que es la carpeta que el
	// sandbox monta; code_path apunta a su bot.py.
	basePath := fmt.Sprintf("submissions/%s/v%d", agentId, version)
	codePath := ""
	for _, name := range bundle.SortedPaths() {
		saved, err := s.artifacts.Save(ctx, basePath+"/"+model.BotBundleDirName+"/"+name, bundle.Files[name])
		if err != nil {
			return nil, err
		}
		if name == "bot.py" {
			codePath = saved
		}
	}
	if _, err := s.artifacts.Save(ctx, basePath+"/"+model.BotBundleManifestName, bundleManifest); err != nil {
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

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}
