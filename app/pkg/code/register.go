package code

import "net/http"

func init() {
	register(ErrConnectDB, http.StatusInternalServerError, "Init db error")
	register(ErrConnectGRPC, http.StatusInternalServerError, "Connect to grpc error")
	register(ErrUserNotFound, http.StatusNotFound, "User not found")
	register(ErrUserAlreadyExists, http.StatusBadRequest, "User already exists")
	register(ErrUserPasswordIncorrect, http.StatusBadRequest, "User password incorrect")
	register(ErrSmsSend, http.StatusBadRequest, "Send sms error")
	register(ErrCodeNotExist, http.StatusBadRequest, "Sms code incorrect or expired")
	register(ErrCodeInCorrect, http.StatusBadRequest, "Sms code incorrect")
	register(ErrUserLoginLocked, http.StatusForbidden, "User login temporarily locked")
	register(ErrSmsRateLimited, http.StatusForbidden, "Sms send temporarily rate limited")
	register(ErrSmsVerifyLocked, http.StatusForbidden, "Sms verification temporarily locked")
	register(ErrGoodsNotFound, http.StatusNotFound, "Goods not found")
	register(ErrCategoryNotFound, http.StatusNotFound, "Category not found")
	register(ErrEsUnmarshal, http.StatusInternalServerError, "Es unmarshal error")
	register(ErrInventoryNotFound, http.StatusNotFound, "Inventory not found")
	register(ErrInvSellDetailNotFound, http.StatusBadRequest, "Inventory sell detail not found")
	register(ErrInvNotEnough, http.StatusBadRequest, "Inventory not enough")
	register(ErrShopCartItemNotFound, http.StatusNotFound, "ShopCart item not found")
	register(ErrSubmitOrder, http.StatusBadRequest, "Submit order error")
	register(ErrNoGoodsSelect, http.StatusBadRequest, "No Goods selected")
	register(ErrOrderNotFound, http.StatusNotFound, "Order not found")
	register(ErrOrderConflict, http.StatusBadRequest, "Order already exists with different data")
}
