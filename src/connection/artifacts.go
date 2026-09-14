package connection

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/F4nk1/Agentrix/src/config"
	"github.com/F4nk1/Agentrix/src/tracer"
)

type ArtifactStore interface {
	Save(ctx context.Context, subpath string, data []byte) (string, error)
	Read(ctx context.Context, subpath string) ([]byte, error)
	Exists(subpath string) bool
	GetPath(subpath string) string
	Delete(ctx context.Context, subpath string) error
}

type artifactStore struct {
	baseDir string
}

func NewArtifactStore(ctx context.Context, cfg *config.Config) (ArtifactStore, error) {
	baseDir := cfg.Artifacts.Dir
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		tracer.Errorf(ctx, "Failed to create artifacts directory '%s': %s", baseDir, err)
		return nil, fmt.Errorf("failed to create artifacts dir: %w", err)
	}

	tracer.Debugf(ctx, "Artifact store initialized at: %s", baseDir)
	return &artifactStore{baseDir: baseDir}, nil
}

func (s *artifactStore) GetPath(subpath string) string {
	return filepath.Join(s.baseDir, subpath)
}

func (s *artifactStore) Save(ctx context.Context, subpath string, data []byte) (string, error) {
	target := s.GetPath(subpath)
	parentDir := filepath.Dir(target)

	if err := os.MkdirAll(parentDir, 0755); err != nil {
		tracer.Errorf(ctx, "Failed to create artifact directory '%s': %s", parentDir, err)
		return "", err
	}

	if err := os.WriteFile(target, data, 0644); err != nil {
		tracer.Errorf(ctx, "Failed to save artifact file '%s': %s", target, err)
		return "", err
	}

	tracer.Debugf(ctx, "Saved artifact to '%s' (%d bytes)", target, len(data))
	return target, nil
}

func (s *artifactStore) Read(ctx context.Context, subpath string) ([]byte, error) {
	target := s.GetPath(subpath)
	data, err := os.ReadFile(target)
	if err != nil {
		tracer.Warnf(ctx, "Failed to read artifact file '%s': %s", target, err)
		return nil, err
	}
	return data, nil
}

func (s *artifactStore) Exists(subpath string) bool {
	target := s.GetPath(subpath)
	_, err := os.Stat(target)
	return err == nil
}

func (s *artifactStore) Delete(ctx context.Context, subpath string) error {
	target := s.GetPath(subpath)
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		tracer.Errorf(ctx, "Failed to delete artifact '%s': %s", target, err)
		return err
	}
	return nil
}
