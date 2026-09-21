package connection

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ArtifactStore is a filesystem store addressed by slash-separated keys
// (e.g. "submissions/sha256/<hex>.py"). Writes are atomic: content goes to a
// temporary file that is fsynced and renamed into place.
type ArtifactStore struct {
	root string
}

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*(/[A-Za-z0-9][A-Za-z0-9_.-]*)+$`)

var ErrArtifactNotFound = errors.New("artifact not found")

func NewArtifactStore(root string) (*ArtifactStore, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create artifacts dir: %w", err)
	}
	return &ArtifactStore{root: abs}, nil
}

func (s *ArtifactStore) Root() string { return s.root }

// Path maps a validated key to an absolute path inside the store.
func (s *ArtifactStore) Path(key string) (string, error) {
	if !keyPattern.MatchString(key) || strings.Contains(key, "..") || path.Clean(key) != key {
		return "", fmt.Errorf("invalid artifact key %q", key)
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}

// Put writes data atomically and returns its SHA-256.
func (s *ArtifactStore) Put(key string, data []byte) (string, error) {
	target, err := s.Path(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".tmp-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	cleanup := func(e error) (string, error) {
		_ = tmp.Close()
		if rmErr := os.Remove(tmpName); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			return "", errors.Join(e, rmErr)
		}
		return "", e
	}
	if _, err := tmp.Write(data); err != nil {
		return cleanup(err)
	}
	if err := tmp.Sync(); err != nil {
		return cleanup(err)
	}
	if err := tmp.Close(); err != nil {
		return cleanup(err)
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		return cleanup(err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return cleanup(err)
	}
	return SHA256Bytes(data), nil
}

// Create opens a writer for a new artifact at key; Commit makes it visible.
func (s *ArtifactStore) Create(key string) (*PendingArtifact, error) {
	target, err := s.Path(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".tmp-*")
	if err != nil {
		return nil, err
	}
	return &PendingArtifact{file: tmp, target: target}, nil
}

type PendingArtifact struct {
	file   *os.File
	target string
	done   bool
}

func (p *PendingArtifact) Write(b []byte) (int, error) { return p.file.Write(b) }

func (p *PendingArtifact) TempPath() string { return p.file.Name() }

// Commit fsyncs and renames the file into place.
func (p *PendingArtifact) Commit() error {
	if p.done {
		return errors.New("artifact already finalized")
	}
	p.done = true
	if err := p.file.Sync(); err != nil {
		_ = p.file.Close()
		_ = os.Remove(p.file.Name())
		return err
	}
	if err := p.file.Close(); err != nil {
		_ = os.Remove(p.file.Name())
		return err
	}
	return os.Rename(p.file.Name(), p.target)
}

// Abort discards the temporary file.
func (p *PendingArtifact) Abort() error {
	if p.done {
		return nil
	}
	p.done = true
	closeErr := p.file.Close()
	if err := os.Remove(p.file.Name()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
		return closeErr
	}
	return nil
}

func (s *ArtifactStore) Open(key string) (*os.File, error) {
	p, err := s.Path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrArtifactNotFound
	}
	return f, err
}

func (s *ArtifactStore) Read(key string) ([]byte, error) {
	f, err := s.Open(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Digest returns SHA-256 and size of a stored artifact.
func (s *ArtifactStore) Digest(key string) (string, int64, error) {
	f, err := s.Open(key)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// Verify checks that key exists with the expected digest.
func (s *ArtifactStore) Verify(key, sha string) error {
	got, _, err := s.Digest(key)
	if err != nil {
		return err
	}
	if got != sha {
		return fmt.Errorf("artifact %s digest mismatch: expected %s, got %s", key, sha, got)
	}
	return nil
}

func (s *ArtifactStore) Exists(key string) (bool, error) {
	p, err := s.Path(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Move renames src to dst (same filesystem) and fsyncs the directory.
func (s *ArtifactStore) Move(src, dst string) error {
	from, err := s.Path(src)
	if err != nil {
		return err
	}
	to, err := s.Path(dst)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		return err
	}
	if err := os.Rename(from, to); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrArtifactNotFound
		}
		return err
	}
	dir, err := os.Open(filepath.Dir(to))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (s *ArtifactStore) Delete(key string) error {
	p, err := s.Path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// StaleKeys lists keys under prefix older than cutoff (temporary files
// included), for reconciliation of orphaned artifacts.
func (s *ArtifactStore) StaleKeys(prefix string, cutoff time.Time) ([]string, error) {
	dir, err := s.Path(prefix + "/x")
	if err != nil {
		return nil, err
	}
	dir = filepath.Dir(dir)
	var out []string
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return filepath.SkipDir
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			rel, err := filepath.Rel(s.root, p)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// RemoveRaw deletes a path returned by StaleKeys, including temporary files
// whose names are not valid keys.
func (s *ArtifactStore) RemoveRaw(rel string) error {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return fmt.Errorf("refusing to remove %q", rel)
	}
	if err := os.Remove(filepath.Join(s.root, clean)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func SHA256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func SHA256File(p string) (string, int64, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
