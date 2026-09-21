package executor

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

// Adversarial tests against the real bubblewrap sandbox. They skip only when
// bubblewrap is unavailable and AGENTRIX_REQUIRE_SANDBOX is not set (CI sets
// it, so a missing sandbox fails the build).

func sandboxRuntime(t *testing.T) BotRuntime {
	t.Helper()
	rt := &BubblewrapRuntime{}
	if err := rt.Check(); err != nil {
		if os.Getenv("AGENTRIX_REQUIRE_SANDBOX") == "1" {
			t.Fatalf("sandbox required: %v", err)
		}
		t.Skipf("bubblewrap unavailable: %v", err)
	}
	return rt
}

func testLimits() model.ExecutionLimits {
	return model.ExecutionLimits{MaxTicks: 10, TurnTimeoutMs: 1500, InitTimeoutMs: 2000, WallTimeMs: 30000, MemoryMB: 128,
		CPUSeconds: 3, MaxProcesses: 8, MaxOutputKB: 64, MaxFileSizeKB: 256, MaxStderrKB: 8, MaxLineBytes: 4096}
}

// botScript wraps a payload that runs after reading init and the first
// perception, then answers with a valid action if it is still alive.
func botScript(t *testing.T, payload string) string {
	t.Helper()
	code := `import json, sys, os
init = json.loads(sys.stdin.readline())
msg = json.loads(sys.stdin.readline())
result = "ok"
try:
` + indent(payload, "    ") + `
except Exception as e:
    result = type(e).__name__ + ":" + str(e)
sys.stdout.write(json.dumps({"type": "action", "tick": msg["tick"], "action": {"result": result}}) + "\n")
sys.stdout.flush()
sys.stdin.readline()
`
	path := filepath.Join(t.TempDir(), "bot.py")
	require.NoError(t, os.WriteFile(path, []byte(code), 0o640))
	return path
}

