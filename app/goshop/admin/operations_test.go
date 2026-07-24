package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
)

func TestRequireResourceScopeRejectsCrossDomainAndStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name, domain, store, team string
		claims                    map[string]any
		want                      int
	}{
		{name: "allowed catalog store", domain: "catalog", store: "store-a", claims: map[string]any{"resource_domains": []string{"catalog"}, "resource_stores": []string{"store-a"}}, want: http.StatusNoContent},
		{name: "cross domain", domain: "operations", store: "store-a", claims: map[string]any{"resource_domains": []string{"catalog"}, "resource_stores": []string{"store-a"}}, want: http.StatusForbidden},
		{name: "cross store", domain: "catalog", store: "store-b", claims: map[string]any{"resource_domains": []string{"catalog"}, "resource_stores": []string{"store-a"}}, want: http.StatusForbidden},
		{name: "catalog requires store", domain: "catalog", claims: map[string]any{"resource_domains": []string{"catalog"}}, want: http.StatusForbidden},
		{name: "ops requires team", domain: "operations", team: "warehouse-a", claims: map[string]any{"resource_domains": []string{"operations"}, "resource_teams": []string{"warehouse-a"}}, want: http.StatusNoContent},
		{name: "ops rejects store shape", domain: "operations", store: "store-a", claims: map[string]any{"resource_domains": []string{"operations"}, "resource_stores": []string{"store-a"}}, want: http.StatusForbidden},
		{name: "platform rejects store shape", domain: "platform", store: "store-a", claims: map[string]any{"resource_domains": []string{"platform"}}, want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(c *gin.Context) {
				c.Set(middlewares.JWTPayloadKey, tt.claims)
				requireResourceScope(authz.BusinessDomain(tt.domain))(c)
			}, func(c *gin.Context) { c.Status(http.StatusNoContent) })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Resource-Domain", tt.domain)
			req.Header.Set("X-Store-ID", tt.store)
			req.Header.Set("X-Team-ID", tt.team)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d", rec.Code, tt.want)
			}
		})
	}
}

func TestBuiltinOperationalRolesAreMutuallyScoped(t *testing.T) {
	definitions := map[authz.StaffRole]authz.RoleDefinition{}
	for _, definition := range authz.BuiltinRoleDefinitions() {
		definitions[definition.Name] = definition
	}
	contains := func(role authz.StaffRole, permission authz.Permission) bool {
		for _, candidate := range definitions[role].Permissions {
			if candidate == permission {
				return true
			}
		}
		return false
	}
	if contains(authz.StaffRoleCatalog, authz.PermissionInventoryWriteAny) {
		t.Fatal("catalog role can write inventory")
	}
	if contains(authz.StaffRoleOps, authz.PermissionOrderRefundAny) {
		t.Fatal("ops role can refund orders")
	}
	if contains(authz.StaffRoleFinance, authz.PermissionOrderCloseAny) {
		t.Fatal("finance role can close orders")
	}
	if !contains(authz.StaffRoleFinance, authz.PermissionOrderRefundAny) {
		t.Fatal("finance role cannot refund orders")
	}
}
