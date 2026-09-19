package connection

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/F4nk1/Agentrix/src/config"
)

type ArtifactStore interface {
	Save(ctx context.Context, subpath string, data []byte) (string, error)
	Read(ctx context.Context, subpath string) ([]byte, error)
	Exists(subpath string) bool
	GetPath(subpath string) string
	Delete(ctx context.Context, subpath string) error
	OpenWriter(ctx context.Context, subpath string) (io.WriteCloser, string, error)
	Move(ctx context.Context, sourceSubpath, targetSubpath string) error
}

func (s *artifactStore) Move(ctx context.Context, sourceSubpath, targetSubpath string) error {
	src := s.GetPath(sourceSubpath)
	dst := s.GetPath(targetSubpath)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

func (s *artifactStore) OpenWriter(ctx context.Context, subpath string) (io.WriteCloser, string, error) {
	target := s.GetPath(subpath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, "", err
	}
	file, err := os.Create(target)
	if err != nil {
		return nil, "", err
	}
	return file, target, nil
}

type artifactStore struct {
	baseDir string
}

func NewArtifactStore(ctx context.Context, cfg *config.Config) (ArtifactStore, error) {
	baseDir := cfg.Artifacts.Dir
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifacts dir: %w", err)
	}

	return &artifactStore{baseDir: baseDir}, nil
}

func (s *artifactStore) GetPath(subpath string) string {
	return filepath.Join(s.baseDir, subpath)
}

func (s *artifactStore) Save(ctx context.Context, subpath string, data []byte) (string, error) {
	target := s.GetPath(subpath)
	parentDir := filepath.Dir(target)

	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(target, data, 0644); err != nil {
		return "", err
	}

	return target, nil
}

func (s *artifactStore) Read(ctx context.Context, subpath string) ([]byte, error) {
	target := s.GetPath(subpath)
	data, err := os.ReadFile(target)
	if err != nil {
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
		return err
	}
	return nil
}
