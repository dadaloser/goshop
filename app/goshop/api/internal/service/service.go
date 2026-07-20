package service

import (
	"goshop/app/goshop/api/internal/data"
	"goshop/app/goshop/api/internal/loginattempt"
	vGoods "goshop/app/goshop/api/internal/service/goods/v1"
	vInventory "goshop/app/goshop/api/internal/service/inventory/v1"
	vOrder "goshop/app/goshop/api/internal/service/order/v1"
	vSms "goshop/app/goshop/api/internal/service/sms/v1"
	vUser "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/goshop/api/internal/smsattempt"
	"goshop/app/goshop/api/internal/smscode"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/options"
)

// 注意循环引用
// 使用工厂模式构建服务
type ServiceFactory interface {
	Goods() vGoods.GoodsSrv
	Inventory() vInventory.InventorySrv
	Orders() vOrder.OrderSrv
	Users() vUser.UserSrv
	Sms() vSms.SmsSrv
}

type service struct {
	data data.DataFactory

	smsOpts *options.SmsOptions

	jwtOpts *options.JwtOptions

	codeStore smscode.Store

	loginAttempts loginattempt.Store

	smsAttempts smsattempt.Store

	tokenVersions tokenversion.Store
}

func (s *service) Sms() vSms.SmsSrv {
	if s == nil {
		return vSms.NewSmsService(nil)
	}
	return vSms.NewSmsService(s.smsOpts)
}

func (s *service) Goods() vGoods.GoodsSrv {
	if s == nil {
		return vGoods.NewGoods(nil)
	}
	return vGoods.NewGoods(s.data)
}

func (s *service) Inventory() vInventory.InventorySrv {
	if s == nil {
		return vInventory.NewInventory(nil)
	}
	return vInventory.NewInventory(s.data)
}

func (s *service) Orders() vOrder.OrderSrv {
	if s == nil {
		return vOrder.NewOrderService(nil)
	}
	return vOrder.NewOrderService(s.data)
}

func (s *service) Users() vUser.UserSrv {
	if s == nil {
		return vUser.NewUserService(nil, nil, nil, nil, nil, nil)
	}
	return vUser.NewUserService(s.data, s.jwtOpts, s.codeStore, s.loginAttempts, s.smsAttempts, s.tokenVersions)
}

func NewService(store data.DataFactory, smsOpts *options.SmsOptions, jwtOpts *options.JwtOptions, codeStore smscode.Store, loginAttempts loginattempt.Store, smsAttempts smsattempt.Store, tokenVersions tokenversion.Store) *service {
	return &service{data: store,
		smsOpts:       smsOpts,
		jwtOpts:       jwtOpts,
		codeStore:     codeStore,
		loginAttempts: loginAttempts,
		smsAttempts:   smsAttempts,
		tokenVersions: tokenVersions,
	}
}

var _ ServiceFactory = &service{}
