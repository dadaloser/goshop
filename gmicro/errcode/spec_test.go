package errcode

import (
	"testing"
)

func TestCatalogContainsCoreContracts(t *testing.T) {
	contracts := map[int]Contract{}
	for _, contract := range Catalog {
		if _, exists := contracts[contract.Code]; exists {
			t.Fatalf("Catalog declares duplicate code %d", contract.Code)
		}
		contracts[contract.Code] = contract
	}
	for _, code := range []int{ErrValidation, ErrTokenInvalid, ErrPageNotFound, ErrUnknown} {
		if _, exists := contracts[code]; !exists {
			t.Fatalf("Catalog is missing code %d", code)
		}
	}
}
