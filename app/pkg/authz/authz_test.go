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
		PermissionUserCreateAny,
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
		PermissionRoleWriteAny,
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

func TestDomainsForRolesDeduplicatesAndSorts(t *testing.T) {
	got := DomainsForRoles([]string{string(StaffRoleOps), string(StaffRoleCatalog), string(StaffRoleOps), "unknown"})
	want := []BusinessDomain{BusinessDomainCatalog, BusinessDomainOps}
	if len(got) != len(want) {
		t.Fatalf("len(DomainsForRoles()) = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DomainsForRoles()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCanManageRoleSet(t *testing.T) {
	tests := []struct {
		name        string
		actorRoles  []string
		targetRoles []string
		want        bool
	}{
		{
			name:        "super admin can manage any target",
			actorRoles:  []string{string(StaffRoleSuperAdmin)},
			targetRoles: []string{string(StaffRoleFinance)},
			want:        true,
		},
		{
			name:        "admin platform role can manage non super admin targets",
			actorRoles:  []string{string(StaffRoleAdmin)},
			targetRoles: []string{string(StaffRoleFinance)},
			want:        true,
		},
		{
			name:        "ops cannot manage finance target",
			actorRoles:  []string{string(StaffRoleOps)},
			targetRoles: []string{string(StaffRoleFinance)},
			want:        false,
		},
		{
			name:        "ops can manage ops target",
			actorRoles:  []string{string(StaffRoleOps)},
			targetRoles: []string{string(StaffRoleOps)},
			want:        true,
		},
		{
			name:        "non super admin cannot manage super admin target",
			actorRoles:  []string{string(StaffRoleAdmin)},
			targetRoles: []string{string(StaffRoleSuperAdmin)},
			want:        false,
		},
	}

	for _, tt := range tests {
		if got := CanManageRoleSet(tt.actorRoles, tt.targetRoles); got != tt.want {
			t.Fatalf("%s: CanManageRoleSet(%v, %v) = %v, want %v", tt.name, tt.actorRoles, tt.targetRoles, got, tt.want)
		}
	}
}

func TestReservedNonStaffRoleNamesDoNotOverlapBuiltinStaffRoles(t *testing.T) {
	for _, roleName := range ReservedNonStaffRoleNames() {
		if IsValidStaffRole(roleName) {
			t.Fatalf("reserved non-staff role %q overlaps builtin staff role", roleName)
		}
	}
}

func TestResourceScopeMatchesDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain BusinessDomain
		store  string
		team   string
		want   bool
	}{
		{name: "platform without identifiers", domain: BusinessDomainPlatform, want: true},
		{name: "platform rejects store", domain: BusinessDomainPlatform, store: "store-a", want: false},
		{name: "catalog requires store", domain: BusinessDomainCatalog, store: "store-a", want: true},
		{name: "catalog rejects missing store", domain: BusinessDomainCatalog, want: false},
		{name: "catalog rejects team", domain: BusinessDomainCatalog, store: "store-a", team: "team-a", want: false},
		{name: "ops requires team", domain: BusinessDomainOps, team: "warehouse-a", want: true},
		{name: "ops rejects store", domain: BusinessDomainOps, store: "store-a", team: "warehouse-a", want: false},
	}

	for _, tt := range tests {
		if got := ResourceScopeMatchesDomain(tt.domain, tt.store, tt.team); got != tt.want {
			t.Fatalf("%s: ResourceScopeMatchesDomain(%q, %q, %q) = %v, want %v", tt.name, tt.domain, tt.store, tt.team, got, tt.want)
		}
	}
}
