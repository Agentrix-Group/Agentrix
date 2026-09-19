package executor

import (
	"context"
	"errors"
	"time"
)

var (
	ErrAgentUnavailable   = errors.New("agent unavailable")
	ErrAgentTimeout       = errors.New("agent timeout")
	ErrAgentExecution     = errors.New("agent execution failed")
	ErrAgentInvalidAction = errors.New("agent action invalid")
	ErrSandboxUnavailable = errors.New("sandbox runtime unavailable (fail-closed)")
)

type Sandbox interface {
	ValidateBot(ctx context.Context, codePath string) error
	StartSession(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error)
}

type agentSandbox struct {
	timeout time.Duration
	runtime BotRuntime
}

func NewSandbox(timeout time.Duration, runtime ...BotRuntime) Sandbox {
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	var rt BotRuntime
	if len(runtime) > 0 && runtime[0] != nil {
		rt = runtime[0]
	} else {
		rt = DefaultBotRuntime()
	}
	return &agentSandbox{timeout: timeout, runtime: rt}
}
