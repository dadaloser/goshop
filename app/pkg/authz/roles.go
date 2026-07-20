package authz

import "sort"

// LegacyUserRole is the historical numeric role stored on the user record.
// It remains for compatibility only and must not be used as the primary staff
// authorization model.
type LegacyUserRole int32

const (
	LegacyUserRoleCustomer LegacyUserRole = 1
	LegacyUserRoleAdmin    LegacyUserRole = 2
)

// IsValidLegacyUserRole reports whether the legacy numeric role is one of the
// compatibility values still recognized by the system.
func IsValidLegacyUserRole(role int32) bool {
	switch LegacyUserRole(role) {
	case LegacyUserRoleCustomer, LegacyUserRoleAdmin:
		return true
	default:
		return false
	}
}

type StaffRole string

const (
	StaffRoleSupport    StaffRole = "support"
	StaffRoleOps        StaffRole = "ops"
	StaffRoleFinance    StaffRole = "finance"
	StaffRoleCatalog    StaffRole = "catalog"
	StaffRoleAdmin      StaffRole = "admin"
	StaffRoleSuperAdmin StaffRole = "super_admin"
)

type RoleDefinition struct {
	Name        StaffRole
	Description string
	Permissions []Permission
}

var builtinRoleDefinitions = map[StaffRole]RoleDefinition{
	StaffRoleSupport: {
		Name:        StaffRoleSupport,
		Description: "customer support operations",
		Permissions: []Permission{
			PermissionRoleReadAny,
			PermissionUserListAny,
			PermissionUserReadAny,
			PermissionOrderReadAny,
			PermissionAuditReadAny,
		},
	},
	StaffRoleOps: {
		Name:        StaffRoleOps,
		Description: "operations management",
		Permissions: []Permission{
			PermissionGoodsReadAny,
			PermissionGoodsWriteAny,
			PermissionInventoryWriteAny,
			PermissionOrderReadAny,
			PermissionOrderCloseAny,
			PermissionAuditReadAny,
		},
	},
	StaffRoleFinance: {
		Name:        StaffRoleFinance,
		Description: "payment and refund operations",
		Permissions: []Permission{
			PermissionOrderReadAny,
			PermissionOrderRefundAny,
			PermissionAuditReadAny,
		},
	},
	StaffRoleCatalog: {
		Name:        StaffRoleCatalog,
		Description: "catalog and inventory maintenance",
		Permissions: []Permission{
			PermissionGoodsReadAny,
			PermissionGoodsWriteAny,
			PermissionInventoryWriteAny,
		},
	},
	StaffRoleAdmin: {
		Name:        StaffRoleAdmin,
		Description: "broad backoffice administration",
		Permissions: []Permission{
			PermissionRoleReadAny,
			PermissionUserListAny,
			PermissionUserReadAny,
			PermissionUserDisableAny,
			PermissionGoodsReadAny,
			PermissionGoodsWriteAny,
			PermissionInventoryWriteAny,
			PermissionOrderReadAny,
			PermissionOrderCloseAny,
			PermissionOrderRefundAny,
			PermissionAuditReadAny,
		},
	},
	StaffRoleSuperAdmin: {
		Name:        StaffRoleSuperAdmin,
		Description: "full backoffice administration",
		Permissions: []Permission{
			PermissionRoleReadAny,
			PermissionRoleAssignAny,
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
		},
	},
}

// BuiltinRoleDefinitions returns the built-in staff roles in stable order.
func BuiltinRoleDefinitions() []RoleDefinition {
	roles := make([]RoleDefinition, 0, len(builtinRoleDefinitions))
	for _, role := range []StaffRole{
		StaffRoleSupport,
		StaffRoleOps,
		StaffRoleFinance,
		StaffRoleCatalog,
		StaffRoleAdmin,
		StaffRoleSuperAdmin,
	} {
		definition := builtinRoleDefinitions[role]
		definition.Permissions = append([]Permission(nil), definition.Permissions...)
		roles = append(roles, definition)
	}
	return roles
}

func IsValidStaffRole(role string) bool {
	_, ok := builtinRoleDefinitions[StaffRole(role)]
	return ok
}

func PermissionsForRoles(roleNames []string) []Permission {
	permissions := make(map[Permission]struct{})
	for _, roleName := range roleNames {
		definition, ok := builtinRoleDefinitions[StaffRole(roleName)]
		if !ok {
			continue
		}
		for _, permission := range definition.Permissions {
			permissions[permission] = struct{}{}
		}
	}

	result := make([]Permission, 0, len(permissions))
	for permission := range permissions {
		result = append(result, permission)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}
