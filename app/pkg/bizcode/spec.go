package bizcode

// ConnectGRPCSpec describes failures that prevent the API from reaching a
// downstream gRPC service.
var ConnectGRPCSpec = codeSpec(ErrConnectGRPC)

// GoodsInvalidSpec describes invalid goods API requests.
var GoodsInvalidSpec = codeSpec(ErrGoodsInvalid)

// InventoryNotFoundSpec describes a requested inventory record that does not exist.
var InventoryNotFoundSpec = codeSpec(ErrInventoryNotFound)

// InventorySellDetailInvalidSpec describes an invalid inventory sell-detail request.
var InventorySellDetailInvalidSpec = codeSpec(ErrInvSellDetailNotFound)

// UserNotFoundSpec describes a requested user that does not exist.
var UserNotFoundSpec = codeSpec(ErrUserNotFound)

// UserAccountInactiveSpec describes a user account that is not allowed to sign in.
var UserAccountInactiveSpec = codeSpec(ErrUserAccountInactive)

// UserPasswordIncorrectSpec describes invalid user credentials.
var UserPasswordIncorrectSpec = codeSpec(ErrUserPasswordIncorrect)

// LoginIdentifierInvalidSpec describes an invalid username, mobile number, or email.
var LoginIdentifierInvalidSpec = codeSpec(ErrUserPasswordIncorrect)

// UserLoginLockedSpec describes a temporarily rate-limited password login.
var UserLoginLockedSpec = codeSpec(ErrUserLoginLocked)

// SMSCodeIncorrectSpec describes an invalid SMS verification code.
var SMSCodeIncorrectSpec = codeSpec(ErrCodeInCorrect)

// PasswordConfirmationMismatchSpec describes a registration request with
// inconsistent password fields.
var PasswordConfirmationMismatchSpec = codeSpec(ErrPasswordConfirmationMismatch)

// CaptchaVerificationFailedSpec describes an invalid or expired captcha.
var CaptchaVerificationFailedSpec = codeSpec(ErrCaptchaVerificationFailed)

// DeviceSessionUnavailableSpec describes a temporary failure retrieving device
// login records without exposing the backing storage or service details.
var DeviceSessionUnavailableSpec = codeSpec(ErrDeviceSessionUnavailable)

// SMSCodeNotExistSpec describes a missing or expired SMS verification code.
var SMSCodeNotExistSpec = codeSpec(ErrCodeNotExist)

// SMSVerifyLockedSpec describes a temporarily rate-limited SMS verification.
var SMSVerifyLockedSpec = codeSpec(ErrSmsVerifyLocked)

// EmailVerificationUnavailableSpec describes an unavailable email verification dependency.
var EmailVerificationUnavailableSpec = codeSpec(ErrEmailVerificationUnavailable)
