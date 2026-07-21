// Package authz defines the shared authorization vocabulary used by GoShop.
//
// Authentication mechanisms may differ between customer, staff, bootstrap, and
// service principals, but authorization checks should use these values instead
// of defining permission strings locally.
package authz

import "strings"

// PrincipalType identifies the kind of identity making a request.
type PrincipalType string

const (
	PrincipalAnonymous       PrincipalType = "anonymous"
	PrincipalCustomer        PrincipalType = "customer"
	PrincipalStaff           PrincipalType = "staff"
	PrincipalAdminBootstrap  PrincipalType = "admin_bootstrap"
	PrincipalInternalService PrincipalType = "internal_service"
)

// AccountStatus controls whether an authenticated account may be used.
type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusDisabled AccountStatus = "disabled"
	AccountStatusLocked   AccountStatus = "locked"
	AccountStatusDeleted  AccountStatus = "deleted"
)

// Permission is an authorization action with an explicit resource scope.
type Permission string

const (
	PermissionUserProfileReadSelf    Permission = "user:profile:read:self"
	PermissionUserProfileUpdateSelf  Permission = "user:profile:update:self"
	PermissionUserAccountDeleteSelf  Permission = "user:account:delete:self"
	PermissionCartReadSelf           Permission = "cart:read:self"
	PermissionCartWriteSelf          Permission = "cart:write:self"
	PermissionOrderCreateSelf        Permission = "order:create:self"
	PermissionOrderReadSelf          Permission = "order:read:self"
	PermissionOrderPaySelf           Permission = "order:pay:self"
	PermissionOrderStatusLogReadSelf Permission = "order:status_log:read:self"
	PermissionInventoryReadPublic    Permission = "inventory:read:public"

	PermissionUserCreateAny           Permission = "user:create:any"
	PermissionUserListAny             Permission = "user:list:any"
	PermissionUserReadAny             Permission = "user:read:any"
	PermissionUserDisableAny          Permission = "user:disable:any"
	PermissionGoodsReadAny            Permission = "goods:read:any"
	PermissionGoodsWriteAny           Permission = "goods:write:any"
	PermissionInventoryWriteAny       Permission = "inventory:write:any"
	PermissionOrderReadAny            Permission = "order:read:any"
	PermissionOrderCloseAny           Permission = "order:close:any"
	PermissionOrderRefundAny          Permission = "order:refund:any"
	PermissionPaymentCallbackSimulate Permission = "payment:callback:simulate"
	PermissionAuditReadAny            Permission = "audit:read:any"
	PermissionRoleReadAny             Permission = "role:read:any"
	PermissionRoleAssignAny           Permission = "role:assign:any"
	PermissionRoleWriteAny            Permission = "role:write:any"
)

var customerPermissions = []Permission{
	PermissionUserProfileReadSelf,
	PermissionUserProfileUpdateSelf,
	PermissionUserAccountDeleteSelf,
	PermissionCartReadSelf,
	PermissionCartWriteSelf,
	PermissionOrderCreateSelf,
	PermissionOrderReadSelf,
	PermissionOrderPaySelf,
	PermissionOrderStatusLogReadSelf,
}

var allPermissions = []Permission{
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

// CustomerPermissions returns a copy of the permissions granted to storefront
// customer sessions.
func CustomerPermissions() []Permission {
	permissions := make([]Permission, len(customerPermissions))
	copy(permissions, customerPermissions)
	return permissions
}

// CustomerScopes returns the customer permissions in their JWT string form.
func CustomerScopes() []string {
	permissions := CustomerPermissions()
	scopes := make([]string, len(permissions))
	for i, permission := range permissions {
		scopes[i] = string(permission)
	}
	return scopes
}

// AllPermissions returns the complete permission vocabulary recognized by the
// current authorization layer.
func AllPermissions() []Permission {
	permissions := make([]Permission, len(allPermissions))
	copy(permissions, allPermissions)
	return permissions
}

// IsValidPermission reports whether the string belongs to the shared
// authorization vocabulary.
func IsValidPermission(value string) bool {
	for _, permission := range allPermissions {
		if string(permission) == strings.TrimSpace(value) {
			return true
		}
	}
	return false
}

// NormalizeAccountStatus normalizes persisted and token account status. Empty
// values are treated as active only for compatibility during the column rollout.
func NormalizeAccountStatus(status string) AccountStatus {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return AccountStatusActive
	}
	return AccountStatus(status)
}
