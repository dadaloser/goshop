package v1

import (
	"context"
	"strings"
	"testing"
	"time"

	"goshop/app/pkg/authz"
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
	if store.created.Status != string(authz.AccountStatusActive) {
		t.Fatalf("created status = %q, want %q", store.created.Status, authz.AccountStatusActive)
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
			"user_001": {Username: stringPtr("user_001"), Role: int(authz.LegacyUserRoleCustomer)},
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

func TestUserService_CreateRejectsUnknownLegacyRole(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{},
	}
	svc := NewUserService(store)

	err := svc.Create(context.Background(), &UserDTO{
		UserDO: dv1.UserDO{
			Mobile:   "13800138000",
			Password: "Secret123!",
			Role:     99,
		},
	})
	if !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
	if store.created != nil {
		t.Fatal("Create() persisted user with invalid legacy role")
	}
}

func TestUserService_GetByUsernameNormalizesIdentifier(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{
			"user_001": {
				Username: stringPtr("user_001"),
				Role:     int(authz.LegacyUserRoleCustomer),
			},
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

func TestUserService_ListStaffRolesReturnsDescriptionsAndPermissions(t *testing.T) {
	store := &fakeUserStore{
		roles: []dv1.RoleDO{
			{Name: string(authz.StaffRoleAdmin), Description: "broad backoffice administration", Permissions: []string{string(authz.PermissionUserListAny)}},
			{Name: string(authz.StaffRoleSuperAdmin), Description: "full backoffice administration", Permissions: []string{string(authz.PermissionRoleWriteAny)}},
		},
	}
	svc := NewUserService(store)

	roles, err := svc.ListStaffRoles(context.Background())
	if err != nil {
		t.Fatalf("ListStaffRoles() error = %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("len(ListStaffRoles()) = %d, want 2", len(roles))
	}
	if roles[0].Name != string(authz.StaffRoleAdmin) {
		t.Fatalf("roles[0].Name = %q, want %q", roles[0].Name, authz.StaffRoleAdmin)
	}
	if !roles[0].Builtin {
		t.Fatal("roles[0].Builtin = false, want true")
	}
	if len(roles[0].Permissions) == 0 {
		t.Fatal("roles[0].Permissions is empty")
	}
	if len(roles[0].Domains) == 0 {
		t.Fatal("roles[0].Domains is empty")
	}
}

func TestUserService_UpdateStaffRoleValidatesAndNormalizesPermissions(t *testing.T) {
	store := &fakeUserStore{}
	svc := NewUserService(store)

	role, err := svc.UpdateStaffRole(context.Background(), StaffRoleDTO{
		Name:        string(authz.StaffRoleOps),
		Description: "updated ops role",
		Permissions: []string{" order:read:any ", string(authz.PermissionOrderCloseAny), string(authz.PermissionOrderReadAny)},
	})
	if err != nil {
		t.Fatalf("UpdateStaffRole() error = %v", err)
	}
	if store.updatedRole == nil {
		t.Fatal("UpdateStaffRole() did not reach store")
	}
	if len(store.updatedRole.Permissions) != 2 {
		t.Fatalf("updated permissions = %#v, want deduplicated permissions", store.updatedRole.Permissions)
	}
	if !role.Builtin {
		t.Fatal("updated role should remain builtin")
	}
	if len(role.Domains) == 0 {
		t.Fatal("updated role domains are empty")
	}

	if _, err = svc.UpdateStaffRole(context.Background(), StaffRoleDTO{
		Name:        "custom_role",
		Description: "custom",
		Permissions: []string{string(authz.PermissionOrderReadAny)},
	}); !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("UpdateStaffRole() custom role error = %v, want ErrValidation", err)
	}

	if _, err = svc.UpdateStaffRole(context.Background(), StaffRoleDTO{
		Name:        string(authz.StaffRoleOps),
		Description: "updated ops role",
		Permissions: []string{"role:own:any"},
		Domains:     []string{string(authz.BusinessDomainOps)},
	}); !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("UpdateStaffRole() unknown permission error = %v, want ErrValidation", err)
	}
}

func TestUserService_CreateStaffRoleValidatesAndCreatesCustomRole(t *testing.T) {
	store := &fakeUserStore{}
	svc := NewUserService(store)

	role, err := svc.CreateStaffRole(context.Background(), StaffRoleDTO{
		Name:        "ops_delegate",
		Description: "operations delegate",
		Permissions: []string{string(authz.PermissionOrderReadAny), string(authz.PermissionOrderCloseAny)},
		Domains:     []string{string(authz.BusinessDomainOps)},
	})
	if err != nil {
		t.Fatalf("CreateStaffRole() error = %v", err)
	}
	if store.createdRole == nil {
		t.Fatal("CreateStaffRole() did not reach store")
	}
	if role.Builtin {
		t.Fatal("custom role unexpectedly marked builtin")
	}
	if len(role.Domains) != 1 || role.Domains[0] != string(authz.BusinessDomainOps) {
		t.Fatalf("created role domains = %#v, want operations", role.Domains)
	}

	if _, err = svc.CreateStaffRole(context.Background(), StaffRoleDTO{
		Name:        string(authz.StaffRoleOps),
		Description: "builtin duplicate",
		Permissions: []string{string(authz.PermissionOrderReadAny)},
		Domains:     []string{string(authz.BusinessDomainOps)},
	}); !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("CreateStaffRole() builtin duplicate error = %v, want ErrValidation", err)
	}
}

func TestUserService_DeleteStaffRoleValidatesAndDeletesCustomRole(t *testing.T) {
	store := &fakeUserStore{}
	svc := NewUserService(store)

	if err := svc.DeleteStaffRole(context.Background(), " ops_delegate "); err != nil {
		t.Fatalf("DeleteStaffRole() error = %v", err)
	}
	if store.deletedRoleName != "ops_delegate" {
		t.Fatalf("deleted role name = %q, want ops_delegate", store.deletedRoleName)
	}

	if err := svc.DeleteStaffRole(context.Background(), string(authz.StaffRoleOps)); !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("DeleteStaffRole() builtin delete error = %v, want ErrValidation", err)
	}

	if err := svc.DeleteStaffRole(context.Background(), "!!!"); !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("DeleteStaffRole() invalid name error = %v, want ErrValidation", err)
	}
}

