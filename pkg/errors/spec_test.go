package errors

import (
	stderrors "errors"
	"fmt"
	"testing"
)

func TestSpecOfFindsSpecInWrappedChain(t *testing.T) {
	spec := Spec{
		Code:      990101,
		Kind:      KindNotFound,
		Message:   "user not found",
		Reference: "https://example.test/errors/user-not-found",
	}

	cause := stderrors.New("record not found")
	err := Wrap(WrapSpec(cause, spec, "query user"), "load profile")

	got, ok := SpecOf(err)
	if !ok {
		t.Fatal("SpecOf() found no specification")
	}
	if got != spec {
		t.Fatalf("SpecOf() = %#v, want %#v", got, spec)
	}
}

func TestParseCoderAndIsCodeFindWrappedCode(t *testing.T) {
	const code = 990102
	Register(defaultCoder{C: code, Ext: "wrapped error"})

	err := WithMessage(NewCode(code, "database query failed"), "load user")
	if got := ParseCoder(err).Code(); got != code {
		t.Fatalf("ParseCoder() code = %d, want %d", got, code)
	}
	if !IsCode(err, code) {
		t.Fatalf("IsCode(%v, %d) = false, want true", err, code)
	}
}

func TestWrapCodeUsesRegisteredSpecAndPreservesCause(t *testing.T) {
	const code = 990103
	Register(defaultCoder{
		C:   code,
		K:   KindUnavailable,
		Ext: "dependency unavailable",
	})

	cause := stderrors.New("dial tcp: connection refused")
	err := WrapCode(cause, code, "connect inventory service")

	if !stderrors.Is(err, cause) {
		t.Fatal("WrapCode() does not preserve the cause")
	}
	if err.Error() != "connect inventory service" {
		t.Fatalf("WrapCode() diagnostic = %q, want %q", err.Error(), "connect inventory service")
	}
	if got := fmt.Sprintf("%v", err); got != "dependency unavailable" {
		t.Fatalf("WrapCode() public message = %q, want %q", got, "dependency unavailable")
	}
	if spec, ok := SpecOf(err); !ok || spec != SpecForCode(code) {
		t.Fatalf("SpecOf(WrapCode()) = (%#v, %t), want (%#v, true)", spec, ok, SpecForCode(code))
	}
}
