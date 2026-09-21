package engine

import (
	"errors"
	"fmt"
)

var (
	ErrExecutableNotFound    = errors.New("engine executable not found")
	ErrProcessStartFailed    = errors.New("failed to start engine process")
	ErrTimeout               = errors.New("engine operation timed out")
	ErrProcessExited         = errors.New("engine process exited unexpectedly")
	ErrInvalidJSON           = errors.New("invalid JSON received from engine")
	ErrIncompatibleVersion   = errors.New("incompatible engine protocol version")
	ErrInvalidSequence       = errors.New("invalid protocol sequence received")
	ErrUncleanShutdown       = errors.New("engine failed to shut down cleanly")
	ErrLineTooLong           = errors.New("engine stdout line exceeded maximum allowed length")
	ErrUnexpectedMessageType = errors.New("unexpected message type received from engine")
	ErrEngineNotStarted           = errors.New("engine has not been started")
	ErrMatchAlreadyStarted        = errors.New("match has already been initialized")
	ErrMatchNotInitialized        = errors.New("match has not been initialized")
	ErrMatchIDMismatch            = errors.New("engine match ID mismatch")
	ErrEngineDigestMismatch       = errors.New("engine binary digest mismatch")
	ErrInvalidLifecycleTransition = errors.New("invalid engine lifecycle transition")
)

// EngineDeclaredError represents an explicit error emitted by the engine via TypeEngineError.
type EngineDeclaredError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Fatal   bool                   `json:"fatal"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *EngineDeclaredError) Error() string {
	return fmt.Sprintf("engine error [%s]: %s (fatal=%v)", e.Code, e.Message, e.Fatal)
}
