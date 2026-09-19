package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/engine"
	"github.com/stretchr/testify/require"
)

func TestBotRuntime_CannotReadHostSecretsOrEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://secret:password@10.0.0.1:5432/agentrix")
	t.Setenv("AGENTRIX_SECRET_TOKEN", "super-secret-production-token-xyz")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "AKIASECRETSECRETSECRET")

	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hostSecretBot := writeProtocolBot(t, `import json, sys, os
init = json.loads(sys.stdin.readline())
msg = json.loads(sys.stdin.readline())

leaks = {
    "db": os.environ.get("DATABASE_URL"),
    "token": os.environ.get("AGENTRIX_SECRET_TOKEN"),
    "aws": os.environ.get("AWS_SECRET_ACCESS_KEY"),
}

print(json.dumps({
    "type": "action",
    "tick": msg["tick"],
    "action": {"thrust": "FORWARD", "leaks": leaks}
}), flush=True)
`)

	sandbox := NewSandbox(2 * time.Second).(*agentSandbox)
	session, err := sandbox.StartSession(ctx, "match-env-leak", map[string]string{"p1": hostSecretBot}, 1, 0)
	r.NoError(err)
	defer session.Close(ctx, "", "test")

	res := session.ExecuteTurn(ctx, 0, "p1", json.RawMessage(`{"tick":0}`))
	r.Equal(engine.ActionStatusValid, res.Status)

	var payload map[string]interface{}
	r.NoError(json.Unmarshal(res.Payload, &payload))
	leaks, ok := payload["leaks"].(map[string]interface{})
	r.True(ok, "leaks map should be present")
	r.Nil(leaks["db"], "DATABASE_URL must NOT be exposed to bot")
	r.Nil(leaks["token"], "AGENTRIX_SECRET_TOKEN must NOT be exposed to bot")
	r.Nil(leaks["aws"], "AWS_SECRET_ACCESS_KEY must NOT be exposed to bot")
}

func TestBotRuntime_CannotReadHostFilesOutsideSandbox(t *testing.T) {
	if !IsRootlessSandboxAvailable() {
		t.Skip("rootless sandbox not available on host")
	}

	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Write a secret file outside the bot directory in host tmp
	secretHostFile := filepath.Join(os.TempDir(), "agentrix_test_host_secret.txt")
	r.NoError(os.WriteFile(secretHostFile, []byte("TOP_SECRET_HOST_DATA"), 0o600))
	defer os.Remove(secretHostFile)

	spyBot := writeProtocolBot(t, `import json, sys, os
init = json.loads(sys.stdin.readline())
msg = json.loads(sys.stdin.readline())

secret_path = "`+secretHostFile+`"
can_read = False
try:
    with open(secret_path, "r") as f:
        _ = f.read()
    can_read = True
except Exception:
    can_read = False

print(json.dumps({
    "type": "action",
    "tick": msg["tick"],
    "action": {"thrust": "FORWARD", "can_read_secret": can_read}
}), flush=True)
`)

	sandbox := NewSandbox(2 * time.Second).(*agentSandbox)
	session, err := sandbox.StartSession(ctx, "match-file-spy", map[string]string{"p1": spyBot}, 1, 0)
	r.NoError(err)
	defer session.Close(ctx, "", "test")

	res := session.ExecuteTurn(ctx, 0, "p1", json.RawMessage(`{"tick":0}`))
	r.Equal(engine.ActionStatusValid, res.Status)

	var payload map[string]interface{}
	r.NoError(json.Unmarshal(res.Payload, &payload))
	r.Equal(false, payload["can_read_secret"], "bot must not be able to read files on host outside sandbox")
}

func TestBotRuntime_InfiniteLoopTerminatedCleanly(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hangBot := writeProtocolBot(t, `import json, sys
init = json.loads(sys.stdin.readline())
msg = json.loads(sys.stdin.readline())
# Infinite CPU loop
while True:
    pass
`)

	sandbox := NewSandbox(300 * time.Millisecond).(*agentSandbox)
	session, err := sandbox.StartSession(ctx, "match-infinite-loop", map[string]string{"p1": hangBot}, 1, 0)
	r.NoError(err)
	defer session.Close(ctx, "", "test")

	start := time.Now()
	res := session.ExecuteTurn(ctx, 0, "p1", json.RawMessage(`{"tick":0}`))
	elapsed := time.Since(start)

	r.Equal(engine.ActionStatusDisqualified, res.Status)
	r.Equal("timeout", res.ErrorDetails)
	r.Less(elapsed, 2*time.Second, "infinite loop must be killed within timeout without stalling worker")
}

func TestBotRuntime_FailClosedWhenNoSandboxAvailable(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	failClosed := &FailClosedRuntime{}
	r.Equal("fail-closed", failClosed.Name())
	r.True(failClosed.IsAvailable())

	botPath := filepath.Join(t.TempDir(), "dummy.py")
	r.NoError(os.WriteFile(botPath, []byte("print('hello')\n"), 0o600))

	proc, err := failClosed.Spawn(ctx, BotRuntimeConfig{
		PlayerID: "p1",
		CodePath: botPath,
	})
	r.Nil(proc)
	r.ErrorIs(err, ErrSandboxUnavailable, "production must fail closed when sandbox is unavailable")
}
