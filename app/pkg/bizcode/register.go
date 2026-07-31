package bizcode

import "goshop/pkg/errors"

// Catalog contains the domain-level public error contracts. Registration is
// explicit: applications must call RegisterAll during startup.
var Catalog = errors.Catalog{
	{Code: ErrConnectDB, Kind: errors.KindUnavailable, Message: "数据库服务暂不可用"},
	{Code: ErrConnectGRPC, Kind: errors.KindUnavailable, Message: "远程服务暂不可用"},
	{Code: ErrUserNotFound, Kind: errors.KindNotFound, Message: "用户不存在"},
	{Code: ErrUserAlreadyExists, Kind: errors.KindConflict, Message: "用户已存在"},
	{Code: ErrUserPasswordIncorrect, Kind: errors.KindUnauthenticated, Message: "用户名或密码错误"},
	{Code: ErrSmsSend, Kind: errors.KindUnavailable, Message: "短信发送失败"},
	{Code: ErrCodeNotExist, Kind: errors.KindInvalidArgument, Message: "短信验证码错误或已过期"},
	{Code: ErrCodeInCorrect, Kind: errors.KindInvalidArgument, Message: "短信验证码错误"},
	{Code: ErrPasswordConfirmationMismatch, Kind: errors.KindInvalidArgument, Message: "两次输入的密码不一致"},
	{Code: ErrCaptchaVerificationFailed, Kind: errors.KindInvalidArgument, Message: "图形验证码错误或已过期"},
	{Code: ErrDeviceSessionUnavailable, Kind: errors.KindUnavailable, Message: "设备获取失败，请稍后重试"},
	{Code: ErrUserLoginLocked, Kind: errors.KindRateLimited, Message: "用户登录已被暂时锁定"},
	{Code: ErrSmsRateLimited, Kind: errors.KindRateLimited, Message: "短信发送过于频繁，请稍后再试"},
	{Code: ErrSmsVerifyLocked, Kind: errors.KindRateLimited, Message: "短信验证码校验已被暂时锁定"},
	{Code: ErrUserAccountInactive, Kind: errors.KindPermissionDenied, Message: "用户账号未启用"},
	{Code: ErrAccountDeletionBlocked, Kind: errors.KindConflict, Message: "账号存在未完成的订单、退款或售后事项，暂时无法注销"},
	{Code: ErrEmailVerificationUnavailable, Kind: errors.KindUnavailable, Message: "邮箱验证服务暂不可用"},
	{Code: ErrUserMobileAlreadyExists, Kind: errors.KindConflict, Message: "手机号已存在"},
	{Code: ErrUserEmailAlreadyExists, Kind: errors.KindConflict, Message: "邮箱已存在"},
	{Code: ErrUsernameAlreadyExists, Kind: errors.KindConflict, Message: "用户名已存在"},
	{Code: ErrGoodsNotFound, Kind: errors.KindNotFound, Message: "商品不存在"},
	{Code: ErrGoodsInvalid, Kind: errors.KindInvalidArgument, Message: "商品请求参数无效"},
	{Code: ErrCategoryNotFound, Kind: errors.KindNotFound, Message: "商品分类不存在"},
	{Code: ErrBrandNotFound, Kind: errors.KindNotFound, Message: "品牌不存在"},
	{Code: ErrBannerNotFound, Kind: errors.KindNotFound, Message: "轮播图不存在"},
	{Code: ErrCategoryBrandNotFound, Kind: errors.KindNotFound, Message: "商品分类与品牌关系不存在"},
	{Code: ErrEsUnmarshal, Kind: errors.KindInternal, Message: "搜索服务数据解析失败"},
	{Code: ErrInventoryNotFound, Kind: errors.KindNotFound, Message: "库存不存在"},
	{Code: ErrInvSellDetailNotFound, Kind: errors.KindInvalidArgument, Message: "库存扣减明细不存在"},
	{Code: ErrInvNotEnough, Kind: errors.KindConflict, Message: "库存不足"},
	{Code: ErrShopCartItemNotFound, Kind: errors.KindNotFound, Message: "购物车商品不存在"},
	{Code: ErrSubmitOrder, Kind: errors.KindConflict, Message: "提交订单失败"},
	{Code: ErrNoGoodsSelect, Kind: errors.KindInvalidArgument, Message: "未选择商品"},
	{Code: ErrOrderNotFound, Kind: errors.KindNotFound, Message: "订单不存在"},
	{Code: ErrOrderConflict, Kind: errors.KindConflict, Message: "订单数据冲突"},
	{Code: ErrOrderStatusInvalid, Kind: errors.KindConflict, Message: "订单状态无效"},
}

// RegisterAll explicitly adds every domain error contract to the shared errors
// catalog. It is safe to call more than once.
func RegisterAll() { Catalog.RegisterAll() }
