package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
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

func TestSpecOfFindsSpecInAggregate(t *testing.T) {
	spec := Spec{Code: 990107, Kind: KindNotFound, Message: "user not found"}
	err := NewAggregate([]error{stderrors.New("other failure"), NewSpec(spec, "query user")})

	got, ok := SpecOf(err)
	if !ok || got != spec {
		t.Fatalf("SpecOf(Aggregate) = (%#v, %t), want (%#v, true)", got, ok, spec)
	}
}

func TestSpecOfStopsOnCyclicError(t *testing.T) {
	if _, ok := SpecOf(cyclicError{}); ok {
		t.Fatal("SpecOf() found a specification in cyclic error")
	}
	if got := ParseCoder(cyclicError{}); got != unknownCoder {
		t.Fatalf("ParseCoder(cyclic error) = %#v, want unknown coder", got)
	}
}

func TestSpecValidate(t *testing.T) {
	tests := []struct {
		name string
		spec Spec
	}{
		{name: "zero code", spec: Spec{Kind: KindInternal, Message: "internal error"}},
		{name: "invalid kind", spec: Spec{Code: 990108, Kind: "invalid", Message: "invalid"}},
		{name: "empty message", spec: Spec{Code: 990109, Kind: KindInternal}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.spec.Validate(); err == nil {
				t.Fatal("Spec.Validate() = nil, want validation error")
			}
		})
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
	if err.Error() != "dependency unavailable" {
		t.Fatalf("WrapCode() public message = %q, want %q", err.Error(), "dependency unavailable")
	}
	if diagnostic := fmt.Sprintf("%+v", err); !strings.Contains(diagnostic, "connect inventory service") {
		t.Fatalf("WrapCode() detailed format = %q, want diagnostic", diagnostic)
	}
	if spec, ok := SpecOf(err); !ok || spec != SpecForCode(code) {
		t.Fatalf("SpecOf(WrapCode()) = (%#v, %t), want (%#v, true)", spec, ok, SpecForCode(code))
	}
}

func TestDetailedFormatIncludesJoinedBranches(t *testing.T) {
	left := NewSpec(Spec{Code: 990104, Kind: KindUnavailable, Message: "left unavailable"}, "left diagnostic")
	right := NewSpec(Spec{Code: 990105, Kind: KindUnavailable, Message: "right unavailable"}, "right diagnostic")
	err := WrapSpec(stderrors.Join(left, right), Spec{Code: 990106, Kind: KindUnavailable, Message: "combined unavailable"}, "combined diagnostic")

	detail := fmt.Sprintf("%+v", err)
	for _, want := range []string{"combined diagnostic", "left diagnostic", "right diagnostic"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detailed format = %q, missing %q", detail, want)
		}
	}
}
