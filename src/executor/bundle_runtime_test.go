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

// writeBundle crea en disco la estructura de un paquete v2 admitido
// (ADR-0014) y devuelve el code_path de su bot.py.
func writeBundle(t *testing.T, files map[string]string) string {
	t.Helper()
	version := filepath.Join(t.TempDir(), "submissions", "agent", "v1")
	bundle := filepath.Join(version, model.BotBundleDirName)
	for name, content := range files {
		target := filepath.Join(bundle, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
		require.NoError(t, os.WriteFile(target, []byte(content), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(version, model.BotBundleManifestName), []byte(`{}`), 0o644))
	return filepath.Join(bundle, "bot.py")
}

var stdlibRuntime = pythonRuntime{name: model.BotRuntimePythonStdlib, interpreter: "python3"}

func TestSandboxArgs_MountBundleOrSingleFileReadOnlyAtBot(t *testing.T) {
	r := require.New(t)
	bundle := botMount{hostPath: "/data/v1/bundle", isBundle: true}
	single := botMount{hostPath: "/data/v1/bot.py"}

	bw := strings.Join(bubblewrapArgs(bundle, []string{"/usr"}, stdlibRuntime), " ")
	r.Contains(bw, "--ro-bind /data/v1/bundle /bot")
	r.Contains(bw, "--remount-ro / --chdir /bot python3 /bot/bot.py")
	r.Contains(bw, "--unshare-net")
	bwSingle := strings.Join(bubblewrapArgs(single, []string{"/usr"}, stdlibRuntime), " ")
	r.Contains(bwSingle, "--ro-bind /data/v1/bot.py /bot/bot.py")
	r.NotContains(bwSingle, "--ro-bind /data/v1 ")

	pm := strings.Join(podmanArgs(bundle, "img", stdlibRuntime), " ")
	r.Contains(pm, "-v /data/v1/bundle:/bot:ro -w /bot img python3 /bot/bot.py")
	r.Contains(pm, "--network none")
	r.Contains(strings.Join(podmanArgs(single, "img", stdlibRuntime), " "), "-v /data/v1/bot.py:/bot/bot.py:ro")
}

// runSingleTurn ejecuta un turno de un bot en el sandbox real y devuelve su
// acción como mapa.
func runSingleTurn(t *testing.T, codePath string) map[string]interface{} {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sandbox := NewSandbox(3 * time.Second).(*agentSandbox)
	session, err := sandbox.StartSession(ctx, "match-bundle", map[string]string{"p1": codePath}, 1, 0)
	require.NoError(t, err)
	defer session.Close(ctx, "", "test")
	res := session.ExecuteTurn(ctx, 0, "p1", json.RawMessage(`{"tick":0}`))
	require.Equal(t, engine.ActionStatusValid, res.Status, "bot must answer: %s", res.ErrorDetails)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(res.Payload, &payload))
	return payload
}

// ADR-0014 (N2): un bot v2 importa sus propios módulos y lee su modelo
// desde /bot, pero no puede escribir ahí.
func TestBotRuntime_BundleImportsModulesReadsModelAndIsReadOnly(t *testing.T) {
	if !IsRootlessSandboxAvailable() {
		t.Skip("rootless sandbox not available on host")
	}
	r := require.New(t)
	codePath := writeBundle(t, map[string]string{
		"bot.py": `import json, os, sys
import helper
init = json.loads(sys.stdin.readline())
msg = json.loads(sys.stdin.readline())
params = json.load(open("model/params.json"))
errors = {}
for target in ("/bot/hack.txt", "model/hack.json"):
    try:
        with open(target, "w") as f:
            f.write("x")
        errors[target] = None
    except OSError as e:
        errors[target] = type(e).__name__
print(json.dumps({"type": "action", "tick": msg["tick"], "action": {
    "thrust": "FORWARD", "gain": params["gain"], "helper": helper.NAME,
    "cwd": os.getcwd(), "write_errors": errors}}), flush=True)
`,
		"helper.py":         "NAME = 'helper-ok'\n",
		"model/params.json": `{"gain": 0.75}`,
		"agentrix.json":     `{"name":"x","entrypoint":"bot.py","protocol_version":"1.0"}`,
	})
	payload := runSingleTurn(t, codePath)
	r.Equal(0.75, payload["gain"], "the model file is readable")
	r.Equal("helper-ok", payload["helper"], "sibling modules are importable")
	r.Equal("/bot", payload["cwd"])
	writeErrors := payload["write_errors"].(map[string]interface{})
	for target, errName := range writeErrors {
		r.NotNil(errName, "writing %s must fail", target)
	}
	r.NoFileExists(filepath.Join(filepath.Dir(codePath), "hack.txt"))
	r.NoFileExists(filepath.Join(filepath.Dir(codePath), "model", "hack.json"))
}

// Un bot v1 sigue viendo solo su archivo: el resto de su carpeta no existe
// para él.
func TestBotRuntime_SingleFileBotCannotSeeSiblingFiles(t *testing.T) {
	if !IsRootlessSandboxAvailable() {
		t.Skip("rootless sandbox not available on host")
	}
	r := require.New(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("s3cret"), 0o644))
	codePath := filepath.Join(dir, "bot.py")
	require.NoError(t, os.WriteFile(codePath, []byte(`import json, os, sys
init = json.loads(sys.stdin.readline())
msg = json.loads(sys.stdin.readline())
print(json.dumps({"type": "action", "tick": msg["tick"], "action": {
    "thrust": "FORWARD", "listing": sorted(os.listdir("/bot")),
    "secret": os.path.exists("secret.txt") or os.path.exists("/bot/secret.txt")}}), flush=True)
`), 0o644))
	payload := runSingleTurn(t, codePath)
	r.Equal([]interface{}{"bot.py"}, payload["listing"])
	r.Equal(false, payload["secret"])
}

