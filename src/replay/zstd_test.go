package replay

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestZstdCompressionAndDecompression(t *testing.T) {
	r := require.New(t)
	if !IsZstdAvailable() {
		t.Skip("zstd command not available on host")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "sample.ndjson")
	ndjsonContent := `{"recordType":"header","gameId":"starfighter","matchId":"m-101"}
{"recordType":"frame","tick":0,"stateHash":"hash0","entities":[]}
{"recordType":"frame","tick":1,"stateHash":"hash1","entities":[]}
{"recordType":"footer","totalTicks":2,"winnerId":"p1"}
`
	r.NoError(os.WriteFile(sourcePath, []byte(ndjsonContent), 0o644))

	// Compress
	zstPath, err := CompressZstd(ctx, sourcePath)
	r.NoError(err)
	r.FileExists(zstPath)
	r.Equal(sourcePath+".zst", zstPath)

	// Decompress
	decompressed, err := DecompressZstd(ctx, zstPath)
	r.NoError(err)
	r.Equal(ndjsonContent, string(decompressed))

	// Non-existent source error
	_, err = CompressZstd(ctx, filepath.Join(tmpDir, "missing.ndjson"))
	r.Error(err)

	// Non-existent decompress error
	_, err = DecompressZstd(ctx, filepath.Join(tmpDir, "missing.zst"))
	r.Error(err)
}
