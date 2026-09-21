package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/model"
)

const (
	BotProtocol        = "agentrix-bot/1"
	BotProtocolVersion = "1.0"
)

// Disqualification causes (ADR-0004).
const (
	CauseTimeout       = "timeout"
	CauseCrash         = "crash"
	CauseInvalidAction = "invalid_action"
	CauseProtocol      = "protocol_error"
	CauseResourceLimit = "resource_limit"
)

// BotLaunch describes one participant of a session.
type BotLaunch struct {
	PlayerID string
	CodePath string
	Runtime  string
}

type botConn struct {
	playerID    string
	proc        BotProcess
	lines       chan lineResult
	outputBytes int64
	cause       string
	closeOnce   sync.Once
}

type lineResult struct {
	line []byte
	err  error
}

// Session keeps one persistent sandboxed process per player for the whole
// match. Every bot receives init once, then one perception per tick.
type Session struct {
	matchID string
	limits  model.ExecutionLimits
	bots    map[string]*botConn
	order   []string
	started bool
}

type outgoing struct {
	Type            string          `json:"type"`
	Protocol        string          `json:"protocol,omitempty"`
	ProtocolVersion string          `json:"protocol_version,omitempty"`
	MatchID         string          `json:"match_id,omitempty"`
	PlayerID        string          `json:"player_id,omitempty"`
	GameID          string          `json:"game_id,omitempty"`
	Seed            *int64          `json:"seed,omitempty"`
	TimeoutMs       int             `json:"timeout_ms,omitempty"`
	Tick            *int            `json:"tick,omitempty"`
	Perception      json.RawMessage `json:"perception,omitempty"`
	Winner          *string         `json:"winner,omitempty"`
	Reason          string          `json:"reason,omitempty"`
}

type incoming struct {
	Type   string          `json:"type"`
	Tick   *int            `json:"tick"`
	Action json.RawMessage `json:"action"`
}

// StartSession spawns every bot and sends init. A spawn failure is an
// infrastructure error (ErrSpawn) and aborts the session: a competitor is
// never silently replaced or dropped. A bot that dies right after starting
// is its own fault and is disqualified on its first turn.
func StartSession(ctx context.Context, runtime BotRuntime, matchID, gameID string, seed int64,
	limits model.ExecutionLimits, launches []BotLaunch) (*Session, error) {
	s := &Session{matchID: matchID, limits: limits, bots: map[string]*botConn{}}
	for _, l := range launches {
		proc, err := runtime.Spawn(ctx, BotSpec{CodePath: l.CodePath, Runtime: l.Runtime, Limits: limits})
		if err != nil {
			s.Close("", "aborted")
			if errors.Is(err, ErrSpawn) {
				return nil, err
			}
			return nil, fmt.Errorf("%w: %v", ErrSpawn, err)
		}
		conn := &botConn{playerID: l.PlayerID, proc: proc, lines: make(chan lineResult)}
		go conn.read(limits.MaxLineBytes)
		s.bots[l.PlayerID] = conn
		s.order = append(s.order, l.PlayerID)
		seedCopy := seed
		if err := conn.send(outgoing{Type: "init", Protocol: BotProtocol, ProtocolVersion: BotProtocolVersion,
			MatchID: matchID, PlayerID: l.PlayerID, GameID: gameID, Seed: &seedCopy, TimeoutMs: limits.TurnTimeoutMs},
			time.Duration(limits.InitTimeoutMs)*time.Millisecond); err != nil {
			conn.fail(CauseCrash)
		}
	}
	return s, nil
}

func (c *botConn) read(maxLine int) {
	defer close(c.lines)
	reader := bufio.NewReaderSize(c.proc.Stdout(), 64*1024)
	for {
		line, err := readBoundedLine(reader, maxLine)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				c.lines <- lineResult{err: err}
			}
			return
		}
		c.lines <- lineResult{line: line}
	}
}

var errLineTooLong = errors.New("line too long")

func readBoundedLine(r *bufio.Reader, max int) ([]byte, error) {
	var out []byte
	for {
		chunk, err := r.ReadSlice('\n')
		out = append(out, chunk...)
		if len(out) > max+1 {
			return nil, errLineTooLong
		}
		if err == nil {
			return out, nil
		}
		if !errors.Is(err, bufio.ErrBufferFull) {
			if len(out) > 0 && errors.Is(err, io.EOF) {
				return out, nil
			}
			return nil, err
		}
	}
}

// send writes one line with a deadline, so a bot that stops reading its
// stdin cannot block the worker.
func (c *botConn) send(msg outgoing, timeout time.Duration) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	stdin := c.proc.Stdin()
	if d, ok := stdin.(interface{ SetWriteDeadline(time.Time) error }); ok {
		if err := d.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return err
		}
	}
	_, err = stdin.Write(append(raw, '\n'))
	return err
}

func (c *botConn) fail(cause string) {
	if c.cause == "" {
		c.cause = cause
	}
	c.shutdown()
}

