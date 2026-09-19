package connection

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/stretchr/testify/require"
)

func TestArtifactStore(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	cfg := &config.Config{
		Artifacts: config.Artifacts{
			Dir: tempDir,
		},
	}

	store, err := NewArtifactStore(ctx, cfg)
	r.NoError(err)
	r.NotNil(store)

	subpath := "bots/alpha/v1.py"
	data := []byte("print('hello bot')")

	// Exists before save
	r.False(store.Exists(subpath))

	// Save
	path, err := store.Save(ctx, subpath, data)
	r.NoError(err)
	r.NotEmpty(path)
	r.Equal(store.GetPath(subpath), path)

	// Exists after save
	r.True(store.Exists(subpath))

	// Read
	readData, err := store.Read(ctx, subpath)
	r.NoError(err)
	r.Equal(data, readData)

	// Delete
	err = store.Delete(ctx, subpath)
	r.NoError(err)
	r.False(store.Exists(subpath))

	// Delete non-existent (should not error)
	err = store.Delete(ctx, "non-existent")
	r.NoError(err)
}
