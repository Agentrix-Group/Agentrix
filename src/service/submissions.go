package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
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
	// bundle-manifest.json: digest, runtime y SHA-256 por archivo.
	bundleManifest, err := json.MarshalIndent(map[string]any{
		"digest":  bundle.Digest,
		"runtime": manifest.Runtime,
		"files":   bundle.FileDigests,
	}, "", "  ")
	if err != nil {
		return nil, err
	}

	version := 1
	if existing, listErr := s.repo.ListSubmissionsByAgent(ctx, agentId); listErr == nil {
		version = len(existing) + 1
	}
	// La API no ejecuta bots (ADR-0007): la prueba de admisión la hace un
	// worker (ProcessNextAdmission). Hasta entonces el bot está 'validating'.
	submission := &model.Submission{
		Id: uuid.New().String(), AgentId: agentId, Version: version,
		Language: "python", Status: common.SubmissionStatusValidating,
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

// admissionStaleAfter es cuánto espera una admisión reservada por un worker
// que no la terminó (por ejemplo, porque se cayó) antes de que otro la retome.
const admissionStaleAfter = 2 * time.Minute

// ProcessNextAdmission ejecuta la prueba de admisión de la siguiente
// submission 'validating' (ADR-0014, N4). Corre solo en el worker: la API
// no ejecuta bots (ADR-0007). Devuelve false si no había trabajo.
func (s *service) ProcessNextAdmission(ctx context.Context) (bool, error) {
	if s.validator == nil {
		return false, fmt.Errorf("%w: validator unavailable", ErrAdmissionFailed)
	}
	admissions, ok := s.repo.(repository.SubmissionAdmissionRepository)
	if !ok {
		return false, fmt.Errorf("%w: repository does not support asynchronous admission", ErrAdmissionFailed)
	}
	submission, err := admissions.ClaimNextAdmission(ctx, admissionStaleAfter)
	if err != nil || submission == nil {
		return false, err
	}
	status, detail := common.SubmissionStatusReady, ""
	if err := s.validator.ValidateBot(ctx, submission.CodePath); err != nil {
		status, detail = common.SubmissionStatusRejected, err.Error()
	}
	if err := admissions.FinishAdmission(ctx, submission.Id, status, detail); err != nil {
		return true, err
	}
	tracer.InfoEvent(ctx, tracer.ScopeAgent, "submission.admission.finished", "Admisión del bot terminada",
		tracer.String("submission_id", submission.Id), tracer.String("status", status))
	return true, nil
}
