package user

import (
	"context"
	"testing"

	upbv1 "goshop/api/user/v1"
	"goshop/pkg/common/auth"
)

func TestCheckPassWordAcceptsBcryptHash(t *testing.T) {
	hashedPassword, err := auth.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	got, err := (&userServer{}).CheckPassWord(context.Background(), &upbv1.PasswordCheckInfo{
		Password:          "secret",
		EncryptedPassword: hashedPassword,
	})
	if err != nil {
		t.Fatalf("CheckPassWord() error = %v", err)
	}
	if !got.Success {
		t.Fatal("CheckPassWord() success = false, want true")
	}
}

func TestCheckPassWordRejectsMalformedHash(t *testing.T) {
	got, err := (&userServer{}).CheckPassWord(context.Background(), &upbv1.PasswordCheckInfo{
		Password:          "secret",
		EncryptedPassword: "not-a-hash",
	})
	if err != nil {
		t.Fatalf("CheckPassWord() error = %v", err)
	}
	if got.Success {
		t.Fatal("CheckPassWord() success = true, want false")
	}
}