func TestUserService_ReplaceUserRoleBindingValidatesAndReturnsBinding(t *testing.T) {
	store := &fakeUserStore{
		authByID: map[uint64]*dv1.UserAuthDO{
			7: {
				UserDO: dv1.UserDO{
					BaseModel: dv1.BaseModel{ID: 7},
					Role:      int(authz.LegacyUserRoleAdmin),
				},
			},
		},
	}
	svc := NewUserService(store)

	binding, err := svc.ReplaceUserRoleBinding(context.Background(), 7, []string{" SUPER_ADMIN ", "super_admin"}, AuditActorDTO{UserID: 1, PrincipalType: string(authz.PrincipalStaff)})
	if err != nil {
		t.Fatalf("ReplaceUserRoleBinding() error = %v", err)
	}
	if store.replacedUserID != 7 {
		t.Fatalf("replaced user id = %d, want 7", store.replacedUserID)
	}
	if len(store.replacedRoles) != 1 {
		t.Fatalf("len(replaced roles) = %d, want 1 normalized role", len(store.replacedRoles))
	}
	if got := store.replacedRoles[0]; got != string(authz.StaffRoleSuperAdmin) {
		t.Fatalf("replaced role = %q, want %q", got, authz.StaffRoleSuperAdmin)
	}
	if len(binding.StaffRoles) != 1 {
		t.Fatalf("binding roles = %#v, want one normalized role", binding.StaffRoles)
	}
	if got := binding.StaffRoles[0]; got != string(authz.StaffRoleSuperAdmin) {
		t.Fatalf("binding role = %q, want %q", got, authz.StaffRoleSuperAdmin)
	}
	if len(binding.Permissions) == 0 {
		t.Fatal("binding permissions is empty")
	}

	if _, err = svc.ReplaceUserRoleBinding(context.Background(), 7, []string{"owner"}, AuditActorDTO{}); !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("ReplaceUserRoleBinding() invalid role error = %v, want ErrValidation", err)
	}
}

func TestUserService_CreateStaff(t *testing.T) {
	store := &fakeUserStore{
		usersByIdentifier: map[string]*dv1.UserDO{},
	}
	svc := NewUserService(store)

	created, err := svc.CreateStaff(context.Background(), &UserDTO{
		UserDO: dv1.UserDO{
			Username: stringPtr(" Ops_001 "),
			Mobile:   "13800138000",
			Email:    stringPtr(" OPS@example.com "),
			NickName: "ops",
			Password: "Secret123!",
		},
	}, []string{string(authz.StaffRoleOps)}, "disabled", AuditActorDTO{UserID: 9, PrincipalType: string(authz.PrincipalStaff)})
	if err != nil {
		t.Fatalf("CreateStaff() error = %v", err)
	}
	if store.createdStaff == nil {
		t.Fatal("CreateStaff() did not persist staff user")
	}
	if store.createdStaff.Role != int(authz.LegacyUserRoleAdmin) {
		t.Fatalf("created legacy role = %d, want %d", store.createdStaff.Role, authz.LegacyUserRoleAdmin)
	}
	if store.createdStaff.Status != string(authz.AccountStatusDisabled) {
		t.Fatalf("created status = %q, want %q", store.createdStaff.Status, authz.AccountStatusDisabled)
	}
	if store.createdStaffActor == nil || store.createdStaffActor.UserID != 9 {
		t.Fatalf("created staff actor = %#v, want user id 9", store.createdStaffActor)
	}
	if len(created.StaffRoles) != 1 || created.StaffRoles[0] != string(authz.StaffRoleOps) {
		t.Fatalf("created roles = %#v, want ops", created.StaffRoles)
	}
}

