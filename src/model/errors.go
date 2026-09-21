package model

import (
	"errors"
	"fmt"
)

// Kind classifies domain errors so that the HTTP layer maps them to a stable
// status code. Infrastructure failures are plain errors and map to 500.
type Kind string

const (
	KindValidation   Kind = "validation"         // 422: well-formed request that violates a rule
	KindBadRequest   Kind = "bad_request"        // 400: malformed request
	KindNotFound     Kind = "not_found"          // 404
	KindConflict     Kind = "conflict"           // 409: uniqueness, concurrent change, idempotency mismatch
	KindTransition   Kind = "invalid_transition" // 409: state machine rejects the command
	KindForbidden    Kind = "forbidden"          // 403: authenticated but not allowed
	KindUnauthorized Kind = "unauthorized"       // 401: missing or invalid credentials
	KindUnavailable  Kind = "unavailable"        // 503: a required dependency is not ready (e.g. no engine artifact)
)

// Error is a domain error with a safe, user-facing message.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	// Details carries safe structured data for the client (e.g. the id of
	// the run that is already in progress).
	Details map[string]any
	Err     error
}

// WithDetails returns a copy of a domain error with structured details.
func WithDetails(err error, details map[string]any) error {
	domain, ok := AsError(err)
	if !ok {
		return err
	}
	copy := *domain
	copy.Details = details
	return &copy
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func newError(kind Kind, code, format string, args ...any) *Error {
	return &Error{Kind: kind, Code: code, Message: fmt.Sprintf(format, args...)}
}

func Validation(code, format string, args ...any) error {
	return newError(KindValidation, code, format, args...)
}
func BadRequest(code, format string, args ...any) error {
	return newError(KindBadRequest, code, format, args...)
}
func NotFound(code, format string, args ...any) error {
	return newError(KindNotFound, code, format, args...)
}
func Conflict(code, format string, args ...any) error {
	return newError(KindConflict, code, format, args...)
}
func Forbidden(code, format string, args ...any) error {
	return newError(KindForbidden, code, format, args...)
}
func Unauthorized(code, format string, args ...any) error {
	return newError(KindUnauthorized, code, format, args...)
}
func Unavailable(code, format string, args ...any) error {
	return newError(KindUnavailable, code, format, args...)
}

// InvalidTransition reports a state machine rejection.
func InvalidTransition(entity string, from, to any) error {
	return newError(KindTransition, "invalid_transition", "%s cannot transition from %q to %q", entity, from, to)
}

// KindOf returns the domain kind of err, or "" for infrastructure errors.
func KindOf(err error) Kind {
	var domain *Error
	if errors.As(err, &domain) {
		return domain.Kind
	}
	return ""
}

// AsError returns the domain error wrapped in err, if any.
func AsError(err error) (*Error, bool) {
	var domain *Error
	ok := errors.As(err, &domain)
	return domain, ok
}
