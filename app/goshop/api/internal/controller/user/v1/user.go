package user

import (
	"goshop/app/goshop/api/internal/service"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/authsession/tokenrevocation"
	"goshop/app/pkg/bizcode"
	"goshop/pkg/errors"

	ut "github.com/go-playground/universal-translator"
)

type userServer struct {
	trans ut.Translator

	sf                                 service.ServiceFactory
	revokedTokens                      tokenrevocation.Store
	smsRegistrationVerificationEnabled bool
}

type ControllerOption func(*userServer)

// WithSMSRegistrationVerification requires SMS codes rather than image
// captchas for registration.
func WithSMSRegistrationVerification(enabled bool) ControllerOption {
	return func(server *userServer) {
		server.smsRegistrationVerificationEnabled = enabled
	}
}

func NewUserController(trans ut.Translator, sf service.ServiceFactory, revokedTokens tokenrevocation.Store, opts ...ControllerOption) *userServer {
	server := &userServer{trans: trans, sf: sf, revokedTokens: revokedTokens}
	for _, opt := range opts {
		if opt != nil {
			opt(server)
		}
	}
	return server
}

func (us *userServer) usersService() (userv1.UserSrv, error) {
	if us == nil || us.sf == nil {
		return nil, errors.NewCode(bizcode.ErrConnectGRPC, "user service is not initialized")
	}
	userSrv := us.sf.Users()
	if userSrv == nil {
		return nil, errors.NewCode(bizcode.ErrConnectGRPC, "user service is not initialized")
	}
	return userSrv, nil
}
