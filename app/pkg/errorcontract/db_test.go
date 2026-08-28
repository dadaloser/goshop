package errorcontract

import (
	stderrors "errors"
	"testing"

	"goshop/app/pkg/errorcatalog"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
)

func TestWrapDatabasePreservesCauseAndBusinessCode(t *testing.T) {
	errorcatalog.RegisterAll()
	wantCause := stderrors.New("database unavailable")
	err := WrapDatabase(wantCause, "query orders")

	if !stderrors.Is(err, wantCause) {
		t.Errorf("WrapDatabase() error = %v, want cause %v", err, wantCause)
	}
	if !apperrors.IsCode(err, errcode.ErrDatabase) {
		t.Errorf("WrapDatabase() error = %v, want business code %d", err, errcode.ErrDatabase)
	}
}
