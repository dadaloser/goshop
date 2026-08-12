package rpc

import (
	"context"
	"goshop/app/pkg/errorcatalog"
	"strconv"
	"testing"

	upbv1 "goshop/api/user/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/bizcode"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestUserRPCErrorPreservesValidationCodeForCreate(t *testing.T) {
	err := userRPCError(status.Error(codes.InvalidArgument, "password is too weak"), errcode.ErrValidation)
	if !errors.IsCode(err, errcode.ErrValidation) {
		t.Fatalf("userRPCError(InvalidArgument) = %v, want code %d", err, errcode.ErrValidation)
	}
}

func TestUserRPCErrorMapsAbortedToSpecificUserConflict(t *testing.T) {
	tests := []struct {
		message string
		code    int
	}{
		{message: "手机号已存在", code: bizcode.ErrUserMobileAlreadyExists},
		{message: "邮箱已存在", code: bizcode.ErrUserEmailAlreadyExists},
		{message: "用户名已存在", code: bizcode.ErrUsernameAlreadyExists},
	}
	for _, tt := range tests {
		err := userRPCError(newBusinessGRPCError(t, codes.Aborted, tt.code, tt.message), errcode.ErrValidation)
		if !errors.IsCode(err, tt.code) {
			t.Errorf("userRPCError(Aborted, %q) = %v, want code %d", tt.message, err, tt.code)
		}
		spec, ok := errors.SpecOf(err)
		if !ok || spec.Message != tt.message {
			t.Errorf("userRPCError(Aborted, %q) public message = %q, want %q", tt.message, spec.Message, tt.message)
		}
	}
}

func TestListUserSessionsHidesInternalServiceFailures(t *testing.T) {
	errorcatalog.RegisterAll()
	store := NewUsers(&fakeUserClient{
		listUserSessionsErr: newBusinessGRPCError(t, codes.Unavailable, errcode.ErrDatabase, "unknown column client_ip"),
	})

	_, err := store.ListUserSessions(context.Background(), 1, 1, 20)
	if !errors.IsCode(err, bizcode.ErrDeviceSessionUnavailable) {
		t.Fatalf("ListUserSessions() error = %v, want code %d", err, bizcode.ErrDeviceSessionUnavailable)
	}
	if spec, ok := errors.SpecOf(err); !ok || spec.Message != "设备获取失败，请稍后重试" {
		t.Fatalf("ListUserSessions() spec = %#v, want safe device-list message", spec)
	}
}

func newBusinessGRPCError(t *testing.T, grpcCode codes.Code, businessCode int, message string) error {
	t.Helper()
	grpcStatus := status.New(grpcCode, message)
	withDetails, err := grpcStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: "GOSHOP_BUSINESS_ERROR",
		Domain: "goshop",
		Metadata: map[string]string{
			"business_code":  strconv.Itoa(businessCode),
			"public_message": message,
		},
	})
	if err != nil {
		t.Fatalf("status.WithDetails() error = %v", err)
	}
	return withDetails.Err()
}

func TestUsersRejectInvalidInputBeforeRPC(t *testing.T) {
	client := &fakeUserClient{}
	store := NewUsers(client)

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "create nil user",
			run: func() error {
				_, err := store.Create(context.Background(), nil)
				return err
			},
			code: errcode.ErrValidation,
		},
		{
			name: "update nil user",
			run: func() error {
				return store.Update(context.Background(), nil)
			},
			code: errcode.ErrValidation,
		},
		{
			name: "update zero id",
			run: func() error {
				return store.Update(context.Background(), &data.User{})
			},
			code: errcode.ErrValidation,
		},
		{
			name: "get zero id",
			run: func() error {
				_, err := store.Get(context.Background(), 0)
				return err
			},
			code: bizcode.ErrUserNotFound,
		},
		{
			name: "delete zero id",
			run: func() error {
				return store.Delete(context.Background(), 0)
			},
			code: bizcode.ErrUserNotFound,
		},
		{
			name: "get empty username",
			run: func() error {
				_, err := store.GetByUsername(context.Background(), " ")
				return err
			},
			code: bizcode.ErrUserNotFound,
		},
		{
			name: "check password empty hash",
			run: func() error {
				return store.CheckPassWord(context.Background(), "secret", " ")
			},
			code: bizcode.ErrUserPasswordIncorrect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %d", err, tt.code)
			}
		})
	}
	if client.called {
		t.Fatal("invalid input reached user RPC client")
	}
}

