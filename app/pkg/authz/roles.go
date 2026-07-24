package authz

import (
	"sort"
	"strings"
)

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
	StaffRoleReview     StaffRole = "review"
	StaffRoleAdmin      StaffRole = "admin"
	StaffRoleSuperAdmin StaffRole = "super_admin"
)

type BusinessDomain string

const (
	BusinessDomainSupport  BusinessDomain = "support"
	BusinessDomainOps      BusinessDomain = "operations"
	BusinessDomainFinance  BusinessDomain = "finance"
	BusinessDomainCatalog  BusinessDomain = "catalog"
	BusinessDomainReview   BusinessDomain = "review"
	BusinessDomainPlatform BusinessDomain = "platform"
)

func AllBusinessDomains() []BusinessDomain {
	return []BusinessDomain{
		BusinessDomainSupport,
		BusinessDomainOps,
		BusinessDomainFinance,
		BusinessDomainCatalog,
		BusinessDomainReview,
		BusinessDomainPlatform,
	}
}

func IsValidBusinessDomain(value string) bool {
	for _, domain := range AllBusinessDomains() {
		if string(domain) == value {
			return true
		}
	}
	return false
}

type ResourceScopeDimension string

const (
	ResourceScopeDimensionNone  ResourceScopeDimension = "none"
	ResourceScopeDimensionStore ResourceScopeDimension = "store"
	ResourceScopeDimensionTeam  ResourceScopeDimension = "team"
)

func ResourceScopeDimensionForDomain(domain BusinessDomain) ResourceScopeDimension {
	switch domain {
	case BusinessDomainPlatform:
		return ResourceScopeDimensionNone
	case BusinessDomainOps:
		return ResourceScopeDimensionTeam
	default:
		return ResourceScopeDimensionStore
	}
}

func ResourceScopeMatchesDomain(
	domain BusinessDomain,
	storeID string,
	teamID string,
) bool {
	storeID = strings.TrimSpace(storeID)
	teamID = strings.TrimSpace(teamID)

	switch ResourceScopeDimensionForDomain(domain) {
	case ResourceScopeDimensionNone:
		return storeID == "" && teamID == ""
	case ResourceScopeDimensionStore:
		return storeID != "" && teamID == ""
	case ResourceScopeDimensionTeam:
		return storeID == "" && teamID != ""
	default:
		return false
	}
}

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
			PermissionInventoryReadAny,
			PermissionInventoryWriteAny,
			PermissionInventoryAuditReadAny,
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
		},
	},
	StaffRoleReview: {
		Name: StaffRoleReview, Description: "review moderation and merchant replies",
		Domains:     []BusinessDomain{BusinessDomainReview},
		Permissions: []Permission{PermissionReviewModerateAny, PermissionReviewReplyAny},
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
			PermissionInventoryReadAny,
			PermissionInventoryWriteAny,
			PermissionInventoryAuditReadAny,
			PermissionOrderReadAny,
			PermissionOrderCloseAny,
			PermissionOrderRefundAny,
			PermissionAuditReadAny,
			PermissionReviewModerateAny,
			PermissionReviewReplyAny,
			PermissionReviewAggregateRebuild,
		},
	},
	StaffRoleSuperAdmin: {
		Name:        StaffRoleSuperAdmin,
		Description: "full backoffice administration",
		Domains:     []BusinessDomain{BusinessDomainPlatform},
		Permissions: []Permission{
			PermissionRoleReadAny,
			PermissionRoleAssignAny,
			PermissionRoleWriteAny,
			PermissionUserCreateAny,
			PermissionUserListAny,
			PermissionUserReadAny,
			PermissionUserDisableAny,
			PermissionGoodsReadAny,
			PermissionGoodsWriteAny,
			PermissionInventoryReadAny,
			PermissionInventoryWriteAny,
			PermissionInventoryAuditReadAny,
			PermissionOrderReadAny,
			PermissionOrderCloseAny,
			PermissionOrderRefundAny,
			PermissionPaymentCallbackSimulate,
			PermissionAuditReadAny,
			PermissionReviewModerateAny,
			PermissionReviewReplyAny,
			PermissionReviewAggregateRebuild,
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
		StaffRoleReview,
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

var reservedNonStaffRoleNames = []string{
	"admin_bootstrap",
	"basic",
	"business_admin",
	"normal_permission",
	"primary_admin",
}

func ReservedNonStaffRoleNames() []string {
	result := make([]string, len(reservedNonStaffRoleNames))
	copy(result, reservedNonStaffRoleNames)
	sort.Strings(result)
	return result
}

func IsReservedNonStaffRoleName(roleName string) bool {
	for _, reserved := range reservedNonStaffRoleNames {
		if reserved == roleName {
			return true
		}
	}
	return false
}
