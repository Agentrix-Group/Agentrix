package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

const mlManifest = `{"digest":"","runtime":"python-ml-cpu","files":{}}`

// writeMLBundle crea un paquete v2 con runtime python-ml-cpu.
func writeMLBundle(t *testing.T, files map[string]string) string {
	t.Helper()
	codePath := writeBundle(t, files)
	version := filepath.Dir(filepath.Dir(codePath))
	require.NoError(t, os.WriteFile(filepath.Join(version, model.BotBundleManifestName), []byte(mlManifest), 0o644))
	return codePath
}

func requireMLRuntime(t *testing.T) {
	t.Helper()
	if !IsRootlessSandboxAvailable() {
		t.Skip("rootless sandbox not available on host")
	}
	if _, err := resolvePythonRuntime(model.BotRuntimePythonMLCPU, 0); err != nil {
		if os.Getenv("AGENTRIX_REQUIRE_SANDBOX") == "1" {
			t.Fatalf("python-ml-cpu runtime required: %v", err)
		}
		t.Skipf("python-ml-cpu runtime not built: %v", err)
	}
}

func TestMLRuntimeArgs_BindRuntimeLimitMemoryAndPinCPU(t *testing.T) {
	r := require.New(t)
	py := pythonRuntime{
		name:        model.BotRuntimePythonMLCPU,
		interpreter: "/opt/runtimes/python-ml-cpu/bin/python",
		bindDir:     "/opt/runtimes",
		wrapper:     []string{"prlimit", "--as=1073741824:1073741824", "taskset", "-c", "3"},
	}
	mount := botMount{hostPath: "/data/v1/bundle", isBundle: true}
	name, args := wrapCommand(py, "bwrap", bubblewrapArgs(mount, []string{"/usr"}, py))
	full := name + " " + strings.Join(args, " ")
	r.True(strings.HasPrefix(full, "prlimit --as=1073741824:1073741824 taskset -c 3 bwrap "), full)
	r.Contains(full, "--ro-bind /opt/runtimes /opt/runtimes")
	r.True(strings.HasSuffix(full, "--chdir /bot /opt/runtimes/python-ml-cpu/bin/python /bot/bot.py"), full)

	stdName, stdArgs := wrapCommand(stdlibRuntime, "bwrap", bubblewrapArgs(mount, []string{"/usr"}, stdlibRuntime))
	r.Equal("bwrap", stdName, "python-stdlib runs without wrappers, as before")
	r.NotContains(strings.Join(stdArgs, " "), "runtimes")

	pm := strings.Join(podmanArgs(mount, "ml-image", py), " ")
	r.Contains(pm, "--cpus 1 -e OMP_NUM_THREADS=1")
	r.Contains(pm, "--memory 1g")
	r.Contains(strings.Join(podmanArgs(mount, "img", stdlibRuntime), " "), "--memory 256m")
}

// runMLTurn ejecuta un turno y devuelve el resultado completo.
func runMLTurn(t *testing.T, codePath string, timeout time.Duration) engine.PlayerActionInput {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sandbox := NewSandbox(timeout).(*agentSandbox)
	session, err := sandbox.StartSession(ctx, "match-ml", map[string]string{"p1": codePath}, 1, 0)
	require.NoError(t, err)
	defer session.Close(ctx, "", "test")
	return session.ExecuteTurn(ctx, 0, "p1", json.RawMessage(`{"tick":0}`))
}

