package replay

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CompressZstd compresses the given source file to sourcePath + ".zst" using the system zstd CLI.
func CompressZstd(ctx context.Context, sourcePath string) (string, error) {
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("source file does not exist: %w", err)
	}
	destPath := sourcePath + ".zst"
	cmd := exec.CommandContext(ctx, "zstd", "-q", "-f", "-k", sourcePath, "-o", destPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("zstd compression failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return destPath, nil
}

// DecompressZstd decompresses a .zst file into memory using the system zstd CLI.
func DecompressZstd(ctx context.Context, zstPath string) ([]byte, error) {
	if _, err := os.Stat(zstPath); err != nil {
		return nil, fmt.Errorf("compressed file does not exist: %w", err)
	}
	cmd := exec.CommandContext(ctx, "zstd", "-d", "-c", "-q", zstPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("zstd decompression failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// IsZstdAvailable returns true if the zstd utility is found in PATH.
func IsZstdAvailable() bool {
	_, err := exec.LookPath("zstd")
	return err == nil
}
