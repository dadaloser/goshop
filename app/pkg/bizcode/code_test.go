package bizcode_test

import (
	"goshop/app/pkg/errorcatalog"
	"testing"
)

func TestCatalogIsValid(t *testing.T) {
	if err := errorcatalog.Catalog.Validate(); err != nil {
		t.Fatalf("Catalog.Validate() error = %v", err)
	}
}
