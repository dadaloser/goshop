package errors

import (
	stderrors "errors"
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

func TestParseCoderAndIsCodeFindWrappedLegacyCode(t *testing.T) {
	const code = 990102
	Register(defaultCoder{C: code, Ext: "wrapped error"})

	err := WithMessage(WithCode(code, "database query failed"), "load user")
	if got := ParseCoder(err).Code(); got != code {
		t.Fatalf("ParseCoder() code = %d, want %d", got, code)
	}
	if !IsCode(err, code) {
		t.Fatalf("IsCode(%v, %d) = false, want true", err, code)
	}
}
