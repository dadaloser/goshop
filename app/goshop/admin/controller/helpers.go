package controller

import (
	"net/http"
	"sort"
	"strings"

	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func currentPermissions(ctx *gin.Context) []string {
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

	var permissions []string
	switch values := payload["scope"].(type) {
	case []string:
		for _, value := range values {
			permissions = append(permissions, strings.ToLower(strings.TrimSpace(value)))
		}
	case []any:
		for _, value := range values {
			if permission, ok := value.(string); ok {
				permissions = append(permissions, strings.ToLower(strings.TrimSpace(permission)))
			}
		}
	}
	return permissions
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

type roleTemplateView struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Domains     []string `json:"domains"`
	Builtin     bool     `json:"builtin"`
	Manageable  bool     `json:"manageable"`
}

func roleTemplateViews(actorRoles []string) []roleTemplateView {
	definitions := authz.BuiltinRoleDefinitions()
	result := make([]roleTemplateView, 0, len(definitions))
	for _, definition := range definitions {
		permissions := make([]string, 0, len(definition.Permissions))
		for _, permission := range definition.Permissions {
			permissions = append(permissions, string(permission))
		}
		domains := make([]string, 0, len(definition.Domains))
		for _, domain := range definition.Domains {
			domains = append(domains, string(domain))
		}
		sort.Strings(permissions)
		sort.Strings(domains)
		result = append(result, roleTemplateView{
			Name:        string(definition.Name),
			Description: definition.Description,
			Permissions: permissions,
			Domains:     domains,
			Builtin:     true,
			Manageable:  authz.CanManageRoleSet(actorRoles, []string{string(definition.Name)}),
		})
	}
	return result
}

func canGrantPermissions(actorPermissions, targetPermissions []string) bool {
	allowed := make(map[string]struct{}, len(actorPermissions))
	for _, permission := range actorPermissions {
		permission = strings.ToLower(strings.TrimSpace(permission))
		if permission == "" {
			continue
		}
		allowed[permission] = struct{}{}
	}
	if len(allowed) == 0 {
		return false
	}
	for _, permission := range targetPermissions {
		permission = strings.ToLower(strings.TrimSpace(permission))
		if permission == "" {
			continue
		}
		if _, ok := allowed[permission]; !ok {
			return false
		}
	}
	return true
}

func canManageBusinessDomains(actorRoles, targetDomains []string) bool {
	if authz.HasRole(actorRoles, authz.StaffRoleSuperAdmin) {
		return true
	}

	actorDomains := domainStringSet(authz.DomainsForRoles(actorRoles))
	if len(actorDomains) == 0 {
		return false
	}
	if _, ok := actorDomains[string(authz.BusinessDomainPlatform)]; ok {
		return true
	}
	for _, domain := range targetDomains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}
		if _, ok := actorDomains[domain]; !ok {
			return false
		}
	}
	return len(targetDomains) > 0
}

func canManageRoleNamesWithCatalog(actorRoles, targetRoleNames []string, catalog []*upbv1.StaffRole) bool {
	if authz.HasRole(actorRoles, authz.StaffRoleSuperAdmin) {
		return true
	}

	roleDomains := make(map[string][]string, len(catalog))
	for _, role := range catalog {
		if role == nil {
			continue
		}
		roleDomains[strings.ToLower(strings.TrimSpace(role.GetName()))] = normalizeStringList(role.GetDomains())
	}
	for _, definition := range authz.BuiltinRoleDefinitions() {
		name := string(definition.Name)
		if _, ok := roleDomains[name]; ok {
			continue
		}
		roleDomains[name] = domainsFromDefinition(definition.Domains)
	}

	targetDomains := make([]string, 0, len(targetRoleNames))
	for _, roleName := range targetRoleNames {
		roleName = strings.ToLower(strings.TrimSpace(roleName))
		if roleName == "" {
			continue
		}
		if roleName == string(authz.StaffRoleSuperAdmin) {
			return false
		}
		domains, ok := roleDomains[roleName]
		if !ok || len(domains) == 0 {
			return false
		}
		targetDomains = append(targetDomains, domains...)
	}
	return canManageBusinessDomains(actorRoles, targetDomains)
}

func normalizeStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

func domainStringSet(domains []authz.BusinessDomain) map[string]struct{} {
	set := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		set[string(domain)] = struct{}{}
	}
	return set
}

func domainsFromDefinition(domains []authz.BusinessDomain) []string {
	values := make([]string, 0, len(domains))
	for _, domain := range domains {
		values = append(values, string(domain))
	}
	sort.Strings(values)
	return values
}

func writeUserRPCError(ctx *gin.Context, err error, fallback string) {
	if ctx == nil {
		return
	}

	httpCode := http.StatusBadGateway
	switch status.Code(err) {
	case codes.InvalidArgument, codes.FailedPrecondition:
		httpCode = http.StatusBadRequest
	case codes.NotFound:
		httpCode = http.StatusNotFound
	case codes.AlreadyExists:
		httpCode = http.StatusConflict
	case codes.PermissionDenied:
		httpCode = http.StatusForbidden
	case codes.Unauthenticated:
		httpCode = http.StatusUnauthorized
	}

	body := gin.H{
		"code": httpCode,
		"msg":  fallback,
	}
	if detail := strings.TrimSpace(status.Convert(err).Message()); detail != "" && detail != fallback {
		body["detail"] = detail
	}
	ctx.JSON(httpCode, body)
}
