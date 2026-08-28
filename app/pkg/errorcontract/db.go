package errorcontract

import (
	"goshop/pkg/errcode"
	"goshop/pkg/errors"
)

// WrapDatabase preserves a storage failure while exposing the stable database error contract.
func WrapDatabase(err error, operation string) error {
	return errors.WrapCode(err, errcode.ErrDatabase, operation)
}
