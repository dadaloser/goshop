package user

import (
	"context"
	"encoding/json"
	"goshop/app/pkg/bizcode"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goshop/app/goshop/api/internal/data"
	goodsv1 "goshop/app/goshop/api/internal/service/goods/v1"
	inventoryv1 "goshop/app/goshop/api/internal/service/inventory/v1"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	smsv1 "goshop/app/goshop/api/internal/service/sms/v1"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
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
			assertUserErrorCodeFromErr(t, err, bizcode.ErrConnectGRPC)
		})
	}
}

func TestWriteLoginResponseRejectsNilDTO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writeLoginResponse(ctx, nil)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	assertUserErrorCode(t, recorder.Body.Bytes(), bizcode.ErrConnectGRPC)
}

func TestGetUserDetailRejectsMissingUserService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &userServer{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middlewares.KeyUserID, float64(1))

	server.GetUserDetail(ctx)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	assertUserErrorCode(t, recorder.Body.Bytes(), bizcode.ErrConnectGRPC)
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

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
	}
	assertUserErrorCode(t, recorder.Body.Bytes(), bizcode.ErrConnectGRPC)
	if userSrv.updateCalled {
		t.Fatal("UpdateUser reached Update after nil Get response")
	}
}

func TestUpdateUserReturnsFriendlyValidationAndSuccessMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("invalid birthday", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPatch, "/user/update", strings.NewReader(`{"name":"alice","gender":"female","birthday":"not-a-date"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")

		(&userServer{}).UpdateUser(ctx)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("UpdateUser() status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if strings.Contains(recorder.Body.String(), "UpdateUserForm") || strings.Contains(recorder.Body.String(), "datetime") {
			t.Fatalf("UpdateUser() leaked validator details: %s", recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), "生日格式应为 YYYY-MM-DD") {
			t.Fatalf("UpdateUser() validation body = %s, want friendly birthday message", recorder.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPatch, "/user/update", strings.NewReader(`{"gender":"female","birthday":"2000-01-02"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")

		(&userServer{}).UpdateUser(ctx)

		if !strings.Contains(recorder.Body.String(), "昵称长度应为 3 到 10 个字符") {
			t.Fatalf("UpdateUser() validation body = %s, want friendly name message", recorder.Body.String())
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPatch, "/user/update", strings.NewReader(`{"name":`))
		ctx.Request.Header.Set("Content-Type", "application/json")

		(&userServer{}).UpdateUser(ctx)

		if !strings.Contains(recorder.Body.String(), "请求内容格式不正确") {
			t.Fatalf("UpdateUser() validation body = %s, want safe JSON message", recorder.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		userSrv := &fakeUserSrv{user: &userv1.UserDTO{}}
		server := &userServer{sf: &fakeUserServiceFactory{users: userSrv}}
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPatch, "/user/update", strings.NewReader(`{"name":"alice","gender":"female","birthday":"2000-01-02"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Set(middlewares.KeyUserID, float64(1))

		server.UpdateUser(ctx)

		if recorder.Code != http.StatusOK {
			t.Fatalf("UpdateUser() status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), "修改成功") {
			t.Fatalf("UpdateUser() body = %s, want success message", recorder.Body.String())
		}
	})
}

func TestOwnDeviceBlacklistUsesAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userSrv := &fakeUserSrv{
		blacklist: data.DeviceBlacklistList{
			TotalCount: 1,
			Items:      []data.DeviceBlacklist{{UserID: 42, DeviceID: "device-a"}},
		},
	}
	server := &userServer{sf: &fakeUserServiceFactory{users: userSrv}}

	t.Run("list", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/user/device_blacklist/self", nil)
		ctx.Set(middlewares.KeyUserID, float64(42))

		server.ListOwnDeviceBlacklist(ctx)

		if recorder.Code != http.StatusOK || userSrv.listBlacklistUserID != 42 {
			t.Fatalf("ListOwnDeviceBlacklist() status=%d user_id=%d, want status=200 user_id=42", recorder.Code, userSrv.listBlacklistUserID)
		}
		if strings.Contains(recorder.Body.String(), `"user_id"`) {
			t.Fatalf("ListOwnDeviceBlacklist() exposed a user identifier: %s", recorder.Body.String())
		}
	})

	t.Run("add ignores client supplied user id", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/user/device_blacklist/self", strings.NewReader(`{"user_id":43,"device_id":"device-a"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Set(middlewares.KeyUserID, float64(42))

		server.AddOwnDeviceBlacklist(ctx)

		if recorder.Code != http.StatusOK || userSrv.addBlacklistUserID != 42 || userSrv.addBlacklistDeviceID != "device-a" {
			t.Fatalf("AddOwnDeviceBlacklist() status=%d user_id=%d device_id=%q, want status=200 user_id=42 device_id=device-a", recorder.Code, userSrv.addBlacklistUserID, userSrv.addBlacklistDeviceID)
		}
	})

	t.Run("delete", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodDelete, "/user/device_blacklist/self/device-a", nil)
		ctx.Params = gin.Params{{Key: "device_id", Value: "device-a"}}
		ctx.Set(middlewares.KeyUserID, float64(42))

		server.DeleteOwnDeviceBlacklist(ctx)

		if recorder.Code != http.StatusOK || userSrv.deleteBlacklistUserID != 42 || userSrv.deleteBlacklistDeviceID != "device-a" {
			t.Fatalf("DeleteOwnDeviceBlacklist() status=%d user_id=%d device_id=%q, want status=200 user_id=42 device_id=device-a", recorder.Code, userSrv.deleteBlacklistUserID, userSrv.deleteBlacklistDeviceID)
		}
	})
}

func TestListDevicesReturnsDeviceIPAndLastOperationTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lastOperation := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	userSrv := &fakeUserSrv{sessions: data.SessionList{
		TotalCount: 1,
		Items: []data.Session{{
			ID:         "session-a",
			DeviceID:   "device-a",
			DeviceName: "iPhone 16",
			ClientIP:   "203.0.113.10",
			LastUsedAt: lastOperation,
			Active:     true,
		}},
	}}
	server := &userServer{sf: &fakeUserServiceFactory{users: userSrv}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/user/devices", nil)
	ctx.Set(middlewares.KeyUserID, float64(42))

	server.ListDevices(ctx)

	if recorder.Code != http.StatusOK || userSrv.listDevicesUserID != 42 {
		t.Fatalf("ListDevices() status=%d user_id=%d, want status=200 user_id=42", recorder.Code, userSrv.listDevicesUserID)
	}
	for _, field := range []string{`"device":"iPhone 16"`, `"ip_address":"203.0.113.10"`, `"last_operation_at":"2026-07-31T12:00:00Z"`, `"session_id":"session-a"`} {
		if !strings.Contains(recorder.Body.String(), field) {
			t.Fatalf("ListDevices() body = %s, want %s", recorder.Body.String(), field)
		}
	}
}

func TestListDevicesHidesInternalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userSrv := &fakeUserSrv{
		listDevicesErr: errors.NewSpec(bizcode.DeviceSessionUnavailableSpec, "unknown column client_ip"),
	}
	server := &userServer{sf: &fakeUserServiceFactory{users: userSrv}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/user/devices", nil)
	ctx.Set(middlewares.KeyUserID, float64(42))

	server.ListDevices(ctx)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("ListDevices() status = %d, want %d body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "设备获取失败，请稍后重试") || strings.Contains(recorder.Body.String(), "Database error") || strings.Contains(recorder.Body.String(), "unknown column") {
		t.Fatalf("ListDevices() exposed an internal failure: %s", recorder.Body.String())
	}
}

func TestLogoutDeviceUsesAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userSrv := &fakeUserSrv{}
	server := &userServer{sf: &fakeUserServiceFactory{users: userSrv}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/user/devices/session-a/logout", nil)
	ctx.Params = gin.Params{{Key: "session_id", Value: "session-a"}}
	ctx.Set(middlewares.KeyUserID, float64(42))

	server.LogoutDevice(ctx)

	if recorder.Code != http.StatusOK || userSrv.logoutDeviceUserID != 42 || userSrv.logoutDeviceSessionID != "session-a" {
		t.Fatalf("LogoutDevice() status=%d user_id=%d session_id=%q, want status=200 user_id=42 session_id=session-a", recorder.Code, userSrv.logoutDeviceUserID, userSrv.logoutDeviceSessionID)
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
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["code"] != float64(200) || response["msg"] != "用户已注销" {
		t.Fatalf(`"response = %#v, want code=200 msg=用户已注销"`, response)
	}

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
	user                    *userv1.UserDTO
	updateCalled            bool
	logoutAllUserID         uint64
	deleteAccountUserID     uint64
	deleteAccountPassword   string
	sessions                data.SessionList
	listDevicesErr          error
	listDevicesUserID       uint64
	logoutDeviceUserID      uint64
	logoutDeviceSessionID   string
	blacklist               data.DeviceBlacklistList
	listBlacklistUserID     uint64
	addBlacklistUserID      uint64
	addBlacklistDeviceID    string
	deleteBlacklistUserID   uint64
	deleteBlacklistDeviceID string
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
	return f.user, nil
}

func (f *fakeUserSrv) GetByUsername(context.Context, string) (*userv1.UserDTO, error) {
	return nil, nil
}

func (f *fakeUserSrv) LogoutAll(_ context.Context, userID uint64) error {
	f.logoutAllUserID = userID
	return nil
}

func (f *fakeUserSrv) Logout(context.Context, uint64, string) error { return nil }

func (f *fakeUserSrv) Refresh(context.Context, string, string) (*userv1.UserDTO, error) {
	return nil, nil
}

func (f *fakeUserSrv) ListDevices(_ context.Context, userID uint64, _ int, _ int) (data.SessionList, error) {
	f.listDevicesUserID = userID
	return f.sessions, f.listDevicesErr
}
func (f *fakeUserSrv) LogoutDevice(_ context.Context, userID uint64, sessionID string) error {
	f.logoutDeviceUserID = userID
	f.logoutDeviceSessionID = sessionID
	return nil
}
func (f *fakeUserSrv) ListDeviceBlacklist(_ context.Context, userID uint64, _ int, _ int) (data.DeviceBlacklistList, error) {
	f.listBlacklistUserID = userID
	return f.blacklist, nil
}
func (f *fakeUserSrv) AddDeviceBlacklist(_ context.Context, userID uint64, deviceID string) error {
	f.addBlacklistUserID = userID
	f.addBlacklistDeviceID = deviceID
	return nil
}
func (f *fakeUserSrv) DeleteDeviceBlacklist(_ context.Context, userID uint64, deviceID string) error {
	f.deleteBlacklistUserID = userID
	f.deleteBlacklistDeviceID = deviceID
	return nil
}

func (f *fakeUserSrv) DeleteAccount(_ context.Context, userID uint64, password string) error {
	f.deleteAccountUserID = userID
	f.deleteAccountPassword = password
	return nil
}
