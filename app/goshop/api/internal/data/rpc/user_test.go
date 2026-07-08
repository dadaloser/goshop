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
		user: &upbv1.UserInfoResponse{Id: 1, NickName: "tester"},
	}
	store := NewUsers(client)

	if _, err := store.GetByUsername(context.Background(), " USER@example.COM "); err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if client.mobileRequest != "user@example.com" {
		t.Fatalf("mobile request = %q, want user@example.com", client.mobileRequest)
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

func (f *fakeUserClient) CreateUser(context.Context, *upbv1.CreateUserInfo, ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	f.called = true
	if f.returnNil {
		return nil, nil
	}
	return &upbv1.UserInfoResponse{Id: 1}, nil
}

func (f *fakeUserClient) UpdateUser(context.Context, *upbv1.UpdateUserInfo, ...grpc.CallOption) (*emptypb.Empty, error) {
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
