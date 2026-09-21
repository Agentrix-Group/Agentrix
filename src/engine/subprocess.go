package engine

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// Write keeps at most limit bytes but always reports the full length:
// returning a short count would make io.Copy stop draining the engine's
// stderr, and the engine would block once the pipe buffer fills.
func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		keep := p
		if len(keep) > remaining {
			keep = keep[:remaining]
		}
		b.buf.Write(keep)
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// ClientLifecycleState tracks the supervisor engine client lifecycle.
type ClientLifecycleState int

const (
	ClientStateCreated ClientLifecycleState = iota
	ClientStateStarted
	ClientStateInitialized
	ClientStateRunning
	ClientStateFinished
	ClientStateClosed
)

type subprocessClient struct {
	mu                 sync.Mutex
	cfg                StartConfig
	cmd                *exec.Cmd
	stdin              io.WriteCloser
	stdoutReader       *bufio.Reader
	stderrBuf          *limitedBuffer
	sendSeq            uint64
	expectedRecvSeq    uint64
	matchID            string
	lifecycle          ClientLifecycleState
	engineVersion      string
	engineDigest       string
	engineReadyPayload *EngineReadyPayload
}

// NewSubprocessClient creates a new unstarted EngineClient.
func NewSubprocessClient() EngineClient {
	return &subprocessClient{}
}

func (c *subprocessClient) Start(ctx context.Context, cfg StartConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lifecycle != ClientStateCreated {
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

	// Compute binary digest and verify pinned digest if requested
	binFile, err := os.Open(cfg.BinaryPath)
	if err != nil {
		return fmt.Errorf("%w: cannot open %s: %v", ErrExecutableNotFound, cfg.BinaryPath, err)
	}
	h := sha256.New()
	if _, err := io.Copy(h, binFile); err != nil {
		_ = binFile.Close()
		return fmt.Errorf("failed to hash engine binary: %w", err)
	}
	_ = binFile.Close()
	c.engineDigest = hex.EncodeToString(h.Sum(nil))

	if cfg.ExpectedDigest != "" && c.engineDigest != cfg.ExpectedDigest {
		return fmt.Errorf("%w: expected %s, got %s", ErrEngineDigestMismatch, cfg.ExpectedDigest, c.engineDigest)
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

	var readyPayload EngineReadyPayload
	if err := fromMap(env.Payload, &readyPayload); err != nil {
		c.forceKill()
		return fmt.Errorf("failed to decode engine_ready payload: %w", err)
	}

	protocolSupported := false
	for _, proto := range readyPayload.SupportedProtocols {
		if proto == ProtocolVersion {
			protocolSupported = true
			break
		}
	}
	if !protocolSupported {
		c.forceKill()
		return fmt.Errorf("%w: engine supported protocols %v do not include %s", ErrIncompatibleVersion, readyPayload.SupportedProtocols, ProtocolVersion)
	}

	c.engineVersion = readyPayload.EngineVersion
	c.engineReadyPayload = &readyPayload
	c.lifecycle = ClientStateStarted
	return nil
}

func (c *subprocessClient) InitializeMatch(ctx context.Context, req InitializeMatchRequest) (*MatchInitializedResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lifecycle == ClientStateCreated || c.lifecycle == ClientStateClosed {
		return nil, ErrEngineNotStarted
	}
	if c.lifecycle != ClientStateStarted {
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

	if result.MatchID != req.MatchID {
		return nil, fmt.Errorf("%w: payload matchId %s does not match requested %s", ErrMatchIDMismatch, result.MatchID, req.MatchID)
	}

	c.lifecycle = ClientStateInitialized
	return &result, nil
}

func (c *subprocessClient) AdvanceTick(ctx context.Context, req AdvanceTickRequest) (*TickResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lifecycle == ClientStateCreated || c.lifecycle == ClientStateClosed {
		return nil, ErrEngineNotStarted
	}
	if c.lifecycle != ClientStateInitialized && c.lifecycle != ClientStateRunning {
		if c.lifecycle == ClientStateStarted {
			return nil, ErrMatchNotInitialized
		}
		return nil, ErrInvalidLifecycleTransition
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
		c.lifecycle = ClientStateFinished
	} else {
		c.lifecycle = ClientStateRunning
	}

	return &result, nil
}

func (c *subprocessClient) FinishMatch(ctx context.Context, reason string) (*MatchResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lifecycle == ClientStateCreated || c.lifecycle == ClientStateClosed {
		return nil, ErrEngineNotStarted
	}
	if c.lifecycle == ClientStateStarted {
		return nil, ErrMatchNotInitialized
	}
	if c.lifecycle != ClientStateInitialized && c.lifecycle != ClientStateRunning {
		return nil, ErrInvalidLifecycleTransition
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

	c.lifecycle = ClientStateFinished
	return &result, nil
}

func (c *subprocessClient) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lifecycle == ClientStateClosed || c.lifecycle == ClientStateCreated {
		return nil
	}
	c.lifecycle = ClientStateClosed

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

		if c.matchID != "" && env.MatchID != "" && env.MatchID != c.matchID {
			return nil, fmt.Errorf("%w: expected matchId %s, got %s", ErrMatchIDMismatch, c.matchID, env.MatchID)
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

func (c *subprocessClient) EngineVersion() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.engineVersion
}

func (c *subprocessClient) EngineDigest() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.engineDigest
}
