package bizcode

import "goshop/pkg/errors"

// Catalog contains the domain-level public error contracts. Registration is
// explicit: applications must call RegisterAll during startup.
var Catalog = errors.Catalog{
	{Code: ErrConnectDB, Kind: errors.KindUnavailable, Message: "Init db error"},
	{Code: ErrConnectGRPC, Kind: errors.KindUnavailable, Message: "Connect to grpc error"},
	{Code: ErrUserNotFound, Kind: errors.KindNotFound, Message: "User not found"},
	{Code: ErrUserAlreadyExists, Kind: errors.KindConflict, Message: "User already exists"},
	{Code: ErrUserPasswordIncorrect, Kind: errors.KindUnauthenticated, Message: "User password incorrect"},
	{Code: ErrSmsSend, Kind: errors.KindUnavailable, Message: "Send sms error"},
	{Code: ErrCodeNotExist, Kind: errors.KindInvalidArgument, Message: "Sms code incorrect or expired"},
	{Code: ErrCodeInCorrect, Kind: errors.KindInvalidArgument, Message: "Sms code incorrect"},
	{Code: ErrUserLoginLocked, Kind: errors.KindRateLimited, Message: "User login temporarily locked"},
	{Code: ErrSmsRateLimited, Kind: errors.KindRateLimited, Message: "Sms send temporarily rate limited"},
	{Code: ErrSmsVerifyLocked, Kind: errors.KindRateLimited, Message: "Sms verification temporarily locked"},
	{Code: ErrUserAccountInactive, Kind: errors.KindPermissionDenied, Message: "User account is not active"},
	{Code: ErrAccountDeletionBlocked, Kind: errors.KindConflict, Message: "Account has unfinished orders, refunds, or after-sales cases"},
	{Code: ErrEmailVerificationUnavailable, Kind: errors.KindUnavailable, Message: "Email verification temporarily unavailable"},
	{Code: ErrGoodsNotFound, Kind: errors.KindNotFound, Message: "Goods not found"},
	{Code: ErrGoodsInvalid, Kind: errors.KindInvalidArgument, Message: "Goods request is invalid"},
	{Code: ErrCategoryNotFound, Kind: errors.KindNotFound, Message: "Category not found"},
	{Code: ErrBrandNotFound, Kind: errors.KindNotFound, Message: "Brand not found"},
	{Code: ErrBannerNotFound, Kind: errors.KindNotFound, Message: "Banner not found"},
	{Code: ErrCategoryBrandNotFound, Kind: errors.KindNotFound, Message: "Category brand relation not found"},
	{Code: ErrEsUnmarshal, Kind: errors.KindInternal, Message: "Es unmarshal error"},
	{Code: ErrInventoryNotFound, Kind: errors.KindNotFound, Message: "Inventory not found"},
	{Code: ErrInvSellDetailNotFound, Kind: errors.KindInvalidArgument, Message: "Inventory sell detail not found"},
	{Code: ErrInvNotEnough, Kind: errors.KindConflict, Message: "Inventory not enough"},
	{Code: ErrShopCartItemNotFound, Kind: errors.KindNotFound, Message: "ShopCart item not found"},
	{Code: ErrSubmitOrder, Kind: errors.KindConflict, Message: "Submit order error"},
	{Code: ErrNoGoodsSelect, Kind: errors.KindInvalidArgument, Message: "No Goods selected"},
	{Code: ErrOrderNotFound, Kind: errors.KindNotFound, Message: "Order not found"},
	{Code: ErrOrderConflict, Kind: errors.KindConflict, Message: "Order already exists with different data"},
	{Code: ErrOrderStatusInvalid, Kind: errors.KindConflict, Message: "Order status is invalid"},
}

// RegisterAll explicitly adds every domain error contract to the shared errors
// catalog. It is safe to call more than once.
func RegisterAll() { Catalog.RegisterAll() }
