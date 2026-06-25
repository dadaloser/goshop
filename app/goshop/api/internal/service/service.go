package service

import (
	"goshop/app/goshop/api/internal/data"
	v1 "goshop/app/goshop/api/internal/service/goods/v1"
	v12 "goshop/app/goshop/api/internal/service/sms/v1"
	v13 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/options"
)

// 注意循环引用
// 使用工厂模式构建服务
type ServiceFactory interface {
	Goods() v1.GoodsSrv
	Users() v13.UserSrv
	Sms() v12.SmsSrv
}

type service struct {
	data data.DataFactory

	smsOpts *options.SmsOptions

	jwtOpts *options.JwtOptions
}

func (s *service) Sms() v12.SmsSrv {
	return v12.NewSmsService(s.smsOpts)
}

func (s *service) Goods() v1.GoodsSrv {
	return v1.NewGoods(s.data)
}

func (s *service) Users() v13.UserSrv {
	return v13.NewUserService(s.data, s.jwtOpts)
}

func NewService(store data.DataFactory, smsOpts *options.SmsOptions, jwtOpts *options.JwtOptions) *service {
	return &service{data: store,
		smsOpts: smsOpts,
		jwtOpts: jwtOpts,
	}
}

var _ ServiceFactory = &service{}
