package errcode

import (
	"testing"

	apperrors "goshop/pkg/errors"
)

func TestCommonCodesAreRegisteredWithSemanticKinds(t *testing.T) {
	tests := []struct {
		code int
		kind apperrors.Kind
	}{
		{code: ErrValidation, kind: apperrors.KindInvalidArgument},
		{code: ErrTokenInvalid, kind: apperrors.KindUnauthenticated},
		{code: ErrPageNotFound, kind: apperrors.KindNotFound},
		{code: ErrUnknown, kind: apperrors.KindInternal},
	}

	for _, tt := range tests {
		err := apperrors.NewCode(tt.code, "internal detail")
		if got := apperrors.ParseCoder(err).Code(); got != tt.code {
			t.Fatalf("ParseCoder() code = %d, want %d", got, tt.code)
		}
		spec, ok := apperrors.SpecOf(err)
		if !ok {
			t.Fatal("SpecOf() found no specification")
		}
		if spec.Kind != tt.kind {
			t.Fatalf("SpecOf() kind = %q, want %q", spec.Kind, tt.kind)
		}
	}
}

func TestValidationSpecMatchesLegacyCode(t *testing.T) {
	err := apperrors.NewSpec(ValidationSpec, "request payload is required")

	if !apperrors.IsCode(err, ErrValidation) {
		t.Fatal("validation specification did not preserve the legacy code")
	}
	if spec, ok := apperrors.SpecOf(err); !ok || spec.Kind != apperrors.KindInvalidArgument {
		t.Fatalf("SpecOf() = %#v, %t; want invalid_argument specification", spec, ok)
	}
}

func TestNewCodeUsesRegisteredSpecification(t *testing.T) {
	err := apperrors.NewCode(ErrValidation, "request payload is required")

	spec, ok := apperrors.SpecOf(err)
	if !ok {
		t.Fatal("SpecOf() found no specification")
	}
	if spec.Code != ErrValidation || spec.Kind != apperrors.KindInvalidArgument {
		t.Fatalf("SpecOf() = %#v, want validation specification", spec)
	}
}
