package v1

import (
	"context"
	"testing"
	"time"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestLogoutAllBumpsTokenVersion(t *testing.T) {
	versions := &fakeTokenVersionStore{}
	svc := NewUserService(
		&fakeDataFactory{users: &fakeUserData{}},
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		nil,
		nil,
		nil,
		versions,
	)

	if err := svc.LogoutAll(context.Background(), 99); err != nil {
		t.Fatalf("LogoutAll() error = %v", err)
	}
	if versions.bumpUserID != 99 {
		t.Fatalf("bumped user id = %d, want 99", versions.bumpUserID)
	}
}

func TestDeleteAccountValidatesPasswordAndDeletesUser(t *testing.T) {
	users := &fakeUserData{
		user: data.User{
			ID:       7,
			NickName: "tester",
			PassWord: "hashed",
		},
	}
	versions := &fakeTokenVersionStore{}
	svc := NewUserService(
		&fakeDataFactory{users: users},
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		nil,
		nil,
		nil,
		versions,
	)

	if err := svc.DeleteAccount(context.Background(), 7, "secret"); err != nil {
		t.Fatalf("DeleteAccount() error = %v", err)
	}
	if !users.deleteCalled {
		t.Fatal("DeleteAccount() did not delete user")
	}
	if users.deletedUserID != 7 {
		t.Fatalf("deleted user id = %d, want 7", users.deletedUserID)
	}
	if versions.bumpUserID != 7 {
		t.Fatalf("bumped user id = %d, want 7", versions.bumpUserID)
	}
}

func TestDeleteAccountRejectsEmptyPassword(t *testing.T) {
	svc := NewUserService(
		&fakeDataFactory{users: &fakeUserData{}},
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		nil,
		nil,
		nil,
		&fakeTokenVersionStore{},
	)

	err := svc.DeleteAccount(context.Background(), 7, " ")
	if !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("DeleteAccount() error = %v, want ErrValidation", err)
	}
}

func TestDeleteAccountReturnsPasswordErrorBeforeDelete(t *testing.T) {
	users := &fakeUserData{
		user: data.User{
			ID:       7,
			NickName: "tester",
			PassWord: "hashed",
		},
		checkPasswordErr: errors.WithCode(code.ErrUserPasswordIncorrect, "bad password"),
	}
	svc := NewUserService(
		&fakeDataFactory{users: users},
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		nil,
		nil,
		nil,
		&fakeTokenVersionStore{},
	)

	err := svc.DeleteAccount(context.Background(), 7, "bad")
	if !errors.IsCode(err, code.ErrUserPasswordIncorrect) {
		t.Fatalf("DeleteAccount() error = %v, want ErrUserPasswordIncorrect", err)
	}
	if users.deleteCalled {
		t.Fatal("DeleteAccount() deleted user after password failure")
	}
}

type fakeTokenVersionStore struct {
	currentVersion uint64
	bumpUserID     uint64
	currentUserID  uint64
}

func (f *fakeTokenVersionStore) CurrentVersion(_ context.Context, userID uint64) (uint64, error) {
	f.currentUserID = userID
	return f.currentVersion, nil
}

func (f *fakeTokenVersionStore) Bump(_ context.Context, userID uint64) (uint64, error) {
	f.bumpUserID = userID
	f.currentVersion++
	return f.currentVersion, nil
}
