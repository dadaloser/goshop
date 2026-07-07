package user

import (
	"goshop/app/goshop/api/internal/service"
	"goshop/app/goshop/api/internal/tokenrevocation"

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