// Mide al bot dentro del sandbox ML: memoria real y virtual con un modelo
// ONNX cargado, núcleos permitidos e hilos de las librerías numéricas.
func TestMLRuntime_ProbeMemoryThreadsAndCPU(t *testing.T) {
	requireMLRuntime(t)
	r := require.New(t)
	onnxModel, err := os.ReadFile(resolveTestPath("games/starfighter/examples/neural/model/policy.onnx"))
	r.NoError(err)
	codePath := writeMLBundle(t, map[string]string{
		"bot.py": `import json, os, resource, sys
import numpy as np, onnxruntime as ort
o = ort.SessionOptions(); o.intra_op_num_threads = 1; o.inter_op_num_threads = 1
s = ort.InferenceSession("model/policy.onnx", o, providers=["CPUExecutionProvider"])
for _ in range(200):
    s.run(["logits"], {"features": np.zeros((1, 6), np.float32)})
status = {l.split(":")[0]: l.split(":")[1].strip() for l in open("/proc/self/status")}
init = json.loads(sys.stdin.readline()); msg = json.loads(sys.stdin.readline())
print(json.dumps({"type": "action", "tick": msg["tick"], "action": {
    "thrust": "OFF", "rss_kb": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
    "vm_peak": status["VmPeak"], "cpus": sorted(os.sched_getaffinity(0)),
    "threads": int(status["Threads"]), "omp": os.environ.get("OMP_NUM_THREADS"),
    "openblas": os.environ.get("OPENBLAS_NUM_THREADS"), "python": sys.version.split()[0]}}), flush=True)
`,
		"model/policy.onnx": string(onnxModel),
	})
	res := runMLTurn(t, codePath, 2*time.Second)
	r.Equal(engine.ActionStatusValid, res.Status, res.ErrorDetails)
	var probe map[string]interface{}
	r.NoError(json.Unmarshal(res.Payload, &probe))
	t.Logf("python-ml-cpu probe: %v", probe)
	r.Equal("3.12.14", probe["python"])
	r.Len(probe["cpus"], 1, "the bot is pinned to a single core")
	r.Equal("1", probe["omp"])
	r.Equal("1", probe["openblas"])
	rssMB := probe["rss_kb"].(float64) / 1024
	r.Less(rssMB, 1024.0, "resident memory must stay under the 1 GB limit")
}

// Un bot no puede reservar más de 1 GB: la asignación falla.
func TestMLRuntime_MemoryLimitIsEnforced(t *testing.T) {
	requireMLRuntime(t)
	r := require.New(t)
	codePath := writeMLBundle(t, map[string]string{
		"bot.py": `import json, sys
init = json.loads(sys.stdin.readline()); msg = json.loads(sys.stdin.readline())
try:
    blob = bytearray(1536 * 1024 * 1024)
    outcome = "allocated"
except MemoryError:
    outcome = "memory_error"
print(json.dumps({"type": "action", "tick": msg["tick"], "action": {"thrust": "OFF", "outcome": outcome}}), flush=True)
`,
	})
	res := runMLTurn(t, codePath, 5*time.Second)
	r.Equal(engine.ActionStatusValid, res.Status, res.ErrorDetails)
	var payload map[string]interface{}
	r.NoError(json.Unmarshal(res.Payload, &payload))
	r.Equal("memory_error", payload["outcome"], "a 1.5 GB allocation must fail under the 1 GB limit")
}

// La primera respuesta de un bot ML puede tardar hasta 10 s (carga del
// modelo); el mismo retraso en python-stdlib es un timeout.
func TestMLRuntime_FirstTurnAllowsModelLoading(t *testing.T) {
	requireMLRuntime(t)
	r := require.New(t)
	slowLoad := `import json, sys, time
time.sleep(5)
init = json.loads(sys.stdin.readline()); msg = json.loads(sys.stdin.readline())
print(json.dumps({"type": "action", "tick": msg["tick"], "action": {"thrust": "OFF"}}), flush=True)
`
	ml := runMLTurn(t, writeMLBundle(t, map[string]string{"bot.py": slowLoad}), 2*time.Second)
	r.Equal(engine.ActionStatusValid, ml.Status, "5 s of model loading fits the 10 s first-turn budget")

	std := runMLTurn(t, writeBundle(t, map[string]string{"bot.py": slowLoad}), 2*time.Second)
	r.Equal(engine.ActionStatusDisqualified, std.Status, "python-stdlib keeps the plain per-turn timeout")
}

// Sin el runtime instalado, la admisión rechaza con un motivo claro.
func TestMLRuntime_AdmissionFailsClosedWhenRuntimeMissing(t *testing.T) {
	if !IsRootlessSandboxAvailable() {
		t.Skip("rootless sandbox not available on host")
	}
	t.Setenv("AGENTRIX_BOT_RUNTIMES_DIR", t.TempDir())
	codePath := writeMLBundle(t, map[string]string{"bot.py": "pass\n"})
	err := NewSandbox(2*time.Second).ValidateBot(context.Background(), codePath)
	require.ErrorIs(t, err, ErrRuntimeUnavailable)
	require.ErrorContains(t, err, "python-ml-cpu is not installed")
}

