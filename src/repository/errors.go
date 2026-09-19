package repository

import "errors"

var (
	ErrFencingTokenMismatch  = errors.New("fencing token mismatch: job lease superseded or invalid")
	ErrMatchAlreadyCommitted = errors.New("match run already committed")
	ErrJobNotFound           = errors.New("match job not found")
	ErrMatchNotFound         = errors.New("match not found")
)
