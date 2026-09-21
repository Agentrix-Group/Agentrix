package engine

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression: a short write made io.Copy stop draining stderr, deadlocking
// engines that log more than the buffer limit.
func TestLimitedBufferKeepsDrainingPastLimit(t *testing.T) {
	b := &limitedBuffer{limit: 10}
	n, err := io.Copy(b, bytes.NewReader(bytes.Repeat([]byte("x"), 1<<20)))
	require.NoError(t, err)
	require.EqualValues(t, 1<<20, n)
	require.Equal(t, 10, len(b.String()))
}
