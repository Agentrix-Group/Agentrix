package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEngineDeclaredError(t *testing.T) {
	r := require.New(t)

	err := &EngineDeclaredError{
		Code:    "ERR_COLLISION_DIVERGENCE",
		Message: "Physics solver reached NaN coordinate",
		Fatal:   true,
	}

	r.Equal("engine error [ERR_COLLISION_DIVERGENCE]: Physics solver reached NaN coordinate (fatal=true)", err.Error())
	r.True(err.Fatal)
}

func TestSentinelErrors(t *testing.T) {
	r := require.New(t)

	sentinelErrors := []error{
		ErrExecutableNotFound,
		ErrProcessStartFailed,
		ErrTimeout,
		ErrProcessExited,
		ErrInvalidJSON,
		ErrIncompatibleVersion,
		ErrInvalidSequence,
		ErrUncleanShutdown,
		ErrLineTooLong,
		ErrUnexpectedMessageType,
		ErrEngineNotStarted,
		ErrMatchAlreadyStarted,
		ErrMatchNotInitialized,
	}

	for _, err := range sentinelErrors {
		r.NotEmpty(err.Error())
		r.True(errors.Is(err, err))
	}
}
