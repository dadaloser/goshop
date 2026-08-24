package authz

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goshop/app/pkg/errorcatalog"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/errcode"
	core "goshop/pkg/transport/httperror"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	errorcatalog.RegisterAll()
	m.Run()
}

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		payload    any
		permission Permission
		wantStatus int
	}{
		{
			name:       "missing principal rejects",
			permission: PermissionOrderReadSelf,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "matching permission passes",
			payload: map[string]any{
				"user_id":        float64(1),
				"principal_type": string(PrincipalCustomer),
				"status":         string(AccountStatusActive),
				"scope":          []any{string(PermissionOrderReadSelf)},
			},
			permission: PermissionOrderReadSelf,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "missing permission rejects",
			payload: map[string]any{
				"user_id":        float64(1),
				"principal_type": string(PrincipalCustomer),
				"status":         string(AccountStatusActive),
				"scope":          []any{string(PermissionCartReadSelf)},
			},
			permission: PermissionOrderReadSelf,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "disabled account rejects",
			payload: map[string]any{
				"user_id":        float64(1),
				"principal_type": string(PrincipalCustomer),
				"status":         string(AccountStatusDisabled),
				"scope":          []any{string(PermissionOrderReadSelf)},
			},
			permission: PermissionOrderReadSelf,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "legacy customer token receives rollout permissions",
			payload: map[string]any{
				"user_id": float64(1),
			},
			permission: PermissionOrderReadSelf,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "staff token without scope rejects",
			payload: map[string]any{
				"user_id":        float64(1),
				"principal_type": string(PrincipalStaff),
				"status":         string(AccountStatusActive),
			},
			permission: PermissionOrderReadAny,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "new customer token without scope rejects",
			payload: map[string]any{
				"user_id":        float64(1),
				"principal_type": string(PrincipalCustomer),
				"status":         string(AccountStatusActive),
			},
			permission: PermissionOrderReadSelf,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "unknown principal type rejects",
			payload: map[string]any{
				"user_id":        float64(1),
				"principal_type": "owner",
				"status":         string(AccountStatusActive),
				"scope":          []any{string(PermissionOrderReadAny)},
			},
			permission: PermissionOrderReadAny,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "zero user id rejects",
			payload: map[string]any{
				"user_id": float64(0),
			},
			permission: PermissionOrderReadSelf,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/resource", func(c *gin.Context) {
				if tt.payload != nil {
					c.Set(middlewares.JWTPayloadKey, tt.payload)
				}
				c.Next()
			}, RequirePermission(tt.permission), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/resource", nil)
			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusNoContent {
				return
			}
			var response core.ErrResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode error response = %v", err)
			}
			if response.Code == http.StatusUnauthorized || response.Code == http.StatusForbidden || response.Code == 0 {
				t.Fatalf("error code = %d, want a business error code", response.Code)
			}
			if response.Code != errcode.ErrTokenInvalid && response.Code != errcode.ErrPermissionDenied {
				t.Fatalf("error code = %d, want authentication or permission error", response.Code)
			}
		})
	}
}

func TestCustomerScopesReturnsCopy(t *testing.T) {
	first := CustomerScopes()
	first[0] = "changed"
	second := CustomerScopes()

	if first[0] == second[0] {
		t.Fatal("CustomerScopes returned shared mutable state")
	}
}

func TestRequirePrincipalTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		payload    any
		allowed    []PrincipalType
		wantStatus int
	}{
		{
			name:       "missing principal rejects",
			allowed:    []PrincipalType{PrincipalStaff},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "staff principal passes",
			payload: map[string]any{
				"user_id":        float64(1),
				"principal_type": string(PrincipalStaff),
				"status":         string(AccountStatusActive),
			},
			allowed:    []PrincipalType{PrincipalStaff},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "bootstrap principal rejected for staff-only route",
			payload: map[string]any{
				"user_id":        float64(1),
				"principal_type": string(PrincipalAdminBootstrap),
				"status":         string(AccountStatusActive),
			},
			allowed:    []PrincipalType{PrincipalStaff},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/resource", func(c *gin.Context) {
				if tt.payload != nil {
					c.Set(middlewares.JWTPayloadKey, tt.payload)
				}
				c.Next()
			}, RequirePrincipalTypes(tt.allowed...), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/resource", nil)
			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusNoContent {
				return
			}
			var response core.ErrResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode error response = %v", err)
			}
			if response.Code == http.StatusUnauthorized || response.Code == http.StatusForbidden || response.Code == 0 {
				t.Fatalf("error code = %d, want a business error code", response.Code)
			}
		})
	}
}
