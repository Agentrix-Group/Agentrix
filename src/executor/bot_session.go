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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/F4nk1/Agentrix/src/engine"
	"github.com/F4nk1/Agentrix/src/tracer"
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
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	lines           chan string
	connected       bool
	disqualified    bool
	disqualifyCause string
	disconnectOnce  sync.Once
}

func pythonCommand(codePath string) (*exec.Cmd, error) {
	resolved := codePath
	if _, err := os.Stat(resolved); err != nil {
		candidate := filepath.Join("../..", resolved)
		if _, candidateErr := os.Stat(candidate); candidateErr != nil {
			return nil, fmt.Errorf("python bot not found: %s", codePath)
		}
		resolved = candidate
	}
	if !strings.EqualFold(filepath.Ext(resolved), ".py") {
		return nil, fmt.Errorf("unsupported bot artifact %q: Agentrix MVP accepts only Python .py files", codePath)
	}
	return exec.Command("python3", resolved), nil
}

func startBotProcess(playerID, codePath string) (*botProcess, error) {
	cmd, err := pythonCommand(codePath)
	if err != nil {
		return nil, err
	}
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
	go func() { _, _ = io.Copy(io.Discard, stderr) }()

	return &botProcess{playerID: playerID, cmd: cmd, stdin: stdin, lines: lines, connected: true}, nil
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
		if p.cmd != nil && p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
			_ = p.cmd.Wait()
		}
	})
}

func (p *botProcess) disqualify(reason string) {
	p.disqualified = true
	p.disqualifyCause = reason
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

	session, err := s.StartSession(ctx, "admission-check", map[string]string{"candidate": codePath}, 1, s.timeout)
	if err != nil {
		return err
	}
	defer session.Close(context.Background(), "", "admission_check")
	perception := json.RawMessage(`{
		"tick":0,
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
	}`)
	input := session.ExecuteTurn(ctx, 0, "candidate", perception)
	if input.Status != engine.ActionStatusValid {
		return fmt.Errorf("bot failed admission tick: %s (%s)", input.Status, input.ErrorDetails)
	}
	return nil
}

// StartSession starts one persistent Python process per player. The first and
// only startup message is init; bots do not negotiate a second handshake.
func (s *agentSandbox) StartSession(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error) {
	if timeout <= 0 {
		timeout = s.timeout
	}
	session := &botSession{matchID: matchID, timeout: timeout, processes: make(map[string]*botProcess, len(players))}
	for playerID, codePath := range players {
		proc, err := startBotProcess(playerID, codePath)
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
		proc.disconnect()
		return engine.PlayerActionInput{Status: engine.ActionStatusCrashed, ErrorDetails: err.Error()}
	}

	line, outcome := proc.recv(ctx, s.timeout)
	switch outcome {
	case "timeout":
		proc.disqualify("timeout")
		tracer.WarnEvent(ctx, tracer.ScopeAgent, "agent.turn.disqualified_timeout", "El bot excedió el tiempo y fue terminado",
			tracer.Origin(tracer.OriginAgent), tracer.String("agent_id", playerID), tracer.Int("tick", tick))
		return engine.PlayerActionInput{Status: engine.ActionStatusDisqualified, ErrorDetails: "timeout"}
	case "cancelled":
		proc.disconnect()
		return engine.PlayerActionInput{Status: engine.ActionStatusCrashed, ErrorDetails: ctx.Err().Error()}
	case "crashed":
		proc.disconnect()
		return engine.PlayerActionInput{Status: engine.ActionStatusCrashed, ErrorDetails: "bot process disconnected"}
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
