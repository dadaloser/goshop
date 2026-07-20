package authz

import (
	"net/http"

	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
)

// RequirePermission rejects requests whose authenticated principal does not
// carry the required permission. It must run after authentication middleware.
func RequirePermission(permission Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := principalFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "authenticated principal is required",
			})
			return
		}

		if principal.status != AccountStatusActive || !principal.hasPermission(permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":       http.StatusForbidden,
				"msg":        "permission denied",
				"permission": permission,
			})
			return
		}
		c.Next()
	}
}

// RequirePrincipalTypes rejects authenticated requests whose principal type is
// not one of the allowed values.
func RequirePrincipalTypes(allowed ...PrincipalType) gin.HandlerFunc {
	allowedSet := make(map[PrincipalType]struct{}, len(allowed))
	for _, principalType := range allowed {
		allowedSet[principalType] = struct{}{}
	}

	return func(c *gin.Context) {
		principal, ok := principalFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "authenticated principal is required",
			})
			return
		}
		if _, ok := allowedSet[principal.typeName]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":           http.StatusForbidden,
				"msg":            "principal type denied",
				"principal_type": principal.typeName,
			})
			return
		}
		c.Next()
	}
}

type principal struct {
	typeName    PrincipalType
	status      AccountStatus
	permissions map[Permission]struct{}
}

func (p principal) hasPermission(permission Permission) bool {
	_, ok := p.permissions[permission]
	return ok
}

func principalFromContext(c *gin.Context) (principal, bool) {
	if c == nil {
		return principal{}, false
	}
	raw, ok := c.Get(middlewares.JWTPayloadKey)
	if !ok {
		return principal{}, false
	}
	payload, ok := raw.(map[string]any)
	if !ok {
		return principal{}, false
	}

	typeName := PrincipalType(stringClaim(payload, "principal_type"))
	legacyCustomer := typeName == ""
	if legacyCustomer {
		typeName = PrincipalCustomer
	}
	if !knownPrincipalType(typeName) {
		return principal{}, false
	}
	if typeName != PrincipalAdminBootstrap && !validUserID(payload["user_id"]) {
		return principal{}, false
	}
	status := AccountStatus(stringClaim(payload, "status"))
	if status == "" && legacyCustomer {
		status = AccountStatusActive
	}

	rawScope, hasScope := payload["scope"]
	permissions := permissionSet(rawScope)
	// Tokens issued before scoped authorization did not carry scope. Preserve
	// customer sessions during rollout while still denying non-customer tokens.
	if !hasScope && legacyCustomer {
		permissions = permissionSet(CustomerScopes())
	}

	return principal{
		typeName:    typeName,
		status:      status,
		permissions: permissions,
	}, true
}

func knownPrincipalType(principalType PrincipalType) bool {
	switch principalType {
	case PrincipalAnonymous, PrincipalCustomer, PrincipalStaff, PrincipalAdminBootstrap, PrincipalInternalService:
		return true
	default:
		return false
	}
}

func validUserID(raw any) bool {
	switch value := raw.(type) {
	case float64:
		return value > 0 && value == float64(uint64(value))
	case uint:
		return value > 0
	case uint64:
		return value > 0
	case int:
		return value > 0
	case int64:
		return value > 0
	default:
		return false
	}
}

func stringClaim(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func permissionSet(raw any) map[Permission]struct{} {
	permissions := make(map[Permission]struct{})
	switch values := raw.(type) {
	case []string:
		for _, value := range values {
			permissions[Permission(value)] = struct{}{}
		}
	case []any:
		for _, value := range values {
			if scope, ok := value.(string); ok {
				permissions[Permission(scope)] = struct{}{}
			}
		}
	}
	return permissions
}