func TestUsersNormalizeUsernameBeforeRPC(t *testing.T) {
	client := &fakeUserClient{
		user: &upbv1.UserInfoResponse{Id: 1, Username: "user_001", NickName: "tester", Status: "disabled"},
	}
	store := NewUsers(client)

	got, err := store.GetByUsername(context.Background(), " USER@example.COM ")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if client.mobileRequest != "user@example.com" {
		t.Fatalf("mobile request = %q, want user@example.com", client.mobileRequest)
	}
	if got.Username != "user_001" {
		t.Fatalf("GetByUsername() username = %q, want user_001", got.Username)
	}
	if got.Status != "disabled" {
		t.Fatalf("GetByUsername() status = %q, want disabled", got.Status)
	}
}

func TestUsersCreateAndUpdateForwardUsername(t *testing.T) {
	client := &fakeUserClient{createResponse: &upbv1.UserInfoResponse{Id: 1, Status: "active"}}
	store := NewUsers(client)

	user := &data.UserCreate{
		Username: "user_001",
		Mobile:   "13800138000",
		Email:    "user@example.com",
		NickName: "tester",
		PassWord: "Strong1!",
	}
	created, err := store.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if client.createRequest == nil {
		t.Fatal("Create() did not call user RPC client")
	}
	if client.createRequest.Username != "user_001" {
		t.Fatalf("Create() username = %q, want user_001", client.createRequest.Username)
	}
	if created.Status != "active" {
		t.Fatalf("Create() status = %q, want active", created.Status)
	}

	if err := store.Update(context.Background(), &data.User{ID: 1, Username: "user_002"}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if client.updateRequest == nil {
		t.Fatal("Update() did not call user RPC client")
	}
	if client.updateRequest.Username != "user_002" {
		t.Fatalf("Update() username = %q, want user_002", client.updateRequest.Username)
	}
}

func TestUsersHandleNilRPCResponses(t *testing.T) {
	store := NewUsers(&fakeUserClient{returnNil: true})

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "create nil response",
			run: func() error {
				_, err := store.Create(context.Background(), &data.UserCreate{Mobile: "13800138000"})
				return err
			},
			code: bizcode.ErrConnectGRPC,
		},
		{
			name: "get nil response",
			run: func() error {
				_, err := store.Get(context.Background(), 1)
				return err
			},
			code: bizcode.ErrUserNotFound,
		},
		{
			name: "get username nil response",
			run: func() error {
				_, err := store.GetByUsername(context.Background(), "user@example.com")
				return err
			},
			code: bizcode.ErrUserNotFound,
		},
		{
			name: "check password nil response",
			run: func() error {
				return store.CheckPassWord(context.Background(), "secret", "hashed")
			},
			code: bizcode.ErrUserPasswordIncorrect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %d", err, tt.code)
			}
		})
	}
}

type fakeUserClient struct {
	upbv1.UserClient
	called              bool
	returnNil           bool
	mobileRequest       string
	user                *upbv1.UserInfoResponse
	authUser            *upbv1.UserAuthResponse
	createRequest       *upbv1.CreateUserInfo
	createResponse      *upbv1.UserInfoResponse
	updateRequest       *upbv1.UpdateUserInfo
	statusRequest       *upbv1.UpdateUserStatusRequest
	createStaffReq      *upbv1.CreateStaffUserRequest
	listUserSessionsErr error
}

func (f *fakeUserClient) ListUserSessions(context.Context, *upbv1.ListUserSessionsRequest, ...grpc.CallOption) (*upbv1.ListUserSessionsResponse, error) {
	f.called = true
	if f.listUserSessionsErr != nil {
		return nil, f.listUserSessionsErr
	}
	return &upbv1.ListUserSessionsResponse{}, nil
}

func (f *fakeUserClient) GetUserList(context.Context, *upbv1.PageInfo, ...grpc.CallOption) (*upbv1.UserListResponse, error) {
	f.called = true
	return &upbv1.UserListResponse{}, nil
}

func (f *fakeUserClient) GetUserByMobile(_ context.Context, in *upbv1.MobileRequest, _ ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	f.called = true
	if in != nil {
		f.mobileRequest = in.Mobile
	}
	if f.returnNil {
		return nil, nil
	}
	if f.user != nil {
		return f.user, nil
	}
	return &upbv1.UserInfoResponse{}, nil
}

func (f *fakeUserClient) GetUserById(context.Context, *upbv1.IdRequest, ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	f.called = true
	if f.returnNil {
		return nil, nil
	}
	if f.user != nil {
		return f.user, nil
	}
	return &upbv1.UserInfoResponse{}, nil
}

func (f *fakeUserClient) GetUserAuthByMobile(_ context.Context, in *upbv1.MobileRequest, _ ...grpc.CallOption) (*upbv1.UserAuthResponse, error) {
	f.called = true
	if in != nil {
		f.mobileRequest = in.Mobile
	}
	if f.returnNil {
		return nil, nil
	}
	if f.authUser != nil {
		return f.authUser, nil
	}
	return &upbv1.UserAuthResponse{}, nil
}

