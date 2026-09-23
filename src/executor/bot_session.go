package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

const BotProtocolVersion = "1.0"

type BotSession interface {
	ExecuteTurn(ctx context.Context, tick int, playerID string, perception json.RawMessage) engine.PlayerActionInput
	Close(ctx context.Context, winner string, reason string)
}

type botOutgoing struct {
	Type            string          `json:"type"`
	ProtocolVersion string          `json:"protocol_version,omitempty"`
	Protocol        string          `json:"protocol,omitempty"`
	MatchID         string          `json:"match_id,omitempty"`
	PlayerID        string          `json:"player_id,omitempty"`
	GameID          string          `json:"game_id,omitempty"`
	Seed            int64           `json:"seed,omitempty"`
	TimeoutMs       int64           `json:"timeout_ms,omitempty"`
	Tick            *int            `json:"tick,omitempty"`
	Perception      json.RawMessage `json:"perception,omitempty"`
	Data            json.RawMessage `json:"data,omitempty"`
	Winner          string          `json:"winner,omitempty"`
	WinnerID        string          `json:"winnerId,omitempty"`
	Reason          string          `json:"reason,omitempty"`
}

type botIncoming struct {
	Type   string          `json:"type"`
	Tick   *int            `json:"tick"`
	Action json.RawMessage `json:"action,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type botProcess struct {
	playerID        string
	proc            BotProcess
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	lines           chan string
	connected       bool
	disqualified    bool
	disqualifyCause string
	disconnectOnce  sync.Once
	// firstTurnTimeout amplía la espera de la primera respuesta (carga del
	// modelo en python-ml-cpu); answered indica que ya respondió una vez.
	firstTurnTimeout time.Duration
	answered         bool
	// stderrTail guarda el final de stderr del bot para diagnosticar una
	// caída (p. ej. MemoryError) sin guardar toda su salida.
	stderrTail *tailBuffer
}

// crashCause es el motivo de descalificación de un bot cuyo proceso murió
// durante la partida (ADR-0014, regla A): su nave sale en ese tick.
const crashCause = "crash"

// tailBuffer conserva los últimos max bytes escritos.
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.max {
		t.buf = append([]byte(nil), t.buf[len(t.buf)-t.max:]...)
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}

func IsRootlessSandboxAvailable() bool {
	rt := &BubblewrapRuntime{}
	return rt.IsAvailable()
}

func startBotProcess(playerID, codePath string, runtime ...BotRuntime) (*botProcess, error) {
	return startBotProcessOn(playerID, codePath, 0, runtime...)
}

// startBotProcessOn inicia el bot con el runtime que declara su paquete; cpu
// es el núcleo asignado (python-ml-cpu).
func startBotProcessOn(playerID, codePath string, cpu int, runtime ...BotRuntime) (*botProcess, error) {
	var rt BotRuntime
	if len(runtime) > 0 && runtime[0] != nil {
		rt = runtime[0]
	} else {
		rt = DefaultBotRuntime()
	}

	runtimeName := model.BotBundleRuntime(codePath)
	botProc, err := rt.Spawn(context.Background(), BotRuntimeConfig{
		PlayerID: playerID,
		CodePath: codePath,
		Runtime:  runtimeName,
		CPU:      cpu,
	})
	if err != nil {
		return nil, err
	}
	var firstTurnTimeout time.Duration
	if runtimeName == model.BotRuntimePythonMLCPU {
		firstTurnTimeout = mlInitTimeout
	}

	lines := make(chan string)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(botProc.Stdout())
		scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	stderrTail := &tailBuffer{max: 2048}
	go func() { _, _ = io.Copy(stderrTail, botProc.Stderr()) }()

	return &botProcess{
		stderrTail:       stderrTail,
		firstTurnTimeout: firstTurnTimeout,
		playerID:         playerID,
		proc:             botProc,
		stdin:            botProc.Stdin(),
		lines:            lines,
		connected:        true,
	}, nil
}

func (p *botProcess) send(msg botOutgoing) error {
	if !p.connected {
		return fmt.Errorf("bot process for %s is disconnected", p.playerID)
	}
	line, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := p.stdin.Write(append(line, '\n')); err != nil {
		p.connected = false
		return err
	}
	return nil
}

func (p *botProcess) recv(ctx context.Context, timeout time.Duration) (string, string) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case line, open := <-p.lines:
		if !open {
			p.connected = false
			return "", "crashed"
		}
		return line, ""
	case <-timer.C:
		return "", "timeout"
	case <-ctx.Done():
		return "", "cancelled"
	}
}

func (p *botProcess) disconnect() {
	p.disconnectOnce.Do(func() {
		p.connected = false
		if p.stdin != nil {
			_ = p.stdin.Close()
		}
		if p.proc != nil {
			_ = p.proc.Kill()
			_ = p.proc.Wait()
		} else if p.cmd != nil && p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
			_ = p.cmd.Wait()
		}
	})
}

func (p *botProcess) disqualify(reason string) {
	p.disqualified = true
	p.disqualifyCause = reason
	tracer.RecordSandboxDisqualification()
	p.disconnect()
}

type botSession struct {
	matchID   string
	timeout   time.Duration
	mu        sync.Mutex
	processes map[string]*botProcess
}

func (s *agentSandbox) ValidateBot(ctx context.Context, codePath string) error {
	source, err := os.ReadFile(codePath)
	if err != nil {
		return err
	}
	parse := exec.CommandContext(ctx, "python3", "-c", "import ast,sys; ast.parse(sys.stdin.read(), filename='bot.py')")
	parse.Stdin = bytes.NewReader(source)
	if output, err := parse.CombinedOutput(); err != nil {
		return fmt.Errorf("invalid Python syntax: %s", strings.TrimSpace(string(output)))
	}

	// Un runtime no instalado se rechaza con un motivo claro, sin ejecutar.
	runtimeName := model.BotBundleRuntime(codePath)
	py, err := resolvePythonRuntime(runtimeName, 0)
	if err != nil {
		return err
	}
	session, err := s.StartSession(ctx, "admission-check", map[string]string{"candidate": codePath}, 1, s.timeout)
	if err != nil {
		return err
	}
	defer session.Close(context.Background(), "", "admission_check")
	var proc *botProcess
	if concrete, ok := session.(*botSession); ok {
		proc = concrete.processes["candidate"]
	}
	// Dos ticks (ADR-0014): el primero incluye la carga del modelo; el
	// segundo mide una respuesta normal dentro del tiempo por tick.
	for tick := 0; tick < 2; tick++ {
		input := session.ExecuteTurn(ctx, tick, "candidate", admissionPerception(tick))
		if input.Status != engine.ActionStatusValid {
			return admissionFailure(tick, input, py, s.timeout, proc)
		}
	}
	return nil
}

// admissionPerception es la percepción de prueba de la admisión.
func admissionPerception(tick int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"tick":%d,
		"player_id":0,
		"myself":{
			"position":{"x":0,"y":0},
			"velocity":{"x":0,"y":0},
			"facing":{"x":1,"y":0},
			"health":100,
			"energy":100,
			"shield_active":false,
			"remaining_bullet_cooldown":0
		},
		"rivals":[],
		"bullets":[]
	}`, tick))
}

