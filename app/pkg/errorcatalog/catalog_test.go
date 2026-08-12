package errorcatalog

import (
	"testing"

	"goshop/gmicro/errcode"
	"goshop/pkg/errors"
)

func TestCatalogIsValidAndContainsFrameworkContracts(t *testing.T) {
	if err := Catalog.Validate(); err != nil {
		t.Fatalf("Catalog.Validate() error = %v", err)
	}

	for _, code := range []int{errcode.ErrValidation, errcode.ErrConflict, errcode.ErrServiceUnavailable, errcode.ErrTimeout} {
		found := false
		for _, spec := range Catalog {
			if spec.Code == code {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Catalog is missing framework code %d", code)
		}
	}
}

func TestNewValidationErrorPreservesReviewedMessage(t *testing.T) {
	RegisterAll()
	err := NewValidationError("mobile is required")
	spec, ok := errors.SpecOf(err)
	if !ok {
		t.Fatal("SpecOf() found no specification")
	}
	if got, want := spec.Message, "mobile is required"; got != want {
		t.Fatalf("public validation message = %q, want %q", got, want)
	}
}
