package v1

import (
	"context"
	"testing"

	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"
)

func TestUserServiceDeleteRejectsZeroID(t *testing.T) {
	svc := NewUserService(&fakeUserStore{})

	err := svc.Delete(context.Background(), 0)
	if !errors.IsCode(err, errcode.ErrValidation) {
		t.Fatalf("Delete() error = %v, want ErrValidation", err)
	}
}

func TestUserServiceDeleteRequestsReversibleDeletion(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{},
	}
	svc := NewUserService(store)

	if err := svc.Delete(context.Background(), 9); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.deletionRequestedID != 9 {
		t.Fatalf("deletion request id = %d, want 9", store.deletionRequestedID)
	}
	if store.deletedID != 0 {
		t.Fatalf("deleted id = %d, want no physical deletion", store.deletedID)
	}
}