// admissionFailure traduce el fallo de un tick de admisión a un motivo que
// el autor del bot pueda entender y corregir.
func admissionFailure(tick int, input engine.PlayerActionInput, py pythonRuntime, turnTimeout time.Duration, proc *botProcess) error {
	stderr := ""
	if proc != nil && proc.stderrTail != nil {
		stderr = proc.stderrTail.String()
	}
	switch {
	case input.ErrorDetails == "timeout" && tick == 0 && py.initTimeout > turnTimeout:
		return fmt.Errorf("bot failed admission: loading the model and answering the first tick took longer than %s", py.initTimeout)
	case input.ErrorDetails == "timeout":
		return fmt.Errorf("bot failed admission: answering tick %d took longer than %s", tick, turnTimeout)
	case strings.Contains(stderr, "MemoryError"):
		return fmt.Errorf("bot failed admission: exceeded the %d MB memory limit", mlMemoryLimitBytes>>20)
	case input.ErrorDetails == crashCause:
		if last := lastErrorLine(stderr); last != "" {
			return fmt.Errorf("bot failed admission: the bot process stopped: %s", last)
		}
		return fmt.Errorf("bot failed admission: the bot process stopped at tick %d", tick)
	default:
		return fmt.Errorf("bot failed admission tick %d: %s (%s)", tick, input.Status, input.ErrorDetails)
	}
}

// lastErrorLine devuelve la última línea no vacía de stderr (en Python, la
// del tipo de excepción), recortada.
func lastErrorLine(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if len(last) > 300 {
		last = last[:300] + "..."
	}
	return last
}

