// Package errorcatalog registers the error contracts required by goshop
// applications. Keep this at the composition root: domain packages only
// declare their catalogs and never mutate the global registry during import.
package errorcatalog

import (
	"goshop/app/pkg/bizcode"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"
)

// Catalog is the complete application error catalog. It is assembled once at
// the composition root so framework and business code have one public source
// of truth.
var Catalog = append(frameworkCatalog(), bizcode.Catalog...)

func frameworkCatalog() errors.Catalog {
	catalog := make(errors.Catalog, 0, len(errcode.Catalog))
	for _, contract := range errcode.Catalog {
		catalog = append(catalog, errors.Spec{
			Code:    contract.Code,
			Kind:    errors.Kind(contract.Kind),
			Message: contract.Message,
		})
	}
	return catalog
}

// RegisterAll adds framework and business error contracts to the shared
// registry. It is idempotent and should be called during application startup.
func RegisterAll() {
	Catalog.RegisterAll()
}

// NewValidationError creates a reviewed, request-specific validation response.
// It is the only supported escape hatch from the stable catalog message.
func NewValidationError(message string) error {
	return errors.NewPublicSpec(errors.Spec{
		Code:    errcode.ErrValidation,
		Kind:    errors.KindInvalidArgument,
		Message: message,
	}, message)
}
