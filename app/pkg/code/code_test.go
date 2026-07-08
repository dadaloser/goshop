package code_test

import (
	"testing"

	appcode "goshop/app/pkg/code"
	gmicrocode "goshop/gmicro/code"
)

func TestErrorCodesAreUnique(t *testing.T) {
	codes := map[int]string{}
	for name, code := range map[string]int{
		"gmicro.ErrSuccess":            gmicrocode.ErrSuccess,
		"gmicro.ErrUnknown":            gmicrocode.ErrUnknown,
		"gmicro.ErrBind":               gmicrocode.ErrBind,
		"gmicro.ErrValidation":         gmicrocode.ErrValidation,
		"gmicro.ErrTokenInvalid":       gmicrocode.ErrTokenInvalid,
		"gmicro.ErrPageNotFound":       gmicrocode.ErrPageNotFound,
		"gmicro.ErrDatabase":           gmicrocode.ErrDatabase,
		"gmicro.ErrEncrypt":            gmicrocode.ErrEncrypt,
		"gmicro.ErrSignatureInvalid":   gmicrocode.ErrSignatureInvalid,
		"gmicro.ErrExpired":            gmicrocode.ErrExpired,
		"gmicro.ErrInvalidAuthHeader":  gmicrocode.ErrInvalidAuthHeader,
		"gmicro.ErrMissingHeader":      gmicrocode.ErrMissingHeader,
		"gmicro.ErrPasswordIncorrect":  gmicrocode.ErrPasswordIncorrect,
		"gmicro.ErrPermissionDenied":   gmicrocode.ErrPermissionDenied,
		"gmicro.ErrEncodingFailed":     gmicrocode.ErrEncodingFailed,
		"gmicro.ErrDecodingFailed":     gmicrocode.ErrDecodingFailed,
		"gmicro.ErrInvalidJSON":        gmicrocode.ErrInvalidJSON,
		"gmicro.ErrEncodingJSON":       gmicrocode.ErrEncodingJSON,
		"gmicro.ErrDecodingJSON":       gmicrocode.ErrDecodingJSON,
		"gmicro.ErrInvalidYaml":        gmicrocode.ErrInvalidYaml,
		"gmicro.ErrEncodingYaml":       gmicrocode.ErrEncodingYaml,
		"gmicro.ErrDecodingYaml":       gmicrocode.ErrDecodingYaml,
		"app.ErrConnectDB":             appcode.ErrConnectDB,
		"app.ErrConnectGRPC":           appcode.ErrConnectGRPC,
		"app.ErrUserNotFound":          appcode.ErrUserNotFound,
		"app.ErrUserAlreadyExists":     appcode.ErrUserAlreadyExists,
		"app.ErrUserPasswordIncorrect": appcode.ErrUserPasswordIncorrect,
		"app.ErrSmsSend":               appcode.ErrSmsSend,
		"app.ErrCodeNotExist":          appcode.ErrCodeNotExist,
		"app.ErrCodeInCorrect":         appcode.ErrCodeInCorrect,
		"app.ErrUserLoginLocked":       appcode.ErrUserLoginLocked,
		"app.ErrSmsRateLimited":        appcode.ErrSmsRateLimited,
		"app.ErrSmsVerifyLocked":       appcode.ErrSmsVerifyLocked,
		"app.ErrGoodsNotFound":         appcode.ErrGoodsNotFound,
		"app.ErrCategoryNotFound":      appcode.ErrCategoryNotFound,
		"app.ErrEsUnmarshal":           appcode.ErrEsUnmarshal,
		"app.ErrInventoryNotFound":     appcode.ErrInventoryNotFound,
		"app.ErrInvSellDetailNotFound": appcode.ErrInvSellDetailNotFound,
		"app.ErrInvNotEnough":          appcode.ErrInvNotEnough,
		"app.ErrShopCartItemNotFound":  appcode.ErrShopCartItemNotFound,
		"app.ErrSubmitOrder":           appcode.ErrSubmitOrder,
		"app.ErrNoGoodsSelect":         appcode.ErrNoGoodsSelect,
		"app.ErrOrderNotFound":         appcode.ErrOrderNotFound,
		"app.ErrOrderConflict":         appcode.ErrOrderConflict,
		"app.ErrOrderStatusInvalid":    appcode.ErrOrderStatusInvalid,
	} {
		if existing, ok := codes[code]; ok {
			t.Fatalf("error code %d is used by both %s and %s", code, existing, name)
		}
		codes[code] = name
	}
}
