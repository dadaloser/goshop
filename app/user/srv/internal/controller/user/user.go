package user

import (
	v1 "goshop/api/user/v1"
	srv1 "goshop/app/user/srv/internal/service/v1"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewUserServer)

type userServer struct {
	v1.UnimplementedUserServer
	srv srv1.UserSrv
}

//func (us *userServer) mustEmbedUnimplementedUserServer() {
//	//TODO implement me
//	panic("implement me")
//}

// java中的ioc，控制翻转 ioc = injection of control
// 代码分层，第三方服务， rpc， redis， 等等， 带来一定的复杂度
func NewUserServer(srv srv1.UserSrv) v1.UserServer {
	return &userServer{srv: srv}
}

var _ v1.UserServer = &userServer{}
