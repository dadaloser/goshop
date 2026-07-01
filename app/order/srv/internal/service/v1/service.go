package service

import (
	"goshop/app/order/srv/internal/boundary"
	v1 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/pkg/options"
)

type ServiceFactory interface {
	Orders() OrderSrv
}

type service struct {
	data     v1.DataFactory
	dtmopts  *options.DtmOptions
	upstream upstream
}

type upstream struct {
	goods boundary.GoodsGateway
}

func (s *service) Orders() OrderSrv {
	return newOrderService(s)
}

var _ ServiceFactory = &service{}

func NewService(data v1.DataFactory, dtmopts *options.DtmOptions, goods boundary.GoodsGateway) *service {
	return &service{
		data:     data,
		dtmopts:  dtmopts,
		upstream: upstream{goods: goods},
	}
}