// StartSession starts one persistent Python process per player. The first and
// only startup message is init; bots do not negotiate a second handshake.
func (s *agentSandbox) StartSession(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error) {
	if timeout <= 0 {
		timeout = s.timeout
	}
	session := &botSession{matchID: matchID, timeout: timeout, processes: make(map[string]*botProcess, len(players))}
	// Núcleos asignados en orden estable de slot: un núcleo por bot ML.
	playerIDs := make([]string, 0, len(players))
	for playerID := range players {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Strings(playerIDs)
	cpus := allowedCPUs()
	for index, playerID := range playerIDs {
		codePath := players[playerID]
		proc, err := startBotProcessOn(playerID, codePath, cpus[index%len(cpus)], s.runtime)
		if err != nil {
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.session.start_failed", "No se pudo iniciar el proceso Python del bot",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Err(err))
			session.processes[playerID] = &botProcess{playerID: playerID}
			continue
		}
		if err := proc.send(botOutgoing{
			Type: "init", ProtocolVersion: BotProtocolVersion, Protocol: "agentrix-bot/1", MatchID: matchID,
			PlayerID: playerID, GameID: "starfighter", Seed: seed, TimeoutMs: timeout.Milliseconds(),
		}); err != nil {
			proc.disconnect()
			session.processes[playerID] = &botProcess{playerID: playerID}
			continue
		}
		session.processes[playerID] = proc
	}
	return session, nil
}

func (s *botSession) ExecuteTurn(ctx context.Context, tick int, playerID string, perception json.RawMessage) engine.PlayerActionInput {
	s.mu.Lock()
	proc, known := s.processes[playerID]
	s.mu.Unlock()
	if !known || proc == nil {
		return engine.PlayerActionInput{Status: engine.ActionStatusCrashed, ErrorDetails: "bot process not configured"}
	}
	if proc.disqualified {
		return engine.PlayerActionInput{Status: engine.ActionStatusDisqualified, ErrorDetails: proc.disqualifyCause}
	}
	if !proc.connected {
		return engine.PlayerActionInput{Status: engine.ActionStatusCrashed, ErrorDetails: "bot process not connected"}
	}

	requestedTick := tick
	if err := proc.send(botOutgoing{
		Type: "perception", MatchID: s.matchID, PlayerID: playerID,
		Tick: &requestedTick, Perception: perception, Data: perception,
	}); err != nil {
		// No se pudo escribirle: el proceso murió.
		proc.disqualify(crashCause)
		return engine.PlayerActionInput{Status: engine.ActionStatusDisqualified, ErrorDetails: crashCause}
	}

	turnTimeout := s.timeout
	if !proc.answered && proc.firstTurnTimeout > turnTimeout {
		turnTimeout = proc.firstTurnTimeout
	}
	line, outcome := proc.recv(ctx, turnTimeout)
	if outcome == "" {
		proc.answered = true
	}
	switch outcome {
	case "timeout":
		tracer.RecordSandboxTimeout()
		proc.disqualify("timeout")
		tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.disqualified_timeout", "El bot excedió el tiempo y fue terminado",
			tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("tick", tick))
		return engine.PlayerActionInput{Status: engine.ActionStatusDisqualified, ErrorDetails: "timeout"}
	case "cancelled":
		proc.disconnect()
		return engine.PlayerActionInput{Status: engine.ActionStatusCrashed, ErrorDetails: ctx.Err().Error()}
	case "crashed":
		// El proceso terminó durante la partida: queda descalificado.
		proc.disqualify(crashCause)
		tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.disqualified_crash", "El proceso del bot terminó y fue descalificado",
			tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("tick", tick))
		return engine.PlayerActionInput{Status: engine.ActionStatusDisqualified, ErrorDetails: crashCause}
	}

	var msg botIncoming
	if err := json.Unmarshal([]byte(line), &msg); err != nil || msg.Type != "action" || msg.Tick == nil {
		proc.disqualify("invalid_output")
		return engine.PlayerActionInput{Status: engine.ActionStatusInvalidOutput, ErrorDetails: "malformed or unexpected message"}
	}
	actionPayload := msg.Action
	if len(actionPayload) == 0 && len(msg.Data) > 0 {
		actionPayload = msg.Data
	}
	if !isJSONObject(actionPayload) {
		proc.disqualify("invalid_output")
		return engine.PlayerActionInput{Status: engine.ActionStatusInvalidOutput, ErrorDetails: "malformed or unexpected message"}
	}
	if *msg.Tick != tick {
		proc.disqualify("tick_mismatch")
		return engine.PlayerActionInput{Status: engine.ActionStatusInvalidOutput, ErrorDetails: fmt.Sprintf("tick mismatch: got %d, expected %d", *msg.Tick, tick)}
	}
	return engine.PlayerActionInput{Status: engine.ActionStatusValid, Payload: append(json.RawMessage(nil), actionPayload...)}
}

func isJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(raw, &object) == nil && object != nil
}

func (s *botSession) Close(ctx context.Context, winner string, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for playerID, proc := range s.processes {
		if proc == nil || !proc.connected {
			continue
		}
		_ = proc.send(botOutgoing{Type: "end", MatchID: s.matchID, Winner: winner, Reason: reason})
		proc.disconnect()
		tracer.InfoEvent(ctx, tracer.ScopeAgent, "agent.session.closed", "Sesión de bot cerrada",
			tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID))
	}
}
