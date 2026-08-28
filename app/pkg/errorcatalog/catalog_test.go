package errorcatalog

import (
	"testing"

	"goshop/app/pkg/bizcode"
	"goshop/pkg/errcode"
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

func TestCatalogContainsEveryFrameworkAndBusinessContract(t *testing.T) {
	contracts := make(map[int]errors.Spec, len(Catalog))
	for _, contract := range Catalog {
		contracts[contract.Code] = contract
	}

	wantCount := len(errcode.Catalog) + len(bizcode.Catalog)
	if got := len(contracts); got != wantCount {
		t.Fatalf("Catalog contains %d unique contracts, want %d", got, wantCount)
	}

	for _, contract := range bizcode.Catalog {
		if got, ok := contracts[contract.Code]; !ok || got != contract {
			t.Fatalf("Catalog is missing business contract %s", contract.Message)
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
