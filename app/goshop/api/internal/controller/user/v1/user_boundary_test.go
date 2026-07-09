package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goodsv1 "goshop/app/goshop/api/internal/service/goods/v1"
	inventoryv1 "goshop/app/goshop/api/internal/service/inventory/v1"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	smsv1 "goshop/app/goshop/api/internal/service/sms/v1"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/code"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

func TestUserControllerUsersServiceRejectsMissingDependencies(t *testing.T) {
	tests := []struct {
		name   string
		server *userServer
	}{
		{
			name:   "nil controller",
			server: nil,
		},
		{
			name:   "nil service factory",
			server: &userServer{},
		},
		{
			name:   "nil user service",
			server: &userServer{sf: &fakeUserServiceFactory{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.server.usersService()
			assertUserErrorCodeFromErr(t, err, code.ErrConnectGRPC)
		})
	}
}

func TestWriteLoginResponseRejectsNilDTO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writeLoginResponse(ctx, nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertUserErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestGetUserDetailRejectsMissingUserService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &userServer{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middlewares.KeyUserID, float64(1))

	server.GetUserDetail(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertUserErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestUpdateUserRejectsNilUserResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userSrv := &fakeUserSrv{}
	server := &userServer{sf: &fakeUserServiceFactory{users: userSrv}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/user?name=alice&gender=male&birthday=2000-01-02", nil)
	ctx.Set(middlewares.KeyUserID, float64(1))

	server.UpdateUser(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}
	assertUserErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
	if userSrv.updateCalled {
		t.Fatal("UpdateUser reached Update after nil Get response")
	}
}

func TestLogoutAllCallsUserService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userSrv := &fakeUserSrv{}
	server := &userServer{sf: &fakeUserServiceFactory{users: userSrv}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/user/logout_all", nil)
	ctx.Set(middlewares.KeyUserID, float64(11))

	server.LogoutAll(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if userSrv.logoutAllUserID != 11 {
		t.Fatalf("logout all user id = %d, want 11", userSrv.logoutAllUserID)
	}
}

func TestDeleteAccountCallsUserService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userSrv := &fakeUserSrv{}
	server := &userServer{sf: &fakeUserServiceFactory{users: userSrv}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/user/account", strings.NewReader(`{"password":"secret"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(middlewares.KeyUserID, float64(12))

	server.DeleteAccount(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if userSrv.deleteAccountUserID != 12 {
		t.Fatalf("delete account user id = %d, want 12", userSrv.deleteAccountUserID)
	}
	if userSrv.deleteAccountPassword != "secret" {
		t.Fatalf("delete account password = %q, want secret", userSrv.deleteAccountPassword)
	}
}

func assertUserErrorCodeFromErr(t *testing.T, err error, want int) {
	t.Helper()

	if !errors.IsCode(err, want) {
		t.Fatalf("error = %v, want code %d", err, want)
	}
}

func assertUserErrorCode(t *testing.T, body []byte, want int) {
	t.Helper()

	var got struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if got.Code != want {
		t.Fatalf("code = %d, want %d", got.Code, want)
	}
}

type fakeUserServiceFactory struct {
	users userv1.UserSrv
}

func (f *fakeUserServiceFactory) Goods() goodsv1.GoodsSrv {
	return nil
}

func (f *fakeUserServiceFactory) Orders() orderv1.OrderSrv {
	return nil
}

func (f *fakeUserServiceFactory) Inventory() inventoryv1.InventorySrv {
	return nil
}

func (f *fakeUserServiceFactory) Users() userv1.UserSrv {
	return f.users
}

func (f *fakeUserServiceFactory) Sms() smsv1.SmsSrv {
	return nil
}

type fakeUserSrv struct {
	updateCalled          bool
	logoutAllUserID       uint64
	deleteAccountUserID   uint64
	deleteAccountPassword string
}

func (f *fakeUserSrv) PasswordLogin(context.Context, string, string) (*userv1.UserDTO, error) {
	return nil, nil
}

func (f *fakeUserSrv) SmsLogin(context.Context, string, string) (*userv1.UserDTO, error) {
	return nil, nil
}

func (f *fakeUserSrv) Register(context.Context, string, string, string, string, string, string) (*userv1.UserDTO, error) {
	return nil, nil
}

func (f *fakeUserSrv) Update(context.Context, *userv1.UserDTO) error {
	f.updateCalled = true
	return nil
}

func (f *fakeUserSrv) Get(context.Context, uint64) (*userv1.UserDTO, error) {
	return nil, nil
}

func (f *fakeUserSrv) GetByUsername(context.Context, string) (*userv1.UserDTO, error) {
	return nil, nil
}

func (f *fakeUserSrv) LogoutAll(_ context.Context, userID uint64) error {
	f.logoutAllUserID = userID
	return nil
}

func (f *fakeUserSrv) DeleteAccount(_ context.Context, userID uint64, password string) error {
	f.deleteAccountUserID = userID
	f.deleteAccountPassword = password
	return nil
}
