package db

import (
	stderrors "errors"
	"testing"

	"goshop/app/pkg/bizcode"
	"goshop/app/pkg/options"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
)

func TestGetDBFactoryOrReturnsPersistedInitializationError(t *testing.T) {
	resetUserFactory()
	t.Cleanup(resetUserFactory)

	wantCause := stderrors.New("database unavailable")
	once.Do(func() {
		dbFactoryInitErr = wantCause
	})

	factory, err := GetDBFactoryOr(&options.MySQLOptions{})
	if factory != nil {
		t.Errorf("GetDBFactoryOr() factory = %T, want nil", factory)
	}
	if !stderrors.Is(err, wantCause) {
		t.Errorf("GetDBFactoryOr() error = %v, want cause %v", err, wantCause)
	}
	if !apperrors.IsCode(err, bizcode.ErrConnectDB) {
		t.Errorf("GetDBFactoryOr() error = %v, want business code %d", err, bizcode.ErrConnectDB)
	}
}

func TestWrapDatabaseErrorPreservesCauseAndBusinessCode(t *testing.T) {
	wantCause := stderrors.New("database unavailable")
	err := wrapDatabaseError(wantCause, "query user")

	if !stderrors.Is(err, wantCause) {
		t.Errorf("wrapDatabaseError() error = %v, want cause %v", err, wantCause)
	}
	if !apperrors.IsCode(err, errcode.ErrDatabase) {
		t.Errorf("wrapDatabaseError() error = %v, want business code %d", err, errcode.ErrDatabase)
	}
}
