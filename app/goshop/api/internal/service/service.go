package service

import (
	"goshop/app/goshop/api/internal/data"
	vGoods "goshop/app/goshop/api/internal/service/goods/v1"
	vSms "goshop/app/goshop/api/internal/service/sms/v1"
	vUser "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/goshop/api/internal/smscode"
	"goshop/app/pkg/options"
)

// 注意循环引用
// 使用工厂模式构建服务
type ServiceFactory interface {
	Goods() vGoods.GoodsSrv
	Users() vUser.UserSrv
	Sms() vSms.SmsSrv
}

type service struct {
	data data.DataFactory

	smsOpts *options.SmsOptions

	jwtOpts *options.JwtOptions

	codeStore smscode.Store
}

func (s *service) Sms() vSms.SmsSrv {
	return vSms.NewSmsService(s.smsOpts)
}

func (s *service) Goods() vGoods.GoodsSrv {
	return vGoods.NewGoods(s.data)
}

func (s *service) Users() vUser.UserSrv {
	return vUser.NewUserService(s.data, s.jwtOpts, s.codeStore)
}

func NewService(store data.DataFactory, smsOpts *options.SmsOptions, jwtOpts *options.JwtOptions, codeStore smscode.Store) *service {
	return &service{data: store,
		smsOpts:   smsOpts,
		jwtOpts:   jwtOpts,
		codeStore: codeStore,
	}
}

var _ ServiceFactory = &service{}
