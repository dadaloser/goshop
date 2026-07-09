package v1

import (
	"context"
	"testing"

	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestUserServiceDeleteRejectsZeroID(t *testing.T) {
	svc := NewUserService(&fakeUserStore{})

	err := svc.Delete(context.Background(), 0)
	if !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("Delete() error = %v, want ErrValidation", err)
	}
}

func TestUserServiceDeleteCallsStore(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{},
	}
	svc := NewUserService(store)

	if err := svc.Delete(context.Background(), 9); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.deletedID != 9 {
		t.Fatalf("deleted id = %d, want 9", store.deletedID)
	}
}
