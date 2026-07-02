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
}
