package user

import (
	"context"
	"testing"

	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestUserServerRejectsNilRequests(t *testing.T) {
	server := &userServer{}

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "create user", run: func() error { _, err := server.CreateUser(context.Background(), nil); return err }},
		{name: "create staff user", run: func() error { _, err := server.CreateStaffUser(context.Background(), nil); return err }},
		{name: "create staff role", run: func() error { _, err := server.CreateStaffRole(context.Background(), nil); return err }},
		{name: "update staff role", run: func() error { _, err := server.UpdateStaffRole(context.Background(), nil); return err }},
		{name: "delete staff role", run: func() error { _, err := server.DeleteStaffRole(context.Background(), nil); return err }},
		{name: "update user", run: func() error { _, err := server.UpdateUser(context.Background(), nil); return err }},
		{name: "update user status", run: func() error { _, err := server.UpdateUserStatus(context.Background(), nil); return err }},
		{name: "delete user", run: func() error { _, err := server.DeleteUser(context.Background(), nil); return err }},
		{name: "get user list", run: func() error { _, err := server.GetUserList(context.Background(), nil); return err }},
		{name: "get user audit logs", run: func() error { _, err := server.ListUserAuditLogs(context.Background(), nil); return err }},
		{name: "get user by id", run: func() error { _, err := server.GetUserById(context.Background(), nil); return err }},
		{name: "get user by mobile", run: func() error { _, err := server.GetUserByMobile(context.Background(), nil); return err }},
		{name: "check password", run: func() error { _, err := server.CheckPassWord(context.Background(), nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, code2.ErrValidation) {
				t.Fatalf("error = %v, want code %d", err, code2.ErrValidation)
			}
		})
	}
}
