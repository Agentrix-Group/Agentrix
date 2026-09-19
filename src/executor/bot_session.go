package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/F4nk1/Agentrix/src/engine"
	"github.com/F4nk1/Agentrix/src/tracer"
)

// BotProtocolVersion identifies the bot<->executor wire protocol (persistent
// process, JSON Lines over stdin/stdout, one message per line). It mirrors
// the shape of the already-implemented Motor<->Go Envelope protocol
// (src/engine/types.go) in spirit -- handshake, init, a per-tick
// request/response, and a clean end -- but is intentionally independent of
// it: bots are a different audience (participant-submitted code, not
// platform-trusted engines) and the payload of "perception"/"action" is
// opaque JSON here, since each game defines its own shape. Field names are
// snake_case to match the rest of Agentrix's JSON contracts
// (contracts/*.schema.json), not the camelCase used internally by the
// engine Envelope.
const BotProtocolVersion = "1.0"

// BotSession represents a live, per-match set of persistent bot processes.
// It replaces the previous design where ExecuteTurnWithPerception spawned a
// brand new process on every tick (one Python invocation per tick, per bot,
// passing state as argv) -- that made a multi-tick handshake/negotiation
// impossible and made a bot's own in-process state (e.g. anything it
// computed last tick) unrecoverable between calls. A BotSession starts one
// process per player when the match begins, keeps it alive for the whole
// match, and only tears it down at the end.
type BotSession interface {
	// ExecuteTurn sends this tick's perception to the given player's bot
	// process and waits (bounded by the session's timeout) for its action.
	// It never returns an error to the caller: a timeout, malformed
	// response, mismatched tick, or a dead process are all reported via
	// the returned PlayerActionInput.Status, matching RF-044 ("una acción
	// ausente, tardía o inválida debe transformarse en la consecuencia
	// definida por el juego" -- deciding that consequence is the engine's
	// job, not this session's).
	ExecuteTurn(ctx context.Context, tick int, playerID string, perception interface{}) engine.PlayerActionInput

	// Close sends an `end` message to every still-connected bot and
	// terminates their processes. Safe to call multiple times.
	Close(ctx context.Context, winner string, reason string)
}

type botOutgoing struct {
	Type            string      `json:"type"`
	ProtocolVersion string      `json:"protocol_version"`
	MatchID         string      `json:"match_id"`
	PlayerID        string      `json:"player_id,omitempty"`
	Seed            int64       `json:"seed,omitempty"`
	TimeoutMs       int64       `json:"timeout_ms,omitempty"`
	Config          interface{} `json:"config,omitempty"`
	Tick            int         `json:"tick,omitempty"`
	Perception      interface{} `json:"perception,omitempty"`
	Winner          string      `json:"winner,omitempty"`
	Reason          string      `json:"reason,omitempty"`
}

type botIncoming struct {
	Type            string                 `json:"type"`
	ProtocolVersion string                 `json:"protocol_version"`
	Tick            int                    `json:"tick"`
	Action          map[string]interface{} `json:"action"`
}

// botProcess wraps one persistent bot subprocess: its stdin for writing
// requests, and a background goroutine forwarding its stdout lines to a
// channel so the session can apply a per-call read timeout without
// blocking on a synchronous read (mirrors the mpsc::channel + recv_timeout
// pattern already used on the Rust side for the same reason -- no async
// runtime needed for this, just a reader goroutine and a channel).
type botProcess struct {
	playerID  string
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	lines     chan string
	connected bool
}

func interpreterFor(codePath string) *exec.Cmd {
	// Minimal, not a full runner system (that's ATD-008, out of scope
	// here): a `.py` script is invoked with `python3`; anything else is
	// invoked directly as an executable. This replaces the previous
	// hardcoded `exec.Command("python3", scriptPath, ...)` that made any
	// non-Python bot impossible.
	resolved := codePath
	if _, err := os.Stat(resolved); err != nil {
		if _, err := os.Stat(filepath.Join("../..", resolved)); err == nil {
			resolved = filepath.Join("../..", resolved)
		}
	}
	if strings.HasSuffix(resolved, ".py") {
		return exec.Command("python3", resolved)
	}
	return exec.Command(resolved)
}

