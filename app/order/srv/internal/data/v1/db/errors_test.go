package db

import (
	stderrors "errors"
	"testing"

	"goshop/app/pkg/bizcode"
	"goshop/app/pkg/options"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
)

func TestGetDataFactoryOrReturnsPersistedInitializationError(t *testing.T) {
	resetOrderFactory()
	t.Cleanup(resetOrderFactory)

	wantCause := stderrors.New("database unavailable")
	once.Do(func() {
		dataFactoryInitErr = wantCause
	})

	factory, err := GetDataFactoryOr(&options.MySQLOptions{})
	if factory != nil {
		t.Errorf("GetDataFactoryOr() factory = %T, want nil", factory)
	}
	if !stderrors.Is(err, wantCause) {
		t.Errorf("GetDataFactoryOr() error = %v, want cause %v", err, wantCause)
	}
	if !apperrors.IsCode(err, bizcode.ErrConnectDB) {
		t.Errorf("GetDataFactoryOr() error = %v, want business code %d", err, bizcode.ErrConnectDB)
	}
}

func TestWrapDatabaseErrorPreservesCauseAndBusinessCode(t *testing.T) {
	wantCause := stderrors.New("database unavailable")
	err := wrapDatabaseError(wantCause, "query orders")

	if !stderrors.Is(err, wantCause) {
		t.Errorf("wrapDatabaseError() error = %v, want cause %v", err, wantCause)
	}
	if !apperrors.IsCode(err, errcode.ErrDatabase) {
		t.Errorf("wrapDatabaseError() error = %v, want business code %d", err, errcode.ErrDatabase)
	}
}
