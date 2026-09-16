package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type limitedBuffer struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type subprocessClient struct {
	mu              sync.Mutex
	cfg             StartConfig
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdoutReader    *bufio.Reader
	stderrBuf       *limitedBuffer
	sendSeq         uint64
	expectedRecvSeq uint64
	matchID         string
	started         bool
	closed          bool
}

// NewSubprocessClient creates a new unstarted EngineClient.
func NewSubprocessClient() EngineClient {
	return &subprocessClient{}
}

func (c *subprocessClient) Start(ctx context.Context, cfg StartConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("engine already started")
	}

	if cfg.BinaryPath == "" {
		return fmt.Errorf("%w: path is empty", ErrExecutableNotFound)
	}

	if _, err := os.Stat(cfg.BinaryPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrExecutableNotFound, cfg.BinaryPath)
		}
	}

	if cfg.MaxLineBytes <= 0 {
		cfg.MaxLineBytes = 1048576 // 1 MB default
	}
	if cfg.StderrLimitBytes <= 0 {
		cfg.StderrLimitBytes = 65536 // 64 KB default
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 5 * time.Second
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 3 * time.Second
	}

	c.cfg = cfg
	c.cmd = exec.Command(cfg.BinaryPath, cfg.Args...)
	if cfg.WorkingDir != "" {
		c.cmd.Dir = cfg.WorkingDir
	}
	if len(cfg.Env) > 0 {
		c.cmd.Env = append(os.Environ(), cfg.Env...)
	}

	stdinPipe, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProcessStartFailed, err)
	}
	c.stdin = stdinPipe

	stdoutPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		_ = stdinPipe.Close()
		return fmt.Errorf("%w: %v", ErrProcessStartFailed, err)
	}
	c.stdoutReader = bufio.NewReaderSize(stdoutPipe, cfg.MaxLineBytes)

	stderrPipe, err := c.cmd.StderrPipe()
	if err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		return fmt.Errorf("%w: %v", ErrProcessStartFailed, err)
	}

	c.stderrBuf = &limitedBuffer{limit: cfg.StderrLimitBytes}
	go func() {
		_, _ = io.Copy(c.stderrBuf, stderrPipe)
	}()

	if err := c.cmd.Start(); err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		return fmt.Errorf("%w: %v", ErrProcessStartFailed, err)
	}

	c.sendSeq = 1
	c.expectedRecvSeq = 1
	c.started = true

	// Wait for engine_ready message with timeout
	handshakeCtx, cancel := context.WithTimeout(ctx, cfg.HandshakeTimeout)
	defer cancel()

	env, err := c.readEnvelopeInternal(handshakeCtx)
	if err != nil {
		c.forceKill()
		return fmt.Errorf("failed waiting for engine_ready: %w", err)
	}

	if env.Type != TypeEngineReady {
		c.forceKill()
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedMessageType, TypeEngineReady, env.Type)
	}

	return nil
}

func (c *subprocessClient) InitializeMatch(ctx context.Context, req InitializeMatchRequest) (*MatchInitializedResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started || c.closed {
		return nil, ErrEngineNotStarted
	}
	if c.matchID != "" {
		return nil, ErrMatchAlreadyStarted
	}

	c.matchID = req.MatchID

	payloadMap, err := toMap(req)
	if err != nil {
		return nil, err
	}

	if err := c.writeEnvelopeInternal(TypeInitializeMatch, req.MatchID, payloadMap); err != nil {
		return nil, err
	}

	env, err := c.readEnvelopeInternal(ctx)
	if err != nil {
		return nil, err
	}

	if env.Type != TypeMatchInitialized {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedMessageType, TypeMatchInitialized, env.Type)
	}

	var result MatchInitializedResult
	if err := fromMap(env.Payload, &result); err != nil {
		return nil, fmt.Errorf("failed to decode match_initialized payload: %w", err)
	}

	return &result, nil
}

func (c *subprocessClient) AdvanceTick(ctx context.Context, req AdvanceTickRequest) (*TickResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started || c.closed {
		return nil, ErrEngineNotStarted
	}
	if c.matchID == "" {
		return nil, ErrMatchNotInitialized
	}

	payloadMap, err := toMap(req)
	if err != nil {
		return nil, err
	}

	if err := c.writeEnvelopeInternal(TypeAdvanceTick, c.matchID, payloadMap); err != nil {
		return nil, err
	}

	env, err := c.readEnvelopeInternal(ctx)
	if err != nil {
		return nil, err
	}

	if env.Type != TypeTickCompleted && env.Type != TypeMatchCompleted {
		return nil, fmt.Errorf("%w: expected %s or %s, got %s", ErrUnexpectedMessageType, TypeTickCompleted, TypeMatchCompleted, env.Type)
	}

	var result TickResult
	if err := fromMap(env.Payload, &result); err != nil {
		return nil, fmt.Errorf("failed to decode tick_completed payload: %w", err)
	}

	if env.Type == TypeMatchCompleted {
		result.IsOver = true
	}

	return &result, nil
}

