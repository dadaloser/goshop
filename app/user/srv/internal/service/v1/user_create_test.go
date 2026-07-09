package v1

import (
	"context"
	"strings"
	"testing"

	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"
)

func TestUserService_CreateNormalizesOptionalIdentifiers(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{},
	}
	svc := NewUserService(store)

	user := &UserDTO{
		UserDO: dv1.UserDO{
			Username: stringPtr(" user_001 "),
			Mobile:   "13800138000",
			Email:    stringPtr(" USER@example.COM "),
			Password: "Secret123!",
		},
	}

	if err := svc.Create(context.Background(), user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store.created == nil {
		t.Fatal("Create() did not persist user")
	}
	if got, want := valueOf(store.created.Username), "user_001"; got != want {
		t.Fatalf("created username = %q, want %q", got, want)
	}
	if got, want := valueOf(store.created.Email), "user@example.com"; got != want {
		t.Fatalf("created email = %q, want %q", got, want)
	}
	if store.created.Password == "Secret123!" {
		t.Fatal("created password was not encrypted")
	}
}

func TestUserService_CreateRejectsDuplicateEmailAfterNormalization(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{
			"user@example.com": {Email: stringPtr("user@example.com")},
		},
	}
	svc := NewUserService(store)

	err := svc.Create(context.Background(), &UserDTO{
		UserDO: dv1.UserDO{
			Mobile:   "13800138000",
			Email:    stringPtr(" USER@example.COM "),
			Password: "Secret123!",
		},
	})
	if !errors.IsCode(err, code.ErrUserAlreadyExists) {
		t.Fatalf("Create() error = %v, want ErrUserAlreadyExists", err)
	}
}

func TestUserService_CreateRejectsDuplicateUsername(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{
			"user_001": {Username: stringPtr("user_001")},
		},
	}
	svc := NewUserService(store)

	err := svc.Create(context.Background(), &UserDTO{
		UserDO: dv1.UserDO{
			Username: stringPtr("user_001"),
			Mobile:   "13800138000",
			Password: "Secret123!",
		},
	})
	if !errors.IsCode(err, code.ErrUserAlreadyExists) {
		t.Fatalf("Create() error = %v, want ErrUserAlreadyExists", err)
	}
}

func TestUserService_CreateRejectsInvalidUsername(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{},
	}
	svc := NewUserService(store)

	err := svc.Create(context.Background(), &UserDTO{
		UserDO: dv1.UserDO{
			Username: stringPtr("user-name"),
			Mobile:   "13800138000",
			Password: "Secret123!",
		},
	})
	if !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	if store.created != nil {
		t.Fatal("Create() persisted user with invalid username")
	}
}

func TestUserService_CreateRejectsInvalidEmail(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{},
	}
	svc := NewUserService(store)

	err := svc.Create(context.Background(), &UserDTO{
		UserDO: dv1.UserDO{
			Mobile:   "13800138000",
			Email:    stringPtr("User <user@example.com>"),
			Password: "Secret123!",
		},
	})
	if !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	if store.created != nil {
		t.Fatal("Create() persisted user with invalid email")
	}
}

func TestUserService_CreateRejectsWeakPasswords(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "too short", password: "Aa1!"},
		{name: "missing upper", password: "secret123!"},
		{name: "missing lower", password: "SECRET123!"},
		{name: "missing digit", password: "Secret!!!"},
		{name: "missing special", password: "Secret123"},
		{name: "contains space", password: "Secret 123!"},
		{name: "too long for bcrypt", password: strings.Repeat("Secret123!", 8)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{
				usersByIdentifier: map[string]*dv1.UserDO{},
			}
			svc := NewUserService(store)

			err := svc.Create(context.Background(), &UserDTO{
				UserDO: dv1.UserDO{
					Mobile:   "13800138000",
					Password: tt.password,
				},
			})
			if !errors.IsCode(err, code2.ErrValidation) {
				t.Fatalf("Create() error = %v, want ErrValidation", err)
			}
			if store.created != nil {
				t.Fatal("Create() persisted user with weak password")
			}
		})
	}
}

func TestUserService_GetByUsernameNormalizesIdentifier(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{
			"user_001": {Username: stringPtr("user_001")},
		},
	}
	svc := NewUserService(store)

	got, err := svc.GetByUsername(context.Background(), " USER_001 ")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if valueOf(got.Username) != "user_001" {
		t.Fatalf("GetByUsername() username = %q, want user_001", valueOf(got.Username))
	}
}

type fakeUserStore struct {
	usersByIdentifier map[string]*dv1.UserDO
	created           *dv1.UserDO
	deletedID         uint64
}

func (f *fakeUserStore) List(context.Context, []string, metav1.ListMeta) (*dv1.UserDOList, error) {
	return &dv1.UserDOList{}, nil
}

func (f *fakeUserStore) GetByMobile(_ context.Context, mobile string) (*dv1.UserDO, error) {
	if user, ok := f.usersByIdentifier[mobile]; ok {
		return user, nil
	}
	return nil, errors.WithCode(code.ErrUserNotFound, "not found")
}

func (f *fakeUserStore) GetByUsername(_ context.Context, username string) (*dv1.UserDO, error) {
	if user, ok := f.usersByIdentifier[username]; ok {
		return user, nil
	}
	return nil, errors.WithCode(code.ErrUserNotFound, "not found")
}

func (f *fakeUserStore) GetByID(context.Context, uint64) (*dv1.UserDO, error) {
	return nil, errors.WithCode(code.ErrUserNotFound, "not found")
}

func (f *fakeUserStore) Create(_ context.Context, user *dv1.UserDO) error {
	f.created = user
	return nil
}

func (f *fakeUserStore) Update(context.Context, *dv1.UserDO) error {
	return nil
}

func (f *fakeUserStore) Delete(_ context.Context, id uint64) error {
	f.deletedID = id
	return nil
}

func stringPtr(value string) *string {
	return &value
}

func valueOf(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ dv1.UserStore = &fakeUserStore{}