func (f *fakeUserClient) GetUserAuthById(context.Context, *upbv1.IdRequest, ...grpc.CallOption) (*upbv1.UserAuthResponse, error) {
	f.called = true
	if f.returnNil {
		return nil, nil
	}
	if f.authUser != nil {
		return f.authUser, nil
	}
	return &upbv1.UserAuthResponse{}, nil
}

func (f *fakeUserClient) ListStaffRoles(context.Context, *emptypb.Empty, ...grpc.CallOption) (*upbv1.StaffRoleListResponse, error) {
	f.called = true
	return &upbv1.StaffRoleListResponse{}, nil
}

func (f *fakeUserClient) CreateStaffRole(context.Context, *upbv1.CreateStaffRoleRequest, ...grpc.CallOption) (*upbv1.StaffRole, error) {
	f.called = true
	return &upbv1.StaffRole{}, nil
}

func (f *fakeUserClient) UpdateStaffRole(context.Context, *upbv1.UpdateStaffRoleRequest, ...grpc.CallOption) (*upbv1.StaffRole, error) {
	f.called = true
	return &upbv1.StaffRole{}, nil
}

func (f *fakeUserClient) DeleteStaffRole(context.Context, *upbv1.DeleteStaffRoleRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	f.called = true
	return &emptypb.Empty{}, nil
}

func (f *fakeUserClient) GetUserStaffRoles(context.Context, *upbv1.IdRequest, ...grpc.CallOption) (*upbv1.UserRoleBindingResponse, error) {
	f.called = true
	return &upbv1.UserRoleBindingResponse{}, nil
}

func (f *fakeUserClient) ReplaceUserStaffRoles(context.Context, *upbv1.ReplaceUserStaffRolesRequest, ...grpc.CallOption) (*upbv1.UserRoleBindingResponse, error) {
	f.called = true
	return &upbv1.UserRoleBindingResponse{}, nil
}

func (f *fakeUserClient) ListUserAuditLogs(context.Context, *upbv1.UserAuditLogPageRequest, ...grpc.CallOption) (*upbv1.UserAuditLogListResponse, error) {
	f.called = true
	return &upbv1.UserAuditLogListResponse{}, nil
}

func (f *fakeUserClient) CreateAdminAuditLog(context.Context, *upbv1.CreateAdminAuditLogRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	f.called = true
	return &emptypb.Empty{}, nil
}

func (f *fakeUserClient) ListAdminAuditLogs(context.Context, *upbv1.AdminAuditLogPageRequest, ...grpc.CallOption) (*upbv1.AdminAuditLogListResponse, error) {
	f.called = true
	return &upbv1.AdminAuditLogListResponse{}, nil
}

func (f *fakeUserClient) CreateUser(_ context.Context, in *upbv1.CreateUserInfo, _ ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	f.called = true
	f.createRequest = in
	if f.returnNil {
		return nil, nil
	}
	if f.createResponse != nil {
		return f.createResponse, nil
	}
	return &upbv1.UserInfoResponse{Id: 1}, nil
}

func (f *fakeUserClient) CreateStaffUser(_ context.Context, in *upbv1.CreateStaffUserRequest, _ ...grpc.CallOption) (*upbv1.StaffUserResponse, error) {
	f.called = true
	f.createStaffReq = in
	if f.returnNil {
		return nil, nil
	}
	return &upbv1.StaffUserResponse{}, nil
}

func (f *fakeUserClient) UpdateUser(_ context.Context, in *upbv1.UpdateUserInfo, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	f.called = true
	f.updateRequest = in
	return &emptypb.Empty{}, nil
}

func (f *fakeUserClient) UpdateUserStatus(_ context.Context, in *upbv1.UpdateUserStatusRequest, _ ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	f.called = true
	f.statusRequest = in
	if f.returnNil {
		return nil, nil
	}
	return &upbv1.UserInfoResponse{Id: in.GetId(), Status: in.GetStatus()}, nil
}

func (f *fakeUserClient) DeleteUser(context.Context, *upbv1.IdRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	f.called = true
	return &emptypb.Empty{}, nil
}

func (f *fakeUserClient) CheckPassWord(context.Context, *upbv1.PasswordCheckInfo, ...grpc.CallOption) (*upbv1.CheckResponse, error) {
	f.called = true
	if f.returnNil {
		return nil, nil
	}
	return &upbv1.CheckResponse{Success: true}, nil
}

var _ upbv1.UserClient = &fakeUserClient{}
