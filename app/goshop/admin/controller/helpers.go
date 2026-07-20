package controller

import (
	"net/http"
	"strings"

	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
)

func currentActor(ctx *gin.Context) (*upbv1.AuditActor, bool) {
	if ctx == nil {
		return nil, false
	}
	raw, ok := ctx.Get(middlewares.JWTPayloadKey)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "authenticated principal is required",
		})
		return nil, false
	}
	payload, ok := raw.(map[string]any)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "authenticated principal is required",
		})
		return nil, false
	}

	userID, ok := uint64Claim(payload["user_id"])
	if !ok || userID == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "token user id invalid",
		})
		return nil, false
	}
	principalType, _ := payload["principal_type"].(string)
	return &upbv1.AuditActor{
		ActorUserId:   int32(userID),
		PrincipalType: principalType,
	}, true
}

func currentRoles(ctx *gin.Context) []string {
	if ctx == nil {
		return nil
	}
	raw, ok := ctx.Get(middlewares.JWTPayloadKey)
	if !ok {
		return nil
	}
	payload, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	var roles []string
	switch values := payload["roles"].(type) {
	case []string:
		for _, value := range values {
			roles = append(roles, strings.ToLower(strings.TrimSpace(value)))
		}
	case []any:
		for _, value := range values {
			if role, ok := value.(string); ok {
				roles = append(roles, strings.ToLower(strings.TrimSpace(role)))
			}
		}
	}
	return roles
}

func hasCurrentRole(ctx *gin.Context, role authz.StaffRole) bool {
	required := string(role)
	for _, current := range currentRoles(ctx) {
		if current == required {
			return true
		}
	}
	return false
}

func uint64Claim(raw any) (uint64, bool) {
	switch value := raw.(type) {
	case float64:
		if value > 0 {
			return uint64(value), true
		}
	case uint64:
		if value > 0 {
			return value, true
		}
	case int64:
		if value > 0 {
			return uint64(value), true
		}
	case int:
		if value > 0 {
			return uint64(value), true
		}
	case uint:
		if value > 0 {
			return uint64(value), true
		}
	}
	return 0, false
}