func (c *botConn) shutdown() {
	c.closeOnce.Do(func() {
		_ = c.proc.Stdin().Close()
		_ = c.proc.Kill()
		_ = c.proc.Wait()
		// drain the reader so its goroutine exits
		go func() {
			for range c.lines {
			}
		}()
	})
}

// Turn sends each live bot its perception for tick and waits (in parallel,
// with the same deadline for every bot) for its action.
func (s *Session) Turn(ctx context.Context, tick int, perceptions map[string]json.RawMessage) map[string]engine.PlayerActionInput {
	timeout := time.Duration(s.limits.TurnTimeoutMs) * time.Millisecond
	if !s.started {
		timeout = time.Duration(s.limits.InitTimeoutMs+s.limits.TurnTimeoutMs) * time.Millisecond
		s.started = true
	}
	out := make(map[string]engine.PlayerActionInput, len(s.order))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, id := range s.order {
		conn := s.bots[id]
		wg.Add(1)
		go func() {
			defer wg.Done()
			input := s.turn(ctx, conn, tick, perceptions[conn.playerID], timeout)
			mu.Lock()
			out[conn.playerID] = input
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func disqualified(cause string) engine.PlayerActionInput {
	return engine.PlayerActionInput{Status: engine.ActionStatusDisqualified, ErrorDetails: cause}
}

func (s *Session) turn(ctx context.Context, c *botConn, tick int, perception json.RawMessage, timeout time.Duration) engine.PlayerActionInput {
	if c.cause != "" {
		return disqualified(c.cause)
	}
	if len(perception) == 0 {
		// The engine sends no perception to a player that is out of the
		// simulation (e.g. destroyed); the bot is not contacted this tick.
		return engine.PlayerActionInput{Status: engine.ActionStatusInactive}
	}
	t := tick
	if err := c.send(outgoing{Type: "perception", Tick: &t, Perception: perception}, timeout); err != nil {
		c.fail(CauseCrash)
		return disqualified(CauseCrash)
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var res lineResult
	var open bool
	select {
	case res, open = <-c.lines:
	case <-timer.C:
		c.fail(CauseTimeout)
		return disqualified(CauseTimeout)
	case <-ctx.Done():
		c.fail(CauseCrash)
		return disqualified(CauseCrash)
	}
	if !open {
		c.fail(c.exitCause())
		return disqualified(c.cause)
	}
	if res.err != nil {
		c.fail(CauseProtocol)
		return disqualified(CauseProtocol)
	}
	c.outputBytes += int64(len(res.line))
	if c.outputBytes > int64(s.limits.MaxOutputKB)<<10 {
		c.fail(CauseResourceLimit)
		return disqualified(CauseResourceLimit)
	}
	var msg incoming
	if err := json.Unmarshal(res.line, &msg); err != nil || msg.Type != "action" || msg.Tick == nil {
		c.fail(CauseProtocol)
		return disqualified(CauseProtocol)
	}
	if *msg.Tick != tick {
		c.fail(CauseProtocol)
		return disqualified(CauseProtocol)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(msg.Action, &obj); err != nil || obj == nil {
		c.fail(CauseInvalidAction)
		return disqualified(CauseInvalidAction)
	}
	return engine.PlayerActionInput{Status: engine.ActionStatusValid, Payload: append(json.RawMessage(nil), msg.Action...)}
}

// exitCause distinguishes resource exhaustion (limit signals, or the
// interpreter reporting it) from ordinary crashes.
func (c *botConn) exitCause() string {
	if rk, ok := c.proc.(interface{ ResourceKilled(time.Duration) bool }); ok && rk.ResourceKilled(500*time.Millisecond) {
		return CauseResourceLimit
	}
	stderr := c.proc.Stderr()
	for _, marker := range []string{"MemoryError", "Resource temporarily unavailable", "Cannot allocate memory", "File size limit"} {
		if strings.Contains(stderr, marker) {
			return CauseResourceLimit
		}
	}
	return CauseCrash
}

// Disqualifications returns the cause per player that failed.
func (s *Session) Disqualifications() map[string]string {
	out := map[string]string{}
	for id, c := range s.bots {
		if c.cause != "" {
			out[id] = c.cause
		}
	}
	return out
}

// StderrTail returns the bounded stderr of a player (for admission errors).
func (s *Session) StderrTail(playerID string) string {
	if c := s.bots[playerID]; c != nil {
		return c.proc.Stderr()
	}
	return ""
}

// Close sends end to live bots and terminates every process.
func (s *Session) Close(winner, reason string) {
	ids := append([]string(nil), s.order...)
	sort.Strings(ids)
	for _, id := range ids {
		c := s.bots[id]
		if c.cause == "" {
			w := winner
			_ = c.send(outgoing{Type: "end", MatchID: s.matchID, Winner: &w, Reason: reason}, 200*time.Millisecond)
		}
		c.shutdown()
	}
}
