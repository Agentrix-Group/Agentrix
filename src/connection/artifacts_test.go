package connection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactKeysCannotEscapeTheStore(t *testing.T) {
	s, err := NewArtifactStore(t.TempDir())
	require.NoError(t, err)
	for _, key := range []string{"../etc/passwd", "/abs/path", "a/../../b", "submissions/../x", "", "UPPER/x", "a//b"} {
		_, err := s.Path(key)
		require.Error(t, err, key)
		_, err = s.Put(key, []byte("x"))
		require.Error(t, err, key)
	}
}

func TestPutIsAtomicAndVerifiable(t *testing.T) {
	root := t.TempDir()
	s, err := NewArtifactStore(root)
	require.NoError(t, err)
	sha, err := s.Put("submissions/sha256/x.py", []byte("print(1)\n"))
	require.NoError(t, err)
	require.Equal(t, SHA256Bytes([]byte("print(1)\n")), sha)
	require.NoError(t, s.Verify("submissions/sha256/x.py", sha))
	require.Error(t, s.Verify("submissions/sha256/x.py", SHA256Bytes([]byte("other"))))
	entries, err := os.ReadDir(filepath.Join(root, "submissions", "sha256"))
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary files are left behind")
	_, err = s.Open("submissions/sha256/missing.py")
	require.ErrorIs(t, err, ErrArtifactNotFound)
}

func TestPendingArtifactCommitAndAbort(t *testing.T) {
	s, err := NewArtifactStore(t.TempDir())
	require.NoError(t, err)
	p, err := s.Create("replays/staging/a.ndjson.gz")
	require.NoError(t, err)
	_, err = p.Write([]byte("data"))
	require.NoError(t, err)
	exists, _ := s.Exists("replays/staging/a.ndjson.gz")
	require.False(t, exists, "uncommitted artifacts are invisible")
	require.NoError(t, p.Commit())
	exists, _ = s.Exists("replays/staging/a.ndjson.gz")
	require.True(t, exists)
	require.NoError(t, s.Move("replays/staging/a.ndjson.gz", "replays/published/a.ndjson.gz"))
	require.ErrorIs(t, s.Move("replays/staging/a.ndjson.gz", "replays/published/b.ndjson.gz"), ErrArtifactNotFound)

	p2, err := s.Create("replays/staging/b.ndjson.gz")
	require.NoError(t, err)
	tmp := p2.TempPath()
	require.NoError(t, p2.Abort())
	_, err = os.Stat(tmp)
	require.True(t, os.IsNotExist(err))
}
