package mysql

import (
	stderrors "errors"
	"testing"

	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
)

func TestWrapDatabaseErrorPreservesCauseAndBusinessCode(t *testing.T) {
	wantCause := stderrors.New("database unavailable")
	err := wrapDatabaseError(wantCause, "adjust inventory")

	if !stderrors.Is(err, wantCause) {
		t.Errorf("wrapDatabaseError() error = %v, want cause %v", err, wantCause)
	}
	if !apperrors.IsCode(err, errcode.ErrDatabase) {
		t.Errorf("wrapDatabaseError() error = %v, want business code %d", err, errcode.ErrDatabase)
	}
}
