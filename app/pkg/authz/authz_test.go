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
