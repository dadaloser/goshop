package bizcode_test

import (
	"testing"

	"goshop/app/pkg/bizcode"
	"goshop/gmicro/errcode"
)

func TestErrorCodesAreUnique(t *testing.T) {
	codes := map[int]string{}
	for name, code := range map[string]int{
		"gmicro.ErrSuccess":                   errcode.ErrSuccess,
		"gmicro.ErrUnknown":                   errcode.ErrUnknown,
		"gmicro.ErrBind":                      errcode.ErrBind,
		"gmicro.ErrValidation":                errcode.ErrValidation,
		"gmicro.ErrTokenInvalid":              errcode.ErrTokenInvalid,
		"gmicro.ErrPageNotFound":              errcode.ErrPageNotFound,
		"gmicro.ErrDatabase":                  errcode.ErrDatabase,
		"gmicro.ErrEncrypt":                   errcode.ErrEncrypt,
		"gmicro.ErrSignatureInvalid":          errcode.ErrSignatureInvalid,
		"gmicro.ErrExpired":                   errcode.ErrExpired,
		"gmicro.ErrInvalidAuthHeader":         errcode.ErrInvalidAuthHeader,
		"gmicro.ErrMissingHeader":             errcode.ErrMissingHeader,
		"gmicro.ErrPasswordIncorrect":         errcode.ErrPasswordIncorrect,
		"gmicro.ErrPermissionDenied":          errcode.ErrPermissionDenied,
		"gmicro.ErrEncodingFailed":            errcode.ErrEncodingFailed,
		"gmicro.ErrDecodingFailed":            errcode.ErrDecodingFailed,
		"gmicro.ErrInvalidJSON":               errcode.ErrInvalidJSON,
		"gmicro.ErrEncodingJSON":              errcode.ErrEncodingJSON,
		"gmicro.ErrDecodingJSON":              errcode.ErrDecodingJSON,
		"gmicro.ErrInvalidYaml":               errcode.ErrInvalidYaml,
		"gmicro.ErrEncodingYaml":              errcode.ErrEncodingYaml,
		"gmicro.ErrDecodingYaml":              errcode.ErrDecodingYaml,
		"app.ErrConnectDB":                    bizcode.ErrConnectDB,
		"app.ErrConnectGRPC":                  bizcode.ErrConnectGRPC,
		"app.ErrUserNotFound":                 bizcode.ErrUserNotFound,
		"app.ErrUserAlreadyExists":            bizcode.ErrUserAlreadyExists,
		"app.ErrUserPasswordIncorrect":        bizcode.ErrUserPasswordIncorrect,
		"app.ErrSmsSend":                      bizcode.ErrSmsSend,
		"app.ErrCodeNotExist":                 bizcode.ErrCodeNotExist,
		"app.ErrCodeInCorrect":                bizcode.ErrCodeInCorrect,
		"app.ErrUserLoginLocked":              bizcode.ErrUserLoginLocked,
		"app.ErrSmsRateLimited":               bizcode.ErrSmsRateLimited,
		"app.ErrSmsVerifyLocked":              bizcode.ErrSmsVerifyLocked,
		"app.ErrUserAccountInactive":          bizcode.ErrUserAccountInactive,
		"app.ErrEmailVerificationUnavailable": bizcode.ErrEmailVerificationUnavailable,
		"app.ErrGoodsNotFound":                bizcode.ErrGoodsNotFound,
		"app.ErrCategoryNotFound":             bizcode.ErrCategoryNotFound,
		"app.ErrEsUnmarshal":                  bizcode.ErrEsUnmarshal,
		"app.ErrInventoryNotFound":            bizcode.ErrInventoryNotFound,
		"app.ErrInvSellDetailNotFound":        bizcode.ErrInvSellDetailNotFound,
		"app.ErrInvNotEnough":                 bizcode.ErrInvNotEnough,
		"app.ErrShopCartItemNotFound":         bizcode.ErrShopCartItemNotFound,
		"app.ErrSubmitOrder":                  bizcode.ErrSubmitOrder,
		"app.ErrNoGoodsSelect":                bizcode.ErrNoGoodsSelect,
		"app.ErrOrderNotFound":                bizcode.ErrOrderNotFound,
		"app.ErrOrderConflict":                bizcode.ErrOrderConflict,
		"app.ErrOrderStatusInvalid":           bizcode.ErrOrderStatusInvalid,
	} {
		if existing, ok := codes[code]; ok {
			t.Fatalf("error code %d is used by both %s and %s", code, existing, name)
		}
		codes[code] = name
	}
}