func startBotProcess(playerID, codePath string) (*botProcess, error) {
	cmd := interpreterFor(codePath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	lines := make(chan string)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	go func() {
		_, _ = io.Copy(io.Discard, stderr)
	}()

	return &botProcess{
		playerID:  playerID,
		cmd:       cmd,
		stdin:     stdin,
		lines:     lines,
		connected: true,
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

// recvWithTimeout returns (line, ok). ok=false covers both a timeout and a
// closed channel (process died / stdout EOF) -- the caller treats both the
// same way, per RF-044.
func (p *botProcess) recvWithTimeout(timeout time.Duration) (string, bool) {
	select {
	case line, open := <-p.lines:
		if !open {
			p.connected = false
			return "", false
		}
		return line, true
	case <-time.After(timeout):
		return "", false
	}
}

func (p *botProcess) drain() {
	for {
		select {
		case <-p.lines:
		default:
			return
		}
	}
}

func extractPerceptionTick(perception interface{}) (int, bool) {
	if pMap, ok := perception.(map[string]interface{}); ok {
		if tVal, exists := pMap["tick"]; exists {
			switch v := tVal.(type) {
			case float64:
				return int(v), true
			case int:
				return v, true
			case int64:
				return int(v), true
			case json.Number:
				if n, err := v.Int64(); err == nil {
					return int(n), true
				}
			}
		}
	}
	return 0, false
}

func isValidTick(msgTick, expectedTick int, perception interface{}) bool {
	if msgTick == expectedTick {
		return true
	}
	if pTick, ok := extractPerceptionTick(perception); ok && msgTick == pTick {
		return true
	}
	// Initial tick boundary: executor asks for tick 1 to advance state from initial tick 0
	if expectedTick == 1 && msgTick == 0 {
		return true
	}
	return false
}

func isStaleTick(msgTick, expectedTick int, perception interface{}) bool {
	// Tick 0 when expectedTick is 1 is the valid initial tick boundary, not stale.
	if expectedTick == 1 && msgTick == 0 {
		return false
	}
	if pTick, ok := extractPerceptionTick(perception); ok {
		if msgTick < pTick {
			return true
		}
	}
	return msgTick < expectedTick
}

func (p *botProcess) disconnect() {
	if !p.connected && p.cmd.Process == nil {
		return
	}
	p.connected = false
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	_ = p.cmd.Wait()
}

type botSession struct {
	matchID   string
	timeout   time.Duration
	mu        sync.Mutex
	processes map[string]*botProcess
}

// StartSession launches one persistent process per player and performs the
// handshake+init steps. A player whose process fails to start, fails to
// handshake, or reports an incompatible protocol version is marked
// disconnected from the start -- ExecuteTurn will report it as
// ActionStatusCrashed for every tick without ever spawning it again,
// consistent with how a mid-match death is handled.
func (s *agentSandbox) StartSession(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error) {
	if timeout <= 0 {
		timeout = s.timeout
	}
	session := &botSession{
		matchID:   matchID,
		timeout:   timeout,
		processes: make(map[string]*botProcess, len(players)),
	}

	for playerID, codePath := range players {
		proc, err := startBotProcess(playerID, codePath)
		if err != nil {
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.session.start_failed", "No se pudo iniciar el proceso del bot",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Err(err))
			session.processes[playerID] = &botProcess{playerID: playerID, connected: false}
			continue
		}

		if err := proc.send(botOutgoing{
			Type:            "handshake",
			ProtocolVersion: BotProtocolVersion,
			MatchID:         matchID,
			PlayerID:        playerID,
			Seed:            seed,
		}); err != nil {
			proc.disconnect()
			session.processes[playerID] = &botProcess{playerID: playerID, connected: false}
			continue
		}

		line, ok := proc.recvWithTimeout(timeout)
		if !ok {
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.session.handshake_timeout", "El bot no respondió el handshake",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID))
			proc.disconnect()
			session.processes[playerID] = &botProcess{playerID: playerID, connected: false}
			continue
		}

		var ack botIncoming
		if err := json.Unmarshal([]byte(line), &ack); err != nil || ack.Type != "handshake_ack" || ack.ProtocolVersion != BotProtocolVersion {
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.session.handshake_invalid", "Handshake inválido o versión de protocolo incompatible",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID))
			proc.disconnect()
			session.processes[playerID] = &botProcess{playerID: playerID, connected: false}
			continue
		}

		_ = proc.send(botOutgoing{
			Type:            "init",
			ProtocolVersion: BotProtocolVersion,
			MatchID:         matchID,
			TimeoutMs:       timeout.Milliseconds(),
		})

		session.processes[playerID] = proc
	}

	return session, nil
}

func (s *botSession) ExecuteTurn(ctx context.Context, tick int, playerID string, perception interface{}) engine.PlayerActionInput {
	s.mu.Lock()
	proc, known := s.processes[playerID]
	s.mu.Unlock()

	if !known || !proc.connected {
		return engine.PlayerActionInput{
			Status:       engine.ActionStatusCrashed,
			ErrorDetails: "bot process not connected",
		}
	}

	// Drain any unconsumed stale output before sending this tick's perception
	proc.drain()

	if err := proc.send(botOutgoing{
		Type:            "perception",
		ProtocolVersion: BotProtocolVersion,
		MatchID:         s.matchID,
		PlayerID:        playerID,
		Tick:            tick,
		Perception:      perception,
	}); err != nil {
		tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.write_failed", "No se pudo enviar la percepción al bot (proceso probablemente muerto)",
			tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("tick", tick))
		return engine.PlayerActionInput{Status: engine.ActionStatusCrashed, ErrorDetails: err.Error()}
	}

	deadline := time.Now().Add(s.timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.timeout", "Timeout esperando la acción del bot",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("tick", tick))
			return engine.PlayerActionInput{Status: engine.ActionStatusTimeout}
		}

		line, ok := proc.recvWithTimeout(remaining)
		if !ok {
			if !proc.connected {
				return engine.PlayerActionInput{Status: engine.ActionStatusCrashed, ErrorDetails: "bot process disconnected"}
			}
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.timeout", "Timeout esperando la acción del bot",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("tick", tick))
			return engine.PlayerActionInput{Status: engine.ActionStatusTimeout}
		}

		var msg botIncoming
		if err := json.Unmarshal([]byte(line), &msg); err != nil || msg.Type != "action" {
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.invalid_output", "El bot devolvió una respuesta inválida",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("tick", tick))
			return engine.PlayerActionInput{Status: engine.ActionStatusInvalidOutput, ErrorDetails: "malformed or unexpected message"}
		}

		if isStaleTick(msg.Tick, tick, perception) {
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.stale_tick_discarded", "Descartando respuesta tardía de tick anterior",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("got_tick", msg.Tick), tracer.Int("expected_tick", tick))
			continue
		}

		if !isValidTick(msg.Tick, tick, perception) {
			tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.tick_mismatch", "El bot respondió a un tick distinto del pedido",
				tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("tick", tick))
			return engine.PlayerActionInput{Status: engine.ActionStatusInvalidOutput, ErrorDetails: "tick mismatch"}
		}

		actionType, _ := msg.Action["type"].(string)
		return engine.PlayerActionInput{
			Status:     engine.ActionStatusValid,
			ActionType: actionType,
			Payload:    msg.Action,
		}
	}
}

func (s *botSession) Close(ctx context.Context, winner string, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for playerID, proc := range s.processes {
		if !proc.connected {
			continue
		}
		_ = proc.send(botOutgoing{
			Type:            "end",
			ProtocolVersion: BotProtocolVersion,
			MatchID:         s.matchID,
			Winner:          winner,
			Reason:          reason,
		})
		proc.disconnect()
		tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.session.closed", "Sesión de bot cerrada",
			tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID))
	}
}
