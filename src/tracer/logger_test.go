package tracer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestConsoleEventIsCompactAndReadable(t *testing.T) {
	buffer := installTestLogger(t, Config{Level: "info", Format: "console", Color: "never"})
	ctx := BeginRequest(context.Background(), "12345678-1234-1234-1234-123456789abc")
	ctx = WithActorID(ctx, "actor-123456789")

	InfoEvent(ctx, ScopeMatch, "match.started", "Partida iniciada",
		String("game", "arena-basica"),
		Err(errors.New("internal detail must stay hidden")),
	)

	output := buffer.String()
	require.Equal(t, 1, strings.Count(output, "\n"))
	require.Contains(t, output, "INFO")
	require.Contains(t, output, "MATCH")
	require.Contains(t, output, "Partida iniciada")
	require.Contains(t, output, "game=arena-basica")
	require.Contains(t, output, "request_id=12345678")
	require.Contains(t, output, "actor_id=actor-12")
	require.NotContains(t, output, "internal detail")
	require.NotContains(t, output, "{")
	require.NotContains(t, output, "logger_test.go")
	require.NotContains(t, output, "goroutine")
}

func TestDebugConsoleShowsDiagnosticDetailWithoutStackTrace(t *testing.T) {
	buffer := installTestLogger(t, Config{Level: "debug", Format: "console", Color: "never"})

	WarnEvent(context.Background(), ScopeDatabase, "database.unavailable", "Sin conexión",
		Err(errors.New("connection refused\nprivate stack line")),
	)

	output := buffer.String()
	require.Equal(t, 1, strings.Count(output, "\n"))
	require.Contains(t, output, `error="connection refused\\nprivate stack line"`)
	require.NotContains(t, output, "logger_test.go")
}

func TestConsoleColorCanBeForced(t *testing.T) {
	buffer := installTestLogger(t, Config{Level: "info", Format: "console", Color: "always"})

	InfoEvent(context.Background(), ScopeSystem, "system.ready", "Agentrix disponible")

	require.Contains(t, buffer.String(), "\x1b[90m")
	require.Contains(t, buffer.String(), "\x1b[34m")
}

func TestJSONEventKeepsStructuredFieldsAndFullCorrelation(t *testing.T) {
	buffer := installTestLogger(t, Config{Level: "info", Format: "json", Color: "never"})
	requestID := "12345678-1234-1234-1234-123456789abc"
	matchID := "87654321-4321-4321-4321-cba987654321"
	jobID := "abcdef12-4321-4321-4321-cba987654321"
	ctx := BeginRequest(context.Background(), requestID)
	ctx = WithActorID(ctx, "account-123456789")
	ctx = WithJobID(ctx, jobID)
	ctx = WithMatchID(ctx, matchID)
	ctx = WithAttempt(ctx, 2)

	ErrorEvent(ctx, ScopeReplay, "replay.save.failed", "No se pudo guardar el replay",
		Origin(OriginInfrastructure), Int("ticks", 42), Err(errors.New("disk unavailable")))

	var event map[string]any
	require.NoError(t, json.Unmarshal(buffer.Bytes(), &event))
	require.Equal(t, "error", event["level"])
	require.Equal(t, "replay", event["scope"])
	require.Equal(t, "replay.save.failed", event["event"])
	require.Equal(t, "infrastructure", event["origin"])
	require.Equal(t, requestID, event["request_id"])
	require.Equal(t, "account-123456789", event["actor_id"])
	require.Equal(t, jobID, event["job_id"])
	require.Equal(t, matchID, event["match_id"])
	require.Equal(t, float64(2), event["attempt"])
	require.Equal(t, "disk unavailable", event["error"])
	require.Equal(t, float64(42), event["ticks"])
	require.NotContains(t, buffer.String(), "caller")
	require.NotContains(t, buffer.String(), "stack")
}

func TestRequestLifecycleEmitsOneTerminalLine(t *testing.T) {
	buffer := installTestLogger(t, Config{Level: "info", Format: "console", Color: "never"})
	ctx := BeginRequest(context.Background(), "12345678-1234-1234-1234-123456789abc")

	FailRequest(ctx, ScopeDatabase, "contests.list.failed", "No se pudieron consultar los concursos",
		Err(errors.New("driver detail")))
	require.Empty(t, buffer.String())

	CompleteRequest(ctx, "GET", "/api/v1/contests", 500, 18*time.Millisecond)

	output := buffer.String()
	require.Equal(t, 1, strings.Count(output, "\n"))
	require.Contains(t, output, "No se pudieron consultar los concursos")
	require.Contains(t, output, "method=GET")
	require.Contains(t, output, "path=/api/v1/contests")
	require.Contains(t, output, "status=500")
	require.Contains(t, output, "elapsed=18ms")
	require.Contains(t, output, "origin=infrastructure")
	require.NotContains(t, output, "driver detail")
}

func TestRoutineReadsAndExpectedClientErrorsStayQuietAtInfo(t *testing.T) {
	buffer := installTestLogger(t, Config{Level: "info", Format: "console", Color: "never"})

	CompleteRequest(context.Background(), "GET", "/api/v1/contests", 200, time.Millisecond)
	CompleteRequest(context.Background(), "POST", "/api/v1/login", 401, time.Millisecond)
	require.Empty(t, buffer.String())

	CompleteRequest(context.Background(), "POST", "/api/v1/submissions", 201, 12*time.Millisecond)
	require.Contains(t, buffer.String(), "Solicitud completada")
}

func TestInvalidConfigurationUsesDocumentedDefaults(t *testing.T) {
	state, warnings := buildState(Config{Level: "verbose", Format: "pretty", Color: "rainbow"}, zapcore.AddSync(&bytes.Buffer{}))

	require.Equal(t, zapcore.InfoLevel, state.level)
	require.Equal(t, "console", state.format)
	require.Len(t, warnings, 3)
}

func TestDiagnosticErrorsRedactSensitiveValues(t *testing.T) {
	field := Err(errors.New("user@example.com at /home/agent/private.py with postgresql://user:secret@db/app"))

	require.Equal(t, "error", field.key)
	require.Equal(t, "[redacted-email] at [redacted-path] with postgresql://user:[redacted]@db/app", field.value)
	require.True(t, field.detailOnly)
}

func TestStructuredFieldsRejectSensitiveValues(t *testing.T) {
	buffer := installTestLogger(t, Config{Level: "info", Format: "json", Color: "never"})

	InfoEvent(context.Background(), ScopeAuth, "security.test", "Evento seguro",
		String("email", "user@example.com"),
		String("token", "secret-token"),
		String("artifact_path", "/home/agent/private.py"),
		String("path", "/api/v1/contests/{id}"),
	)

	var event map[string]any
	require.NoError(t, json.Unmarshal(buffer.Bytes(), &event))
	require.Equal(t, "[redacted]", event["email"])
	require.Equal(t, "[redacted]", event["token"])
	require.Equal(t, "[redacted-path]", event["artifact_path"])
	require.Equal(t, "/api/v1/contests/{id}", event["path"])
}

func installTestLogger(t *testing.T, config Config) *bytes.Buffer {
	t.Helper()
	buffer := &bytes.Buffer{}
	state, warnings := buildState(config, zapcore.AddSync(buffer))
	require.Empty(t, warnings)

	global.Lock()
	previous := global.state
	global.state = state
	global.Unlock()
	t.Cleanup(func() {
		global.Lock()
		global.state = previous
		global.Unlock()
	})
	return buffer
}
