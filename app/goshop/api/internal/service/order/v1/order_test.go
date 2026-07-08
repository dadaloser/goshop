package v1

import (
	"context"
	"testing"

	gpb "goshop/api/goods/v1"
	opb "goshop/api/order/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestSimulatePayCallbackRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		svc  OrderSrv
		req  *PayCallbackRequest
		code int
	}{
		{
			name: "missing data",
			svc:  NewOrderService(nil),
			req:  &PayCallbackRequest{UserID: 1, OrderSn: "order-1"},
			code: code.ErrConnectGRPC,
		},
		{
			name: "nil request",
			svc:  NewOrderService(fakeDataFactory{}),
			code: code.ErrOrderStatusInvalid,
		},
		{
			name: "missing user",
			svc:  NewOrderService(fakeDataFactory{}),
			req:  &PayCallbackRequest{OrderSn: "order-1"},
			code: code.ErrOrderStatusInvalid,
		},
		{
			name: "missing trade no for success",
			svc:  NewOrderService(fakeDataFactory{orderClient: fakeOrderClient{}}),
			req:  &PayCallbackRequest{UserID: 1, OrderSn: "order-1", Success: true},
			code: code.ErrOrderStatusInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.svc.SimulatePayCallback(context.Background(), tt.req)
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %d", err, tt.code)
			}
		})
	}
}

func TestSimulatePayCallbackWritesExpectedStatus(t *testing.T) {
	var got *opb.OrderStatus
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			update: func(_ context.Context, in *opb.OrderStatus, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				got = in
				return &emptypb.Empty{}, nil
			},
		},
	})

	err := svc.SimulatePayCallback(context.Background(), &PayCallbackRequest{
		UserID:  9,
		OrderSn: " order-9 ",
		PayType: "alipay",
		TradeNo: "trade-1",
		Success: true,
	})
	if err != nil {
		t.Fatalf("SimulatePayCallback() error = %v", err)
	}
	if got == nil {
		t.Fatal("SimulatePayCallback() did not call UpdateOrderStatus")
	}
	if got.OrderSn != "order-9" || got.Status != "TRADE_SUCCESS" || got.PayType != "alipay" || got.TradeNo != "trade-1" || got.PayTime <= 0 {
		t.Fatalf("UpdateOrderStatus() got %+v", got)
	}
}

func TestSimulatePayCallbackCanCloseOrder(t *testing.T) {
	var got *opb.OrderStatus
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			update: func(_ context.Context, in *opb.OrderStatus, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				got = in
				return &emptypb.Empty{}, nil
			},
		},
	})

	err := svc.SimulatePayCallback(context.Background(), &PayCallbackRequest{
		UserID:  9,
		OrderSn: "order-9",
		Success: false,
	})
	if err != nil {
		t.Fatalf("SimulatePayCallback() error = %v", err)
	}
	if got == nil || got.Status != "TRADE_CLOSED" {
		t.Fatalf("UpdateOrderStatus() got %+v, want TRADE_CLOSED", got)
	}
}

type fakeDataFactory struct {
	orderClient opb.OrderClient
}

func (f fakeDataFactory) Goods() gpb.GoodsClient {
	return nil
}

func (f fakeDataFactory) Orders() opb.OrderClient {
	return f.orderClient
}

func (f fakeDataFactory) Users() data.UserData {
	return nil
}

type fakeOrderClient struct {
	opb.OrderClient
	update func(context.Context, *opb.OrderStatus, ...grpc.CallOption) (*emptypb.Empty, error)
}

func (f fakeOrderClient) UpdateOrderStatus(ctx context.Context, in *opb.OrderStatus, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.update != nil {
		return f.update(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}