// El digest de un bot v2 se recalcula desde el disco: coincide con el de la
// admisión y cambia si se modifica cualquier archivo del paquete.
func TestBotArtifactDigest_CoversWholeBundle(t *testing.T) {
	r := require.New(t)
	codePath := writeBundle(t, map[string]string{"bot.py": "pass\n", "model/w.json": "[1]"})
	digest, err := model.BotArtifactDigest(codePath)
	r.NoError(err)
	botSum, _ := model.ComputeFileSHA256(codePath)
	modelSum, _ := model.ComputeFileSHA256(filepath.Join(filepath.Dir(codePath), "model", "w.json"))
	r.Equal(model.BundleListingDigest(map[string]string{"bot.py": botSum, "model/w.json": modelSum}), digest)

	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(codePath), "model", "w.json"), []byte("[2]"), 0o644))
	changed, err := model.BotArtifactDigest(codePath)
	r.NoError(err)
	r.NotEqual(digest, changed)
}

// ADR-0014 (N2): una partida real de 5 con dos bots empaquetados, que
// importan su lógica como módulo propio y leen su configuración desde
// model/params.json, se juega completa y sella su replay.
func TestBundleBots_PlayRealFreeForAll(t *testing.T) {
	if !IsRootlessSandboxAvailable() {
		t.Skip("rootless sandbox not available on host")
	}
	aceSource, err := os.ReadFile(resolveTestPath("games/starfighter/examples/bot_ace.py"))
	require.NoError(t, err)
	bundleBot := writeBundle(t, map[string]string{
		"ace.py": string(aceSource),
		"bot.py": `import json
import ace

params = json.load(open("model/params.json"))
init = ace.read()
pilot = ace.Ace(str(init.get("player_id", "")))
pilot.preferred_distance = params["preferred_distance"]
while True:
    msg = ace.read()
    if msg["type"] == "end":
        break
    ace.send({"type": "action", "tick": msg["tick"], "action": pilot.choose_action(msg["perception"])})
`,
		"model/params.json": `{"preferred_distance": 520}`,
		"agentrix.json":     `{"name":"bundle-ace","entrypoint":"bot.py","protocol_version":"1.0"}`,
	})
	ace := "games/starfighter/examples/bot_ace.py"
	st := runRealMatch(t, []string{bundleBot, ace, bundleBot, ace, ace}, 3, 3600)
	require.Contains(t, []string{"eliminated", "score_limit"}, st.reason)
	require.Len(t, st.ranks, 5)
	for _, slot := range []int{0, 2} {
		require.Greater(t, st.fired[slot], 0, "bundle bot in slot %d must play and shoot", slot)
	}
}
