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

func writeProtocolBot(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bot.py")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o755))
	return path
}

func TestBotSessionPersistsAndKeepsPayloadOpaque(t *testing.T) {
	bot := writeProtocolBot(t, `import json, sys
count = 0
init = json.loads(sys.stdin.readline())
assert init["type"] == "init"
for line in sys.stdin:
    msg = json.loads(line)
    if msg["type"] == "end":
        break
    assert msg["type"] == "perception"
    count += 1
    print(json.dumps({"type": "action", "tick": msg["tick"], "action": {
        "thrust": "FORWARD", "turn": "NONE", "shoot": False,
        "shield": False, "seen_count": count
    }}), flush=True)
`)

	sandbox := NewSandbox(time.Second).(*agentSandbox)
	session, err := sandbox.StartSession(context.Background(), "match-1", map[string]string{"p1": bot}, 7, 0)
	require.NoError(t, err)
	defer session.Close(context.Background(), "", "test")

	for tick := 0; tick < 3; tick++ {
		input := session.ExecuteTurn(context.Background(), tick, "p1", json.RawMessage(`{"tick":`+string(rune('0'+tick))+`}`))
		require.Equal(t, engine.ActionStatusValid, input.Status)
		var action map[string]interface{}
		require.NoError(t, json.Unmarshal(input.Payload, &action))
		require.Equal(t, float64(tick+1), action["seen_count"])
	}
}

func TestBotSessionTickZeroIsStrict(t *testing.T) {
	bot := writeProtocolBot(t, `import json, sys
json.loads(sys.stdin.readline())
msg = json.loads(sys.stdin.readline())
print(json.dumps({"type": "action", "tick": msg["tick"] + 1, "action": {"thrust": "OFF"}}), flush=True)
`)
	sandbox := NewSandbox(time.Second).(*agentSandbox)
	session, err := sandbox.StartSession(context.Background(), "match-tick", map[string]string{"p1": bot}, 1, 0)
	require.NoError(t, err)
	defer session.Close(context.Background(), "", "test")

	input := session.ExecuteTurn(context.Background(), 0, "p1", json.RawMessage(`{"tick":0}`))
	require.Equal(t, engine.ActionStatusInvalidOutput, input.Status)
	require.Contains(t, input.ErrorDetails, "got 1, expected 0")

	next := session.ExecuteTurn(context.Background(), 1, "p1", json.RawMessage(`{"tick":1}`))
	require.Equal(t, engine.ActionStatusDisqualified, next.Status)
	require.Equal(t, "tick_mismatch", next.ErrorDetails)
}

func TestBotSessionRequiresExplicitTickAndObjectAction(t *testing.T) {
	for name, response := range map[string]string{
		"missing tick":  `{"type":"action","action":{"thrust":"OFF"}}`,
		"scalar action": `{"type":"action","tick":0,"action":"OFF"}`,
	} {
		t.Run(name, func(t *testing.T) {
			bot := writeProtocolBot(t, `import json, sys
json.loads(sys.stdin.readline())
json.loads(sys.stdin.readline())
print('`+response+`', flush=True)
`)
			sandbox := NewSandbox(time.Second).(*agentSandbox)
			session, err := sandbox.StartSession(context.Background(), "match-invalid", map[string]string{"p1": bot}, 1, 0)
			require.NoError(t, err)
			defer session.Close(context.Background(), "", "test")

			input := session.ExecuteTurn(context.Background(), 0, "p1", json.RawMessage(`{"tick":0}`))
			require.Equal(t, engine.ActionStatusInvalidOutput, input.Status)
			require.Equal(t, "malformed or unexpected message", input.ErrorDetails)
		})
	}
}

func TestBotSessionTimeoutKillsAndDisqualifies(t *testing.T) {
	bot := writeProtocolBot(t, `import json, sys, time
json.loads(sys.stdin.readline())
for line in sys.stdin:
    msg = json.loads(line)
    if msg["type"] == "end":
        break
    time.sleep(1)
    print(json.dumps({"type": "action", "tick": msg["tick"], "action": {"thrust": "OFF"}}), flush=True)
`)
	sandbox := NewSandbox(40 * time.Millisecond).(*agentSandbox)
	session, err := sandbox.StartSession(context.Background(), "match-timeout", map[string]string{"p1": bot}, 1, 0)
	require.NoError(t, err)
	defer session.Close(context.Background(), "", "test")

	start := time.Now()
	first := session.ExecuteTurn(context.Background(), 0, "p1", json.RawMessage(`{"tick":0}`))
	require.Equal(t, engine.ActionStatusDisqualified, first.Status)
	require.Equal(t, "timeout", first.ErrorDetails)
	require.Less(t, time.Since(start), 500*time.Millisecond)

	start = time.Now()
	second := session.ExecuteTurn(context.Background(), 1, "p1", json.RawMessage(`{"tick":1}`))
	require.Equal(t, engine.ActionStatusDisqualified, second.Status)
	require.Equal(t, "timeout", second.ErrorDetails)
	require.Less(t, time.Since(start), 20*time.Millisecond)
}

func TestBotSessionRejectsNonPythonArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755))
	sandbox := NewSandbox(time.Second).(*agentSandbox)
	session, err := sandbox.StartSession(context.Background(), "match-python", map[string]string{"p1": path}, 1, 0)
	require.NoError(t, err)
	defer session.Close(context.Background(), "", "test")

	input := session.ExecuteTurn(context.Background(), 0, "p1", json.RawMessage(`{"tick":0}`))
	require.Equal(t, engine.ActionStatusCrashed, input.Status)
}

func TestBotSessionUnknownPlayerIsCrashed(t *testing.T) {
	sandbox := NewSandbox(time.Second).(*agentSandbox)
	session, err := sandbox.StartSession(context.Background(), "match-empty", map[string]string{}, 1, 0)
	require.NoError(t, err)
	input := session.ExecuteTurn(context.Background(), 0, "missing", json.RawMessage(`{"tick":0}`))
	require.Equal(t, engine.ActionStatusCrashed, input.Status)
}

func TestValidateBotChecksSyntaxAndAdmissionTick(t *testing.T) {
	sandbox := NewSandbox(time.Second).(*agentSandbox)
	valid := writeProtocolBot(t, `import json, sys
init = json.loads(sys.stdin.readline())
assert init["type"] == "init"
msg = json.loads(sys.stdin.readline())
print(json.dumps({"type":"action", "tick":msg["tick"], "action":{
    "thrust":"OFF", "turn":"NONE", "shoot":False, "shield":False
}}), flush=True)
`)
	require.NoError(t, sandbox.ValidateBot(context.Background(), valid))

	invalid := writeProtocolBot(t, "def broken(:\n")
	err := sandbox.ValidateBot(context.Background(), invalid)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid Python syntax")
}
