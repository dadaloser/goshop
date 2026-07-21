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

type BusinessDomain string

const (
	BusinessDomainSupport  BusinessDomain = "support"
	BusinessDomainOps      BusinessDomain = "operations"
	BusinessDomainFinance  BusinessDomain = "finance"
	BusinessDomainCatalog  BusinessDomain = "catalog"
	BusinessDomainPlatform BusinessDomain = "platform"
)

type RoleDefinition struct {
	Name        StaffRole
	Description string
	Permissions []Permission
	Domains     []BusinessDomain
}

var builtinRoleDefinitions = map[StaffRole]RoleDefinition{
	StaffRoleSupport: {
		Name:        StaffRoleSupport,
		Description: "customer support operations",
		Domains:     []BusinessDomain{BusinessDomainSupport},
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
		Domains:     []BusinessDomain{BusinessDomainOps},
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
		Domains:     []BusinessDomain{BusinessDomainFinance},
		Permissions: []Permission{
			PermissionOrderReadAny,
			PermissionOrderRefundAny,
			PermissionAuditReadAny,
		},
	},
	StaffRoleCatalog: {
		Name:        StaffRoleCatalog,
		Description: "catalog and inventory maintenance",
		Domains:     []BusinessDomain{BusinessDomainCatalog},
		Permissions: []Permission{
			PermissionGoodsReadAny,
			PermissionGoodsWriteAny,
			PermissionInventoryWriteAny,
		},
	},
	StaffRoleAdmin: {
		Name:        StaffRoleAdmin,
		Description: "broad backoffice administration",
		Domains:     []BusinessDomain{BusinessDomainPlatform},
		Permissions: []Permission{
			PermissionRoleReadAny,
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
			PermissionAuditReadAny,
		},
	},
	StaffRoleSuperAdmin: {
		Name:        StaffRoleSuperAdmin,
		Description: "full backoffice administration",
		Domains:     []BusinessDomain{BusinessDomainPlatform},
		Permissions: []Permission{
			PermissionRoleReadAny,
			PermissionRoleAssignAny,
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
		definition.Domains = append([]BusinessDomain(nil), definition.Domains...)
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

func DomainsForRoles(roleNames []string) []BusinessDomain {
	domains := make(map[BusinessDomain]struct{})
	for _, roleName := range roleNames {
		definition, ok := builtinRoleDefinitions[StaffRole(roleName)]
		if !ok {
			continue
		}
		for _, domain := range definition.Domains {
			domains[domain] = struct{}{}
		}
	}

	result := make([]BusinessDomain, 0, len(domains))
	for domain := range domains {
		result = append(result, domain)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func HasRole(roleNames []string, required StaffRole) bool {
	for _, roleName := range roleNames {
		if StaffRole(roleName) == required {
			return true
		}
	}
	return false
}

func CanManageRoleSet(actorRoles, targetRoles []string) bool {
	if HasRole(actorRoles, StaffRoleSuperAdmin) {
		return true
	}
	if HasRole(targetRoles, StaffRoleSuperAdmin) {
		return false
	}

	actorDomains := domainSet(DomainsForRoles(actorRoles))
	if len(actorDomains) == 0 {
		return false
	}
	if _, ok := actorDomains[BusinessDomainPlatform]; ok {
		return true
	}

	targetDomains := domainSet(DomainsForRoles(targetRoles))
	if _, ok := targetDomains[BusinessDomainPlatform]; ok {
		return false
	}
	for domain := range targetDomains {
		if _, ok := actorDomains[domain]; !ok {
			return false
		}
	}
	return true
}

func domainSet(domains []BusinessDomain) map[BusinessDomain]struct{} {
	set := make(map[BusinessDomain]struct{}, len(domains))
	for _, domain := range domains {
		set[domain] = struct{}{}
	}
	return set
}
