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
)

type Sandbox interface {
	ValidateBot(ctx context.Context, codePath string) error
	StartSession(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error)
}

type agentSandbox struct {
	timeout time.Duration
}

func NewSandbox(timeout time.Duration) Sandbox {
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	return &agentSandbox{timeout: timeout}
}
