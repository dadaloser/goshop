package user

import (
	"context"
	"crypto/sha512"
	"goshop/pkg/common/auth"
	"strings"

	upbv1 "goshop/api/user/v1"

	"github.com/anaskhan96/go-password-encoder"
)

func (us *userServer) CheckPassWord(ctx context.Context, info *upbv1.PasswordCheckInfo) (*upbv1.CheckResponse, error) {
	//校验密码
	if err := auth.Compare(info.EncryptedPassword, info.Password); err == nil {
		return &upbv1.CheckResponse{Success: true}, nil
	}

	passwordInfo := strings.Split(info.EncryptedPassword, "$")
	if len(passwordInfo) < 4 {
		return &upbv1.CheckResponse{Success: false}, nil
	}

	options := &password.Options{16, 100, 32, sha512.New}
	check := password.Verify(info.Password, passwordInfo[2], passwordInfo[3], options)
	return &upbv1.CheckResponse{Success: check}, nil
}
