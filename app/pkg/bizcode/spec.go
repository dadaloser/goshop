package bizcode

import "goshop/pkg/errors"

// ConnectGRPCSpec describes failures that prevent the API from reaching a
// downstream gRPC service.
var ConnectGRPCSpec = errors.Spec{
	Code:    ErrConnectGRPC,
	Kind:    errors.KindUnavailable,
	Message: "Connect to grpc error",
}

// GoodsInvalidSpec describes invalid goods API requests.
var GoodsInvalidSpec = errors.Spec{
	Code:    ErrGoodsInvalid,
	Kind:    errors.KindInvalidArgument,
	Message: "Goods request is invalid",
}

// InventoryNotFoundSpec describes a requested inventory record that does not exist.
var InventoryNotFoundSpec = errors.Spec{
	Code:    ErrInventoryNotFound,
	Kind:    errors.KindNotFound,
	Message: "Inventory not found",
}

// InventorySellDetailInvalidSpec describes an invalid inventory sell-detail request.
var InventorySellDetailInvalidSpec = errors.Spec{
	Code:    ErrInvSellDetailNotFound,
	Kind:    errors.KindInvalidArgument,
	Message: "Inventory sell detail not found",
}

// UserNotFoundSpec describes a requested user that does not exist.
var UserNotFoundSpec = errors.Spec{
	Code:    ErrUserNotFound,
	Kind:    errors.KindNotFound,
	Message: "User not found",
}

// UserAccountInactiveSpec describes a user account that is not allowed to sign in.
var UserAccountInactiveSpec = errors.Spec{
	Code:    ErrUserAccountInactive,
	Kind:    errors.KindPermissionDenied,
	Message: "User account is not active",
}

// UserPasswordIncorrectSpec describes invalid user credentials.
var UserPasswordIncorrectSpec = errors.Spec{
	Code:    ErrUserPasswordIncorrect,
	Kind:    errors.KindUnauthenticated,
	Message: "User password incorrect",
}

// LoginIdentifierInvalidSpec describes an invalid username, mobile number, or email.
var LoginIdentifierInvalidSpec = errors.Spec{
	Code:    ErrUserPasswordIncorrect,
	Kind:    errors.KindInvalidArgument,
	Message: "User password incorrect",
}

// UserLoginLockedSpec describes a temporarily rate-limited password login.
var UserLoginLockedSpec = errors.Spec{
	Code:    ErrUserLoginLocked,
	Kind:    errors.KindRateLimited,
	Message: "User login temporarily locked",
}

// SMSCodeIncorrectSpec describes an invalid SMS verification code.
var SMSCodeIncorrectSpec = errors.Spec{
	Code:    ErrCodeInCorrect,
	Kind:    errors.KindInvalidArgument,
	Message: "Sms code incorrect",
}

// PasswordConfirmationMismatchSpec describes a registration request with
// inconsistent password fields.
var PasswordConfirmationMismatchSpec = errors.Spec{
	Code:    ErrPasswordConfirmationMismatch,
	Kind:    errors.KindInvalidArgument,
	Message: "Password confirmation does not match",
}

// CaptchaVerificationFailedSpec describes an invalid or expired captcha.
var CaptchaVerificationFailedSpec = errors.Spec{
	Code:    ErrCaptchaVerificationFailed,
	Kind:    errors.KindInvalidArgument,
	Message: "Captcha verification failed",
}

// DeviceSessionUnavailableSpec describes a temporary failure retrieving device
// login records without exposing the backing storage or service details.
var DeviceSessionUnavailableSpec = errors.Spec{
	Code:    ErrDeviceSessionUnavailable,
	Kind:    errors.KindUnavailable,
	Message: "Device login records are temporarily unavailable",
}

// SMSCodeNotExistSpec describes a missing or expired SMS verification code.
var SMSCodeNotExistSpec = errors.Spec{
	Code:    ErrCodeNotExist,
	Kind:    errors.KindInvalidArgument,
	Message: "Sms code incorrect or expired",
}

// SMSVerifyLockedSpec describes a temporarily rate-limited SMS verification.
var SMSVerifyLockedSpec = errors.Spec{
	Code:    ErrSmsVerifyLocked,
	Kind:    errors.KindRateLimited,
	Message: "Sms verification temporarily locked",
}

// EmailVerificationUnavailableSpec describes an unavailable email verification dependency.
var EmailVerificationUnavailableSpec = errors.Spec{
	Code:    ErrEmailVerificationUnavailable,
	Kind:    errors.KindUnavailable,
	Message: "Email verification temporarily unavailable",
}
