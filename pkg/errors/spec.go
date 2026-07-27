package errors

import (
	stderrors "errors"
	"fmt"
)

// Kind classifies an error independently of a transport protocol.
type Kind string

const (
	KindInvalidArgument  Kind = "invalid_argument"
	KindUnauthenticated  Kind = "unauthenticated"
	KindPermissionDenied Kind = "permission_denied"
	KindNotFound         Kind = "not_found"
	KindConflict         Kind = "conflict"
	KindRateLimited      Kind = "rate_limited"
	KindUnavailable      Kind = "unavailable"
	KindTimeout          Kind = "timeout"
	KindInternal         Kind = "internal"
)

// Spec is the transport-independent public contract of a business error.
// Message and Reference may be exposed to callers; internal diagnostic text
// belongs in the error message passed to NewSpec or WrapSpec.
type Spec struct {
	Code      int
	Kind      Kind
	Message   string
	Reference string
}

// Coded is implemented by errors that carry a business error specification.
type Coded interface {
	error
	Spec() Spec
}

// NewSpec returns an error with a public business-error contract and internal
// diagnostic text. The diagnostic text is retained for logs and must not be
// returned directly to clients.
func NewSpec(spec Spec, internal string) error {
	return &withSpec{
		spec:  spec,
		err:   stderrors.New(internal),
		stack: callers(),
	}
}

// WrapSpec attaches a public business-error contract to err and preserves err
// in the standard Go error chain. It returns nil when err is nil.
func WrapSpec(err error, spec Spec, internal string) error {
	if err == nil {
		return nil
	}

	return &withSpec{
		spec:  spec,
		err:   stderrors.New(internal),
		cause: err,
		stack: callers(),
	}
}

// SpecOf returns the first business-error specification in err's chain.
func SpecOf(err error) (Spec, bool) {
	var coded Coded
	if !stderrors.As(err, &coded) {
		return Spec{}, false
	}

	return coded.Spec(), true
}

type withSpec struct {
	spec  Spec
	err   error
	cause error
	*stack
}

func (w *withSpec) Error() string { return w.err.Error() }

func (w *withSpec) Unwrap() error { return w.cause }

func (w *withSpec) Spec() Spec { return w.spec }

func (w *withSpec) Format(state fmt.State, verb rune) {
	if verb != 'v' || (!state.Flag('+') && !state.Flag('#') && !state.Flag('-')) {
		_, _ = fmt.Fprint(state, w.spec.Message)
		return
	}

	_, _ = fmt.Fprint(state, w.Error())
	if w.cause != nil {
		_, _ = fmt.Fprintf(state, ": %+v", w.cause)
	}
	if state.Flag('+') && w.stack != nil {
		w.stack.Format(state, verb)
	}
}

func (w *withCode) Spec() Spec {
	coder := ParseCoder(w)
	return Spec{
		Code:      coder.Code(),
		Message:   coder.String(),
		Reference: coder.Reference(),
	}
}
