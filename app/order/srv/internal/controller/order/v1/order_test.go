package order

import (
	"context"
	"testing"
	"time"

	pb "goshop/api/order/v1"
	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/order/srv/internal/domain/dto"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestOrderServerRejectsNilRequests(t *testing.T) {
	server := &orderServer{}

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "cart item list",
			run: func() error {
				_, err := server.CartItemList(context.Background(), nil)
				return err
			},
		},
		{
			name: "create cart item",
			run: func() error {
				_, err := server.CreateCartItem(context.Background(), nil)
				return err
			},
		},
		{
			name: "update cart item",
			run: func() error {
				_, err := server.UpdateCartItem(context.Background(), nil)
				return err
			},
		},
		{
			name: "delete cart item",
			run: func() error {
				_, err := server.DeleteCartItem(context.Background(), nil)
				return err
			},
		},
		{
			name: "create order",
			run: func() error {
				_, err := server.CreateOrder(context.Background(), nil)
				return err
			},
		},
		{
			name: "create order compensation",
			run: func() error {
				_, err := server.CreateOrderCom(context.Background(), nil)
				return err
			},
		},
		{
			name: "submit order",
			run: func() error {
				_, err := server.SubmitOrder(context.Background(), nil)
				return err
			},
		},
		{
			name: "order list",
			run: func() error {
				_, err := server.OrderList(context.Background(), nil)
				return err
			},
		},
		{
			name: "order detail",
			run: func() error {
				_, err := server.OrderDetail(context.Background(), nil)
				return err
			},
		},
		{
			name: "order status logs",
			run: func() error {
				_, err := server.OrderStatusLogs(context.Background(), nil)
				return err
			},
		},
		{
			name: "update order status",
			run: func() error {
				_, err := server.UpdateOrderStatus(context.Background(), nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, code2.ErrValidation) {
				t.Fatalf("error = %v, want code %d", err, code2.ErrValidation)
			}
		})
	}
}

func TestOrderServerRejectsNilOrderItem(t *testing.T) {
	server := &orderServer{}

	_, err := server.CreateOrder(context.Background(), &pb.OrderRequest{
		OrderItems: []*pb.OrderItemResponse{nil},
	})
	if !errors.IsCode(err, code2.ErrValidation) {
		t.Fatalf("error = %v, want code %d", err, code2.ErrValidation)
	}
}

func TestOrderServerReturnsOrderStatusLogs(t *testing.T) {
	now := time.Unix(1700000000, 0)
	server := NewOrderServer(fakeOrderServiceFactory{
		orders: fakeOrderSrv{
			statusLogs: func(context.Context, uint64, string) (*dto.OrderStatusLogDTOList, error) {
				return &dto.OrderStatusLogDTOList{
					TotalCount: 1,
					Items: []*dto.OrderStatusLogDTO{
						{
							OrderStatusLogDO: do.OrderStatusLogDO{
								BaseModel:  doBaseModel(1, now),
								OrderID:    2,
								OrderSn:    "order-1",
								FromStatus: "WAIT_BUYER_PAY",
								ToStatus:   "TRADE_SUCCESS",
								Reason:     "payment callback",
								Source:     "order.pay_callback",
								Operator:   "system",
							},
						},
					},
				}, nil
			},
		},
	})

	resp, err := server.OrderStatusLogs(context.Background(), &pb.OrderRequest{
		UserId:  9,
		OrderSn: "order-1",
	})
	if err != nil {
		t.Fatalf("OrderStatusLogs() error = %v", err)
	}
	if resp.GetTotal() != 1 || len(resp.GetData()) != 1 {
		t.Fatalf("OrderStatusLogs() response = %+v", resp)
	}
	if got := resp.GetData()[0]; got.GetOrderSn() != "order-1" || got.GetToStatus() != "TRADE_SUCCESS" || got.GetAddTime() == "" {
		t.Fatalf("OrderStatusLogs() item = %+v", got)
	}
}
