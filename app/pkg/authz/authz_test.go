package authz

import (
	"strings"
	"testing"
)

func TestPermissionsHaveExplicitScope(t *testing.T) {
	permissions := []Permission{
		PermissionUserProfileReadSelf,
		PermissionUserProfileUpdateSelf,
		PermissionUserAccountDeleteSelf,
		PermissionCartReadSelf,
		PermissionCartWriteSelf,
		PermissionOrderCreateSelf,
		PermissionOrderReadSelf,
		PermissionOrderPaySelf,
		PermissionOrderStatusLogReadSelf,
		PermissionInventoryReadPublic,
		PermissionUserListAny,
		PermissionUserReadAny,
		PermissionUserDisableAny,
		PermissionGoodsReadAny,
		PermissionGoodsWriteAny,
		PermissionInventoryWriteAny,
		PermissionOrderReadAny,
		PermissionOrderCloseAny,
		PermissionOrderRefundAny,
		PermissionPaymentCallbackSimulate,
		PermissionAuditReadAny,
		PermissionRoleReadAny,
		PermissionRoleAssignAny,
	}

	seen := make(map[Permission]struct{}, len(permissions))
	for _, permission := range permissions {
		permission := permission
		t.Run(string(permission), func(t *testing.T) {
			t.Parallel()
			parts := strings.Split(string(permission), ":")
			if len(parts) < 3 {
				t.Fatalf("permission %q has no explicit scope", permission)
			}
			scope := parts[len(parts)-1]
			if scope != "self" && scope != "team" && scope != "store" && scope != "any" && scope != "public" && scope != "simulate" {
				t.Fatalf("permission %q has unsupported scope %q", permission, scope)
			}
		})

		if _, ok := seen[permission]; ok {
			t.Fatalf("duplicate permission %q", permission)
		}
		seen[permission] = struct{}{}
	}
}

func TestLegacyUserRoleValidation(t *testing.T) {
	tests := []struct {
		role int32
		want bool
	}{
		{role: int32(LegacyUserRoleCustomer), want: true},
		{role: int32(LegacyUserRoleAdmin), want: true},
		{role: 0, want: false},
		{role: 3, want: false},
	}

	for _, tt := range tests {
		if got := IsValidLegacyUserRole(tt.role); got != tt.want {
			t.Fatalf("IsValidLegacyUserRole(%d) = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestPermissionsForRolesDeduplicatesAndSorts(t *testing.T) {
	got := PermissionsForRoles([]string{string(StaffRoleAdmin), string(StaffRoleSupport), "unknown"})
	if len(got) == 0 {
		t.Fatal("PermissionsForRoles() returned no permissions")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Fatalf("permissions are not sorted: %v", got)
		}
	}
	seen := make(map[Permission]struct{}, len(got))
	for _, permission := range got {
		if _, ok := seen[permission]; ok {
			t.Fatalf("duplicate permission %q", permission)
		}
		seen[permission] = struct{}{}
	}
	if _, ok := seen[PermissionUserListAny]; !ok {
		t.Fatalf("PermissionsForRoles() missing %q", PermissionUserListAny)
	}
	if _, ok := seen[PermissionOrderReadAny]; !ok {
		t.Fatalf("PermissionsForRoles() missing %q", PermissionOrderReadAny)
	}
}