func (c *subprocessClient) FinishMatch(ctx context.Context, reason string) (*MatchResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started || c.closed {
		return nil, ErrEngineNotStarted
	}
	if c.matchID == "" {
		return nil, ErrMatchNotInitialized
	}

	payloadMap := map[string]interface{}{
		"reason": reason,
	}

	if err := c.writeEnvelopeInternal(TypeFinishMatch, c.matchID, payloadMap); err != nil {
		return nil, err
	}

	env, err := c.readEnvelopeInternal(ctx)
	if err != nil {
		return nil, err
	}

	if env.Type != TypeMatchCompleted {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedMessageType, TypeMatchCompleted, env.Type)
	}

	var result MatchResult
	if err := fromMap(env.Payload, &result); err != nil {
		return nil, fmt.Errorf("failed to decode match_completed payload: %w", err)
	}

	return &result, nil
}

func (c *subprocessClient) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started || c.closed {
		return nil
	}
	c.closed = true

	// Send shutdown command
	_ = c.writeEnvelopeInternal(TypeShutdown, "", map[string]interface{}{
		"reason": "close",
	})

	// Wait up to ShutdownTimeout for shutdown_ack or process exit
	shutdownCtx, cancel := context.WithTimeout(ctx, c.cfg.ShutdownTimeout)
	defer cancel()

	ackChan := make(chan error, 1)
	go func() {
		for {
			env, err := c.readEnvelopeInternal(shutdownCtx)
			if err != nil {
				ackChan <- err
				return
			}
			if env.Type == TypeShutdownAck {
				ackChan <- nil
				return
			}
		}
	}()

	select {
	case <-shutdownCtx.Done():
		c.forceKill()
		return ErrUncleanShutdown
	case <-ackChan:
		// Normal exit
	}

	if c.stdin != nil {
		_ = c.stdin.Close()
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-time.After(1 * time.Second):
		c.forceKill()
		return ErrUncleanShutdown
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUncleanShutdown, err)
		}
		return nil
	}
}

func (c *subprocessClient) forceKill() {
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

func (c *subprocessClient) writeEnvelopeInternal(msgType string, matchID string, payload map[string]interface{}) error {
	env := Envelope{
		ProtocolVersion: ProtocolVersion,
		Type:            msgType,
		MatchID:         matchID,
		Sequence:        c.sendSeq,
		Payload:         payload,
	}
	c.sendSeq++

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	data = append(data, '\n')
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write message to engine stdin: %w", err)
	}
	return nil
}

func (c *subprocessClient) readEnvelopeInternal(ctx context.Context) (*Envelope, error) {
	type readResult struct {
		line []byte
		err  error
	}
	ch := make(chan readResult, 1)

	go func() {
		line, isPrefix, err := c.stdoutReader.ReadLine()
		if isPrefix {
			ch <- readResult{err: ErrLineTooLong}
			return
		}
		if err != nil {
			if err == io.EOF {
				ch <- readResult{err: ErrProcessExited}
				return
			}
			ch <- readResult{err: err}
			return
		}
		// make copy since ReadLine buffer is reused
		buf := make([]byte, len(line))
		copy(buf, line)
		ch <- readResult{line: buf, err: nil}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}

		var env Envelope
		if err := json.Unmarshal(res.line, &env); err != nil {
			return nil, fmt.Errorf("%w: %v (raw: %s)", ErrInvalidJSON, err, string(res.line))
		}

		if env.ProtocolVersion != ProtocolVersion {
			return nil, fmt.Errorf("%w: expected %s, got %s", ErrIncompatibleVersion, ProtocolVersion, env.ProtocolVersion)
		}

		if env.Sequence != c.expectedRecvSeq {
			return nil, fmt.Errorf("%w: expected sequence %d, got %d", ErrInvalidSequence, c.expectedRecvSeq, env.Sequence)
		}
		c.expectedRecvSeq++

		if env.Type == TypeEngineError {
			var engineErr EngineDeclaredError
			_ = fromMap(env.Payload, &engineErr)
			return nil, &engineErr
		}

		return &env, nil
	}
}

func toMap(in interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func fromMap(in map[string]interface{}, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
