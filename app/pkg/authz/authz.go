// Package authz defines the shared authorization vocabulary used by GoShop.
//
// Authentication mechanisms may differ between customer, staff, bootstrap, and
// service principals, but authorization checks should use these values instead
// of defining permission strings locally.
package authz

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
)
