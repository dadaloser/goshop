package code

const (
	// ErrUserNotFound - 404: User not found.
	ErrUserNotFound int = iota + 100401

	// ErrUserAlreadyExists - 400: User already exists.
	ErrUserAlreadyExists

	// ErrUserPasswordIncorrect - 400: User password incorrect.
	ErrUserPasswordIncorrect

	// ErrSmsSend - 400: Send sms error.
	ErrSmsSend

	// ErrCodeNotExist - 400: Sms code incorrect or expired.
	ErrCodeNotExist

	// ErrCodeInCorrect - 400: Sms code incorrect.
	ErrCodeInCorrect

	// ErrUserLoginLocked - 403: User login is temporarily locked.
	ErrUserLoginLocked

	// ErrSmsRateLimited - 403: Sms send is temporarily rate limited.
	ErrSmsRateLimited

	// ErrSmsVerifyLocked - 403: Sms verification is temporarily locked.
	ErrSmsVerifyLocked

	// ErrUserAccountInactive - 403: User account is not active.
	ErrUserAccountInactive

	// ErrAccountDeletionBlocked - 409: Account has unfinished business.
	ErrAccountDeletionBlocked
)