func indent(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

type turnOutcome struct {
	input engine.PlayerActionInput
	cause string
	took  time.Duration
}

func runOneTurn(t *testing.T, rt BotRuntime, path string, limits model.ExecutionLimits) turnOutcome {
	t.Helper()
	start := time.Now()
	s, err := StartSession(context.Background(), rt, "sbx", "starfighter", 1, limits, []BotLaunch{{PlayerID: "p", CodePath: path, Runtime: "python3"}})
	require.NoError(t, err)
	defer s.Close("", "test")
	out := s.Turn(context.Background(), 0, map[string]json.RawMessage{"p": json.RawMessage(`{"tick":0}`)})
	return turnOutcome{input: out["p"], cause: s.Disqualifications()["p"], took: time.Since(start)}
}

func actionResult(t *testing.T, in engine.PlayerActionInput) string {
	t.Helper()
	require.Equal(t, engine.ActionStatusValid, in.Status, "details: %s", in.ErrorDetails)
	var action struct{ Result string }
	require.NoError(t, json.Unmarshal(in.Payload, &action))
	return action.Result
}

func TestSandbox_ForkBombIsContained(t *testing.T) {
	rt := sandboxRuntime(t)
	out := runOneTurn(t, rt, botScript(t, `
pids = []
for i in range(200):
    pids.append(os.fork())
    if pids[-1] == 0:
        import time; time.sleep(5); os._exit(0)
`), testLimits())
	require.Contains(t, actionResult(t, out.input), "BlockingIOError")
}

func TestSandbox_MemoryBombIsContained(t *testing.T) {
	rt := sandboxRuntime(t)
	out := runOneTurn(t, rt, botScript(t, `
blocks = []
while True:
    blocks.append(bytearray(8 * 1024 * 1024))
`), testLimits())
	require.Contains(t, actionResult(t, out.input), "MemoryError")
}

func TestSandbox_InfiniteLoopTimesOut(t *testing.T) {
	rt := sandboxRuntime(t)
	limits := testLimits()
	limits.CPUSeconds = 30 // the turn deadline fires first
	out := runOneTurn(t, rt, botScript(t, "while True:\n    pass\n"), limits)
	require.Equal(t, engine.ActionStatusDisqualified, out.input.Status)
	require.Equal(t, CauseTimeout, out.cause)
	require.Less(t, out.took, 10*time.Second)
}

func TestSandbox_CPUTimeLimitKills(t *testing.T) {
	rt := sandboxRuntime(t)
	limits := testLimits()
	limits.CPUSeconds, limits.InitTimeoutMs, limits.TurnTimeoutMs = 1, 8000, 8000
	out := runOneTurn(t, rt, botScript(t, "while True:\n    pass\n"), limits)
	require.Equal(t, engine.ActionStatusDisqualified, out.input.Status)
	require.Equal(t, CauseResourceLimit, out.cause)
	require.Less(t, out.took, 6*time.Second, "RLIMIT_CPU must stop the bot well before the turn deadline")
}

func TestSandbox_OutputFloodIsBounded(t *testing.T) {
	rt := sandboxRuntime(t)
	// A huge line is a protocol violation; the bot cannot exhaust memory.
	out := runOneTurn(t, rt, botScript(t, `sys.stdout.write("x" * (4 << 20)); sys.stdout.flush()`), testLimits())
	require.Equal(t, engine.ActionStatusDisqualified, out.input.Status)
	require.Equal(t, CauseProtocol, out.cause)
	// stderr floods are drained and truncated.
	out = runOneTurn(t, rt, botScript(t, `
for i in range(2000):
    sys.stderr.write("e" * 1024 + "\n")
`), testLimits())
	require.Equal(t, "ok", actionResult(t, out.input))
}

func TestSandbox_FileSizeIsBounded(t *testing.T) {
	rt := sandboxRuntime(t)
	out := runOneTurn(t, rt, botScript(t, `
with open("/tmp/big", "wb") as f:
    for i in range(64):
        f.write(b"x" * 65536)
        f.flush()
`), testLimits())
	result := actionResult(t, out.input)
	require.NotEqual(t, "ok", result, "writing 4 MiB must fail with a 256 KiB file limit")
}

func TestSandbox_NoNetworkNoHostFilesNoSecrets(t *testing.T) {
	rt := sandboxRuntime(t)
	t.Setenv("AGENTRIX_SECRET_SENTINEL", "leaked")
	home, _ := os.UserHomeDir()
	cases := map[string]string{
		"network": `import socket
s = socket.create_connection(("1.1.1.1", 80), timeout=1)`,
		"home": "open(" + quote(filepath.Join(home, ".bashrc")) + ").read()",
		"etc":  `open("/etc/passwd").read()`,
		"env": `
if os.environ.get("AGENTRIX_SECRET_SENTINEL") or os.environ.get("DB_PASSWORD") or os.environ.get("ACCESS_SECRET"):
    raise RuntimeError("secret visible")`,
		"write-bot": `open("/bot/bot.py", "a").write("x")`,
	}
	for name, payload := range cases {
		out := runOneTurn(t, rt, botScript(t, payload), testLimits())
		result := actionResult(t, out.input)
		if name == "env" {
			require.Equal(t, "ok", result, name)
			continue
		}
		require.NotEqual(t, "ok", result, "%s must fail inside the sandbox", name)
	}
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }

// Killing a bot kills its whole sandbox, including orphaned children, and
// does not affect other sessions.
func TestSandbox_KillReapsOrphansAndIsolatesSessions(t *testing.T) {
	rt := sandboxRuntime(t)
	marker := "agentrix-orphan-" + time.Now().Format("150405.000000000")
	orphan := botScript(t, `
if os.fork() == 0:
    os.setsid()
    os.execvp("python3", ["python3", "-c", "import time\nwhile True: time.sleep(0.2)", "`+marker+`"])
import time
time.sleep(0.3)
`)
	other := botScript(t, "pass")
	limits := testLimits()
	s1, err := StartSession(context.Background(), rt, "a", "starfighter", 1, limits, []BotLaunch{{PlayerID: "p", CodePath: orphan, Runtime: "python3"}})
	require.NoError(t, err)
	s2, err := StartSession(context.Background(), rt, "b", "starfighter", 1, limits, []BotLaunch{{PlayerID: "q", CodePath: other, Runtime: "python3"}})
	require.NoError(t, err)
	perception := json.RawMessage(`{"tick":0}`)
	s1.Turn(context.Background(), 0, map[string]json.RawMessage{"p": perception})
	alive, _ := exec.Command("pgrep", "-f", marker).Output()
	require.NotEmpty(t, strings.TrimSpace(string(alive)), "the orphan must exist before the sandbox is killed")
	s1.Close("", "test")
	require.Eventually(t, func() bool {
		out, _ := exec.Command("pgrep", "-f", marker).Output()
		return strings.TrimSpace(string(out)) == ""
	}, 5*time.Second, 100*time.Millisecond, "orphaned child survived the sandbox")
	out := s2.Turn(context.Background(), 0, map[string]json.RawMessage{"q": perception})
	require.Equal(t, engine.ActionStatusValid, out["q"].Status)
	s2.Close("", "test")
}

// A spawn failure is an infrastructure error, never a silent forfeit.
func TestSandbox_SpawnFailureIsInfrastructure(t *testing.T) {
	_, err := StartSession(context.Background(), failingRuntime{}, "m", "starfighter", 1, testLimits(),
		[]BotLaunch{{PlayerID: "a", CodePath: "/x", Runtime: "python3"}, {PlayerID: "b", CodePath: "/y", Runtime: "python3"}})
	require.ErrorIs(t, err, ErrSpawn)
}

type failingRuntime struct{}

func (failingRuntime) Name() string { return "failing" }
func (failingRuntime) Check() error { return nil }
func (failingRuntime) Spawn(context.Context, BotSpec) (BotProcess, error) {
	return nil, ErrSpawn
}
