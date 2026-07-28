package bizcode

import (
	"net/http"

	"goshop/pkg/errors"
)

func init() {
	register(ErrConnectDB, http.StatusServiceUnavailable, errors.KindUnavailable, "Init db error")
	register(ErrConnectGRPC, http.StatusServiceUnavailable, errors.KindUnavailable, "Connect to grpc error")
	register(ErrUserNotFound, http.StatusNotFound, errors.KindNotFound, "User not found")
	register(ErrUserAlreadyExists, http.StatusConflict, errors.KindConflict, "User already exists")
	register(ErrUserPasswordIncorrect, http.StatusUnauthorized, errors.KindUnauthenticated, "User password incorrect")
	register(ErrSmsSend, http.StatusServiceUnavailable, errors.KindUnavailable, "Send sms error")
	register(ErrCodeNotExist, http.StatusBadRequest, errors.KindInvalidArgument, "Sms code incorrect or expired")
	register(ErrCodeInCorrect, http.StatusBadRequest, errors.KindInvalidArgument, "Sms code incorrect")
	register(ErrUserLoginLocked, http.StatusTooManyRequests, errors.KindRateLimited, "User login temporarily locked")
	register(ErrSmsRateLimited, http.StatusTooManyRequests, errors.KindRateLimited, "Sms send temporarily rate limited")
	register(ErrSmsVerifyLocked, http.StatusTooManyRequests, errors.KindRateLimited, "Sms verification temporarily locked")
	register(ErrUserAccountInactive, http.StatusForbidden, errors.KindPermissionDenied, "User account is not active")
	register(ErrAccountDeletionBlocked, http.StatusConflict, errors.KindConflict, "Account has unfinished orders, refunds, or after-sales cases")
	register(ErrEmailVerificationUnavailable, http.StatusServiceUnavailable, errors.KindUnavailable, "Email verification temporarily unavailable")
	register(ErrGoodsNotFound, http.StatusNotFound, errors.KindNotFound, "Goods not found")
	register(ErrGoodsInvalid, http.StatusBadRequest, errors.KindInvalidArgument, "Goods request is invalid")
	register(ErrCategoryNotFound, http.StatusNotFound, errors.KindNotFound, "Category not found")
	register(ErrBrandNotFound, http.StatusNotFound, errors.KindNotFound, "Brand not found")
	register(ErrBannerNotFound, http.StatusNotFound, errors.KindNotFound, "Banner not found")
	register(ErrCategoryBrandNotFound, http.StatusNotFound, errors.KindNotFound, "Category brand relation not found")
	register(ErrEsUnmarshal, http.StatusInternalServerError, errors.KindInternal, "Es unmarshal error")
	register(ErrInventoryNotFound, http.StatusNotFound, errors.KindNotFound, "Inventory not found")
	register(ErrInvSellDetailNotFound, http.StatusBadRequest, errors.KindInvalidArgument, "Inventory sell detail not found")
	register(ErrInvNotEnough, http.StatusConflict, errors.KindConflict, "Inventory not enough")
	register(ErrShopCartItemNotFound, http.StatusNotFound, errors.KindNotFound, "ShopCart item not found")
	register(ErrSubmitOrder, http.StatusConflict, errors.KindConflict, "Submit order error")
	register(ErrNoGoodsSelect, http.StatusBadRequest, errors.KindInvalidArgument, "No Goods selected")
	register(ErrOrderNotFound, http.StatusNotFound, errors.KindNotFound, "Order not found")
	register(ErrOrderConflict, http.StatusConflict, errors.KindConflict, "Order already exists with different data")
	register(ErrOrderStatusInvalid, http.StatusConflict, errors.KindConflict, "Order status is invalid")
}
