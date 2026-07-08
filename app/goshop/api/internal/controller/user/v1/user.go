package user

import (
	"goshop/app/goshop/api/internal/service"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/goshop/api/internal/tokenrevocation"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"

	ut "github.com/go-playground/universal-translator"
)

type userServer struct {
	trans ut.Translator

	sf            service.ServiceFactory
	revokedTokens tokenrevocation.Store
}

func NewUserController(trans ut.Translator, sf service.ServiceFactory, revokedTokens tokenrevocation.Store) *userServer {
	return &userServer{trans: trans, sf: sf, revokedTokens: revokedTokens}
}

func (us *userServer) usersService() (userv1.UserSrv, error) {
	if us == nil || us.sf == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "user service is not initialized")
	}
	userSrv := us.sf.Users()
	if userSrv == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "user service is not initialized")
	}
	return userSrv, nil
}
