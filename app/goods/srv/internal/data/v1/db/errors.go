package db

import (
	"goshop/pkg/errcode"
	"goshop/pkg/errors"
)

// wrapDatabaseError preserves a storage failure while exposing the stable database error contract.
func wrapDatabaseError(err error, operation string) error {
	return errors.WrapCode(err, errcode.ErrDatabase, operation)
}
