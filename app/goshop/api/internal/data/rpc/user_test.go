package rpc

import (
	"context"
	"testing"

	upbv1 "goshop/api/user/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

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
				return store.Create(context.Background(), nil)
			},
			code: code2.ErrValidation,
		},
		{
			name: "update nil user",
			run: func() error {
				return store.Update(context.Background(), nil)
			},
			code: code2.ErrValidation,
		},
		{
			name: "update zero id",
			run: func() error {
				return store.Update(context.Background(), &data.User{})
			},
			code: code2.ErrValidation,
		},
		{
			name: "get zero id",
			run: func() error {
				_, err := store.Get(context.Background(), 0)
				return err
			},
			code: code.ErrUserNotFound,
		},
		{
			name: "delete zero id",
			run: func() error {
				return store.Delete(context.Background(), 0)
			},
			code: code.ErrUserNotFound,
		},
		{
			name: "get empty username",
			run: func() error {
				_, err := store.GetByUsername(context.Background(), " ")
				return err
			},
			code: code.ErrUserNotFound,
		},
		{
			name: "check password empty hash",
			run: func() error {
				return store.CheckPassWord(context.Background(), "secret", " ")
			},
			code: code.ErrUserPasswordIncorrect,
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
		user: &upbv1.UserInfoResponse{Id: 1, Username: "user_001", NickName: "tester"},
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
}

func TestUsersCreateAndUpdateForwardUsername(t *testing.T) {
	client := &fakeUserClient{}
	store := NewUsers(client)

	if err := store.Create(context.Background(), &data.User{
		Username: "user_001",
		Mobile:   "13800138000",
		Email:    "user@example.com",
		NickName: "tester",
		PassWord: "Strong1!",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if client.createRequest == nil {
		t.Fatal("Create() did not call user RPC client")
	}
	if client.createRequest.Username != "user_001" {
		t.Fatalf("Create() username = %q, want user_001", client.createRequest.Username)
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
				return store.Create(context.Background(), &data.User{Mobile: "13800138000"})
			},
			code: code.ErrUserAlreadyExists,
		},
		{
			name: "get nil response",
			run: func() error {
				_, err := store.Get(context.Background(), 1)
				return err
			},
			code: code.ErrUserNotFound,
		},
		{
			name: "get username nil response",
			run: func() error {
				_, err := store.GetByUsername(context.Background(), "user@example.com")
				return err
			},
			code: code.ErrUserNotFound,
		},
		{
			name: "check password nil response",
			run: func() error {
				return store.CheckPassWord(context.Background(), "secret", "hashed")
			},
			code: code.ErrUserPasswordIncorrect,
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
	called        bool
	returnNil     bool
	mobileRequest string
	user          *upbv1.UserInfoResponse
	createRequest *upbv1.CreateUserInfo
	updateRequest *upbv1.UpdateUserInfo
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

func (f *fakeUserClient) CreateUser(_ context.Context, in *upbv1.CreateUserInfo, _ ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	f.called = true
	f.createRequest = in
	if f.returnNil {
		return nil, nil
	}
	return &upbv1.UserInfoResponse{Id: 1}, nil
}

func (f *fakeUserClient) UpdateUser(_ context.Context, in *upbv1.UpdateUserInfo, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	f.called = true
	f.updateRequest = in
	return &emptypb.Empty{}, nil
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
