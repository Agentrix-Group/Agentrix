package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/engine"
	"github.com/stretchr/testify/require"
)

// writeCountingBot creates a real Python script that tracks, in its own
// process memory, how many perception messages it has received. If
// BotSession were still spawning a fresh process per tick (the old
// behavior), this counter would reset to 1 on every call; observing it
// increase across calls is the actual proof that the same process is being
// reused across ticks, not just that the wire format round-trips once.
func writeCountingBot(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "counting_bot.py")
	script := `import sys, json

seen = 0
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    msg = json.loads(line)
    t = msg.get("type")
    if t == "handshake":
        print(json.dumps({"type": "handshake_ack", "protocol_version": "1.0"}))
        sys.stdout.flush()
    elif t == "perception":
        seen += 1
        print(json.dumps({
            "type": "action",
            "tick": msg.get("tick", 0),
            "action": {"type": "REST", "seen_count": seen},
        }))
        sys.stdout.flush()
    elif t == "end":
        break
`
	require.NoError(t, os.WriteFile(path, []byte(script), 0755))
	return path
}

func writeSilentBot(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "silent_bot.py")
	// Acks the handshake but never answers a perception -- exercises the
	// per-tick timeout path against a real, still-alive process (not a
	// dead one).
	script := `import sys, json
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    msg = json.loads(line)
    if msg.get("type") == "handshake":
        print(json.dumps({"type": "handshake_ack", "protocol_version": "1.0"}))
        sys.stdout.flush()
    # perception messages are intentionally never answered.
`
	require.NoError(t, os.WriteFile(path, []byte(script), 0755))
	return path
}

func writeBadVersionBot(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad_version_bot.py")
	script := `import sys, json
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    msg = json.loads(line)
    if msg.get("type") == "handshake":
        print(json.dumps({"type": "handshake_ack", "protocol_version": "9.9"}))
        sys.stdout.flush()
`
	require.NoError(t, os.WriteFile(path, []byte(script), 0755))
	return path
}

func TestBotSession_SameProcessPersistsAcrossTicks(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	sandbox := NewSandbox(2 * time.Second).(*agentSandbox)

	botPath := writeCountingBot(t)
	session, err := sandbox.StartSession(ctx, "match-persist", map[string]string{"bot-1": botPath}, 7, 0)
	r.NoError(err)
	defer session.Close(ctx, "", "test cleanup")

	for tick := 0; tick < 5; tick++ {
		input := session.ExecuteTurn(ctx, tick, "bot-1", map[string]interface{}{"tick": tick})
		r.Equal(engine.ActionStatusValid, input.Status, "tick %d should have produced a valid action", tick)
		seenCount, ok := input.Payload["seen_count"].(float64)
		r.True(ok, "action payload should carry seen_count: %#v", input.Payload)
		r.Equal(float64(tick+1), seenCount,
			"seen_count should increase monotonically if the SAME process handled every tick -- "+
				"if it reset to 1 each time, the process would be getting respawned per tick again")
	}
}

func TestBotSession_TimeoutOnRealAliveProcessDoesNotHangOrCrash(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	sandbox := NewSandbox(50 * time.Millisecond).(*agentSandbox)

	botPath := writeSilentBot(t)
	session, err := sandbox.StartSession(ctx, "match-timeout", map[string]string{"bot-1": botPath}, 1, 0)
	r.NoError(err)
	defer session.Close(ctx, "", "test cleanup")

	start := time.Now()
	input := session.ExecuteTurn(ctx, 0, "bot-1", map[string]interface{}{"tick": 0})
	elapsed := time.Since(start)

	r.Equal(engine.ActionStatusTimeout, input.Status)
	r.Less(elapsed, 2*time.Second, "a single tick timeout must not block anywhere near that long")
}

func TestBotSession_IncompatibleProtocolVersionDisconnectsAtStart(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	sandbox := NewSandbox(500 * time.Millisecond).(*agentSandbox)

	botPath := writeBadVersionBot(t)
	session, err := sandbox.StartSession(ctx, "match-badver", map[string]string{"bot-1": botPath}, 1, 0)
	r.NoError(err)
	defer session.Close(ctx, "", "test cleanup")

	// The mismatch is detected during the handshake inside StartSession,
	// so every subsequent ExecuteTurn should report the player as
	// disconnected (crashed) without ever writing to its stdin again.
	input := session.ExecuteTurn(ctx, 0, "bot-1", map[string]interface{}{"tick": 0})
	r.Equal(engine.ActionStatusCrashed, input.Status)
}

func TestBotSession_UnknownPlayerReportsCrashed(t *testing.T) {
	ctx := context.Background()
	sandbox := NewSandbox(500 * time.Millisecond).(*agentSandbox)

	session, err := sandbox.StartSession(ctx, "match-empty", map[string]string{}, 1, 0)
	require.NoError(t, err)
	defer session.Close(ctx, "", "test cleanup")

	input := session.ExecuteTurn(ctx, 0, "nonexistent-player", map[string]interface{}{})
	require.Equal(t, engine.ActionStatusCrashed, input.Status)
}
