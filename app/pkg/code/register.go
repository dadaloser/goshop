package code

import "net/http"

func init() {
	register(ErrConnectDB, http.StatusInternalServerError, "Init db error")
	register(ErrConnectGRPC, http.StatusInternalServerError, "Connect to grpc error")
}