func TestUserService_UpdateStatus(t *testing.T) {
	store := &fakeUserStore{
		userByID: map[uint64]*dv1.UserDO{
			7: {
				BaseModel: dv1.BaseModel{ID: 7},
				Mobile:    "13800138000",
				NickName:  "tester",
				Role:      int(authz.LegacyUserRoleCustomer),
				Status:    string(authz.AccountStatusActive),
			},
		},
	}
	svc := NewUserService(store)

	user, err := svc.UpdateStatus(context.Background(), 7, "disabled", AuditActorDTO{UserID: 5, PrincipalType: string(authz.PrincipalStaff)})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if user.Status != string(authz.AccountStatusDisabled) {
		t.Fatalf("updated status = %q, want %q", user.Status, authz.AccountStatusDisabled)
	}
	if store.updatedStatusID != 7 || store.updatedStatus != string(authz.AccountStatusDisabled) {
		t.Fatalf("store updated = (%d, %q), want (7, %q)", store.updatedStatusID, store.updatedStatus, authz.AccountStatusDisabled)
	}
	if store.updatedStatusActor == nil || store.updatedStatusActor.UserID != 5 {
		t.Fatalf("updated status actor = %#v, want user id 5", store.updatedStatusActor)
	}

	if _, err = svc.UpdateStatus(context.Background(), 7, "deleted", AuditActorDTO{}); !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("UpdateStatus() invalid status error = %v, want ErrValidation", err)
	}
}

func TestUserService_ListUserAuditLogsPassesFilters(t *testing.T) {
	after := time.Unix(1700000000, 0)
	before := time.Unix(1700003600, 0)
	store := &fakeUserStore{}
	svc := NewUserService(store)

	if _, err := svc.ListUserAuditLogs(context.Background(), 7, UserAuditLogFilterDTO{
		Action:             dv1.UserAuditActionStatusUpdated,
		ActorUserID:        9,
		ActorPrincipalType: string(authz.PrincipalStaff),
		CreatedAfter:       &after,
		CreatedBefore:      &before,
	}, metav1.ListMeta{Page: 2, PageSize: 20}); err != nil {
		t.Fatalf("ListUserAuditLogs() error = %v", err)
	}

	if store.auditFilters.Action != dv1.UserAuditActionStatusUpdated {
		t.Fatalf("audit action = %q, want %q", store.auditFilters.Action, dv1.UserAuditActionStatusUpdated)
	}
	if store.auditFilters.ActorUserID != 9 {
		t.Fatalf("audit actor user id = %d, want 9", store.auditFilters.ActorUserID)
	}
	if store.auditFilters.ActorPrincipalType != string(authz.PrincipalStaff) {
		t.Fatalf("audit actor principal type = %q, want %q", store.auditFilters.ActorPrincipalType, authz.PrincipalStaff)
	}
	if store.auditFilters.CreatedAfter == nil || !store.auditFilters.CreatedAfter.Equal(after) {
		t.Fatalf("audit created after = %#v, want %v", store.auditFilters.CreatedAfter, after)
	}
	if store.auditFilters.CreatedBefore == nil || !store.auditFilters.CreatedBefore.Equal(before) {
		t.Fatalf("audit created before = %#v, want %v", store.auditFilters.CreatedBefore, before)
	}
}

