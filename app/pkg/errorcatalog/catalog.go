// Package errorcatalog registers the error contracts required by goshop
// applications. Keep this at the composition root: domain packages only
// declare their catalogs and never mutate the global registry during import.
package errorcatalog

import (
	"goshop/app/pkg/bizcode"
	"goshop/gmicro/errcode"
)

// RegisterAll adds framework and business error contracts to the shared
// registry. It is idempotent and should be called during application startup.
func RegisterAll() {
	errcode.RegisterAll()
	bizcode.RegisterAll()
}