// ADR-0014 (N3): los tres bots de ejemplo (.onnx, .safetensors, .npz)
// juegan una partida real de 5 junto a dos bots Ace.
func TestMLRuntime_ExampleNeuralBotsPlayFiveShipMatch(t *testing.T) {
	requireMLRuntime(t)
	example := func(bot, modelFile string) string {
		dir := resolveTestPath("games/starfighter/examples/neural")
		read := func(name string) string {
			content, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			return string(content)
		}
		return writeMLBundle(t, map[string]string{
			"bot.py":             read(bot),
			"policy.py":          read("policy.py"),
			"model/" + modelFile: read("model/" + modelFile),
		})
	}
	onnxBot := example("bot_onnx.py", "policy.onnx")
	stBot := example("bot_safetensors.py", "policy.safetensors")
	npzBot := example("bot_npz.py", "policy.npz")
	ace := "games/starfighter/examples/bot_ace.py"
	st := runRealMatch(t, []string{onnxBot, ace, stBot, ace, npzBot}, 5, 1800)
	require.Len(t, st.ranks, 5)
	for _, slot := range []int{0, 2, 4} {
		require.Greater(t, st.fired[slot], 0, "neural example bot in slot %d must play and shoot", slot)
	}
	t.Logf("reason=%s ticks=%d fired=%v kills=%v ranks=%v", st.reason, st.ticks, st.fired, st.kills, st.ranks)
}

// ADR-0014 (N3 + regla A): un bot que excede 1 GB sin capturar el error se
// cae y queda descalificado.
func TestMLRuntime_BotExceedingMemoryIsDisqualified(t *testing.T) {
	requireMLRuntime(t)
	codePath := writeMLBundle(t, map[string]string{
		"bot.py": `import json, sys
init = json.loads(sys.stdin.readline()); msg = json.loads(sys.stdin.readline())
blob = bytearray(1536 * 1024 * 1024)
print(json.dumps({"type": "action", "tick": msg["tick"], "action": {"thrust": "OFF"}}), flush=True)
`,
	})
	res := runMLTurn(t, codePath, 5*time.Second)
	require.Equal(t, engine.ActionStatusDisqualified, res.Status)
	require.Equal(t, crashCause, res.ErrorDetails)
}

// loopBot arma un bot que responde en bucle, con código opcional antes de
// leer init (carga del modelo) y por tick.
func loopBot(beforeInit, perTick string) string {
	return "import json, sys, time\n" + beforeInit + `
init = json.loads(sys.stdin.readline())
for line in sys.stdin:
    msg = json.loads(line)
` + perTick + `
    print(json.dumps({"type": "action", "tick": msg["tick"], "action": {"thrust": "OFF"}}), flush=True)
`
}

// ADR-0014 (N4): la admisión ejecuta el bot con su runtime y rechaza, con
// un motivo entendible, lo que no cumple los límites.
func TestMLAdmission_RejectsSlowOrHeavyModelsWithReason(t *testing.T) {
	requireMLRuntime(t)
	sandbox := NewSandbox(2 * time.Second)
	validate := func(bot string) error {
		return sandbox.ValidateBot(context.Background(), writeMLBundle(t, map[string]string{"bot.py": bot}))
	}

	cases := map[string]struct {
		bot    string
		reason string
	}{
		"model load over 10 s": {loopBot("time.sleep(12)", ""), "loading the model and answering the first tick took longer than 10s"},
		"inference over 2 s":   {loopBot("", "    if msg[\"tick\"] == 1: time.sleep(3)"), "answering tick 1 took longer than 2s"},
		"model over 1 GB":      {loopBot("blob = bytearray(1536 * 1024 * 1024)", ""), "exceeded the 1024 MB memory limit"},
		"unavailable library":  {loopBot("import torch", ""), "ModuleNotFoundError: No module named 'torch'"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := validate(tc.bot)
			require.Error(t, err)
			require.ErrorContains(t, err, tc.reason)
		})
	}

	// La plantilla ONNX de ejemplo sí se admite.
	dir := resolveTestPath("games/starfighter/examples/neural")
	read := func(name string) string {
		content, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		return string(content)
	}
	require.NoError(t, sandbox.ValidateBot(context.Background(), writeMLBundle(t, map[string]string{
		"bot.py": read("bot_onnx.py"), "policy.py": read("policy.py"), "model/policy.onnx": read("model/policy.onnx"),
	})))
}