type fakeUserStore struct {
	usersByIdentifier  map[string]*dv1.UserDO
	userByID           map[uint64]*dv1.UserDO
	authByID           map[uint64]*dv1.UserAuthDO
	roles              []dv1.RoleDO
	createdRole        *dv1.RoleDO
	updatedRole        *dv1.RoleDO
	deletedRoleName    string
	replacedUserID     uint64
	replacedRoles      []string
	replacedActor      *dv1.AuditActor
	created            *dv1.UserDO
	createdStaff       *dv1.UserDO
	createdStaffRoles  []string
	createdStaffActor  *dv1.AuditActor
	updatedStatusID    uint64
	updatedStatus      string
	updatedStatusActor *dv1.AuditActor
	auditFilters       dv1.UserAuditLogFilters
	deletedID          uint64
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

func (f *fakeUserStore) GetByID(_ context.Context, id uint64) (*dv1.UserDO, error) {
	if f.userByID != nil {
		if user, ok := f.userByID[id]; ok {
			return user, nil
		}
	}
	return nil, errors.WithCode(code.ErrUserNotFound, "not found")
}

func (f *fakeUserStore) GetAuthByUsername(_ context.Context, username string) (*dv1.UserAuthDO, error) {
	user, err := f.GetByUsername(context.Background(), username)
	if err != nil {
		return nil, err
	}
	return &dv1.UserAuthDO{UserDO: *user}, nil
}

func (f *fakeUserStore) GetAuthByID(_ context.Context, id uint64) (*dv1.UserAuthDO, error) {
	if f.authByID != nil {
		if user, ok := f.authByID[id]; ok {
			return user, nil
		}
	}
	user, err := f.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return &dv1.UserAuthDO{UserDO: *user}, nil
}

func (f *fakeUserStore) ListRoles(context.Context) ([]dv1.RoleDO, error) {
	if len(f.roles) == 0 {
		roles := make([]dv1.RoleDO, 0, len(authz.BuiltinRoleDefinitions()))
		for _, definition := range authz.BuiltinRoleDefinitions() {
			domains := make([]string, 0, len(definition.Domains))
			for _, domain := range definition.Domains {
				domains = append(domains, string(domain))
			}
			roles = append(roles, dv1.RoleDO{
				Name:        string(definition.Name),
				Description: definition.Description,
				Builtin:     true,
				Domains:     domains,
			})
		}
		return roles, nil
	}
	return append([]dv1.RoleDO(nil), f.roles...), nil
}

func (f *fakeUserStore) CreateRole(_ context.Context, roleName, description string, permissions, domains []string) (*dv1.RoleDO, error) {
	f.createdRole = &dv1.RoleDO{
		Name:        roleName,
		Description: description,
		Permissions: append([]string(nil), permissions...),
		Domains:     append([]string(nil), domains...),
	}
	return f.createdRole, nil
}

func (f *fakeUserStore) UpdateRole(_ context.Context, roleName, description string, permissions, domains []string) (*dv1.RoleDO, error) {
	f.updatedRole = &dv1.RoleDO{
		Name:        roleName,
		Description: description,
		Permissions: append([]string(nil), permissions...),
		Domains:     append([]string(nil), domains...),
		Builtin:     authz.IsValidStaffRole(roleName),
	}
	return f.updatedRole, nil
}

func (f *fakeUserStore) DeleteRole(_ context.Context, roleName string) error {
	f.deletedRoleName = roleName
	return nil
}

func (f *fakeUserStore) ReplaceUserRoles(_ context.Context, userID uint64, roleNames []string, actor *dv1.AuditActor) (*dv1.UserAuthDO, error) {
	f.replacedUserID = userID
	f.replacedRoles = append([]string(nil), roleNames...)
	f.replacedActor = actor
	if f.authByID != nil {
		if user, ok := f.authByID[userID]; ok {
			user.StaffRoles = append([]string(nil), roleNames...)
			user.Permissions = []string{string(authz.PermissionRoleReadAny)}
			return user, nil
		}
	}
	return nil, errors.WithCode(code.ErrUserNotFound, "not found")
}

func (f *fakeUserStore) ListAuditLogs(_ context.Context, _ uint64, filters dv1.UserAuditLogFilters, _ metav1.ListMeta) (*dv1.UserAuditLogDOList, error) {
	f.auditFilters = filters
	return &dv1.UserAuditLogDOList{}, nil
}

func (f *fakeUserStore) Create(_ context.Context, user *dv1.UserDO) error {
	f.created = user
	return nil
}

func (f *fakeUserStore) CreateStaff(_ context.Context, user *dv1.UserDO, roleNames []string, actor *dv1.AuditActor) (*dv1.UserAuthDO, error) {
	f.createdStaff = user
	f.createdStaffRoles = append([]string(nil), roleNames...)
	f.createdStaffActor = actor
	if user.ID == 0 {
		user.ID = 10
	}
	return &dv1.UserAuthDO{
		UserDO:      *user,
		StaffRoles:  append([]string(nil), roleNames...),
		Permissions: []string{string(authz.PermissionUserCreateAny)},
	}, nil
}

func (f *fakeUserStore) Update(context.Context, *dv1.UserDO) error {
	return nil
}

func (f *fakeUserStore) UpdateStatus(_ context.Context, id uint64, status string, actor *dv1.AuditActor) error {
	f.updatedStatusID = id
	f.updatedStatus = status
	f.updatedStatusActor = actor
	if f.userByID != nil {
		if user, ok := f.userByID[id]; ok {
			user.Status = status
			return nil
		}
	}
	return errors.WithCode(code.ErrUserNotFound, "not found")
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
