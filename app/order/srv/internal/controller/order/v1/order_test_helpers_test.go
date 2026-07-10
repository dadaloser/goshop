package order

import (
	"context"
	"time"

	"goshop/app/order/srv/internal/domain/dto"
	"goshop/app/order/srv/internal/service/v1"
	gorm2 "goshop/app/pkg/gorm"
	metav1 "goshop/pkg/common/meta/v1"
)

type fakeOrderServiceFactory struct {
	orders fakeOrderSrv
}

func (f fakeOrderServiceFactory) Orders() service.OrderSrv {
	return f.orders
}

func (fakeOrderServiceFactory) RunBackground(context.Context) error {
	return nil
}

type fakeOrderSrv struct {
	statusLogs func(context.Context, uint64, string) (*dto.OrderStatusLogDTOList, error)
}

func (f fakeOrderSrv) CartItemList(context.Context, uint64, metav1.ListMeta, []string) (*dto.ShopCartDTOList, error) {
	return nil, nil
}

func (f fakeOrderSrv) CreateCartItem(context.Context, *dto.ShopCartDTO) (*dto.ShopCartDTO, error) {
	return nil, nil
}

func (f fakeOrderSrv) UpdateCartItem(context.Context, *dto.ShopCartDTO) error {
	return nil
}

func (f fakeOrderSrv) DeleteCartItem(context.Context, uint64, uint64) error {
	return nil
}

func (f fakeOrderSrv) Get(context.Context, uint64, string) (*dto.OrderDTO, error) {
	return nil, nil
}

func (f fakeOrderSrv) StatusLogs(ctx context.Context, userID uint64, orderSn string) (*dto.OrderStatusLogDTOList, error) {
	if f.statusLogs != nil {
		return f.statusLogs(ctx, userID, orderSn)
	}
	return &dto.OrderStatusLogDTOList{}, nil
}

func (f fakeOrderSrv) List(context.Context, uint64, metav1.ListMeta, []string) (*dto.OrderDTOList, error) {
	return nil, nil
}

func (f fakeOrderSrv) Submit(context.Context, *dto.OrderDTO) error {
	return nil
}

func (f fakeOrderSrv) Create(context.Context, *dto.OrderDTO) error {
	return nil
}

func (f fakeOrderSrv) CreateCom(context.Context, *dto.OrderDTO) error {
	return nil
}

func (f fakeOrderSrv) Update(context.Context, *dto.OrderDTO) error {
	return nil
}

func doBaseModel(id int32, createdAt time.Time) gorm2.BaseModel {
	return gorm2.BaseModel{
		ID:        id,
		CreatedAt: createdAt,
	}
}

var _ service.ServiceFactory = fakeOrderServiceFactory{}
var _ service.OrderSrv = fakeOrderSrv{}
var _ = gorm2.BaseModel{}
