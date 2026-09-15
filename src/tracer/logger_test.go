package tracer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoggerFunctions(t *testing.T) {
	r := require.New(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIdKey, "test-req-123456789")
	ctx = context.WithValue(ctx, ParticipantIdKey, "part-test-1")

	// Verify WithContext produces valid logger with fields
	l := WithContext(ctx)
	r.NotNil(l)

	// Ensure calling log methods does not panic
	r.NotPanics(func() {
		Infof(ctx, "Test info message: %s", "hello")
		Debugf(ctx, "Test debug message")
		Warnf(ctx, "Test warn message")
		Errorf(ctx, "Test error message")
		Info(ctx, "Test info args", 123)
		Debug(ctx, "Test debug args")
		Warn(ctx, "Test warn args")
		Error(ctx, "Test error args")
	})
}
