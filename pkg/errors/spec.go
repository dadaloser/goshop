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

// Validate reports whether Spec is a complete public error contract.
func (spec Spec) Validate() error {
	if spec.Code <= 0 {
		return fmt.Errorf("error code must be greater than zero")
	}
	if !spec.Kind.valid() {
		return fmt.Errorf("error kind %q is invalid", spec.Kind)
	}
	if spec.Message == "" {
		return fmt.Errorf("error message must not be empty")
	}
	return nil
}

func (kind Kind) valid() bool {
	switch kind {
	case KindInvalidArgument, KindUnauthenticated, KindPermissionDenied, KindNotFound,
		KindConflict, KindRateLimited, KindUnavailable, KindTimeout, KindInternal:
		return true
	default:
		return false
	}
}

// Coded is implemented by errors that carry a business error specification.
type Coded interface {
	error
	Spec() Spec
}

// NewCode returns an error using the public specification registered for code.
// New business code should prefer a named Spec and NewSpec; this helper keeps
// existing numeric-code APIs on the same transport-independent contract.
func NewCode(code int, internal string) error {
	return NewSpec(SpecForCode(code), internal)
}

// WrapCode attaches the public specification registered for code to err while
// preserving err in the standard Go error chain. It returns nil when err is nil.
func WrapCode(err error, code int, internal string) error {
	return wrapSpec(err, SpecForCode(code), internal, callersSkip(0))
}

// SpecForCode returns the public specification registered for code. Unknown
// codes resolve to the internal-error specification and never expose the
// caller-supplied numeric code to clients.
func SpecForCode(code int) Spec {
	coder, ok := lookupCoder(code)
	if !ok {
		coder = unknownCoder
	}

	spec := Spec{
		Code:      coder.Code(),
		Kind:      coder.Kind(),
		Message:   coder.String(),
		Reference: coder.Reference(),
	}
	return spec
}

// NewSpec returns an error with a public business-error contract and internal
// diagnostic text. Error always returns the public message; diagnostics are
// available only through explicit detailed formatting for controlled logs.
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
	return wrapSpec(err, spec, internal, callers())
}

func wrapSpec(err error, spec Spec, internal string, stack *stack) error {
	if err == nil {
		return nil
	}

	return &withSpec{
		spec:  spec,
		err:   stderrors.New(internal),
		cause: err,
		stack: stack,
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

// Error returns only the public error message. It must be safe to expose at
// every boundary, including paths that accidentally bypass a transport adapter.
func (w *withSpec) Error() string {
	if w.spec.Message != "" {
		return w.spec.Message
	}
	return unknownCoder.String()
}

func (w *withSpec) diagnostic() string { return w.err.Error() }

func (w *withSpec) Unwrap() error { return w.cause }

func (w *withSpec) Spec() Spec { return w.spec }
