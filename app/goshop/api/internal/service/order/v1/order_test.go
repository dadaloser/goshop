package v1

import (
	"context"
	"testing"

	gpb "goshop/api/goods/v1"
	ipb "goshop/api/inventory/v1"
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
			svc:  NewOrderService(fakeDataFactory{orderClient: fakeOrderClient{detail: orderDetailResponse}, inventoryClient: fakeInventoryClient{}}),
			req:  &PayCallbackRequest{UserID: 1, OrderSn: "order-1", Success: true},
			code: code.ErrOrderStatusInvalid,
		},
		{
			name: "empty order goods",
			svc: NewOrderService(fakeDataFactory{
				orderClient: fakeOrderClient{
					detail: func(context.Context, *opb.OrderRequest, ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
						return &opb.OrderInfoDetailResponse{
							OrderInfo: &opb.OrderInfoResponse{OrderSn: "order-1", Status: orderStatusWaitBuyerPay},
						}, nil
					},
				},
				inventoryClient: fakeInventoryClient{},
			}),
			req:  &PayCallbackRequest{UserID: 1, OrderSn: "order-1", TradeNo: "trade-1", Success: true},
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
	var gotDetail *opb.OrderRequest
	var got *opb.OrderStatus
	var confirm *ipb.SellInfo
	var calls []string
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			detail: func(_ context.Context, in *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
				gotDetail = in
				calls = append(calls, "detail")
				return orderDetailResponse(nil, nil)
			},
			update: func(_ context.Context, in *opb.OrderStatus, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				calls = append(calls, "update")
				got = in
				return &emptypb.Empty{}, nil
			},
		},
		inventoryClient: fakeInventoryClient{
			confirm: func(_ context.Context, in *ipb.SellInfo, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				calls = append(calls, "confirm")
				confirm = in
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
	if gotDetail == nil || gotDetail.UserId != 9 || gotDetail.OrderSn != "order-9" {
		t.Fatalf("OrderDetail() got %+v", gotDetail)
	}
	if got == nil {
		t.Fatal("SimulatePayCallback() did not call UpdateOrderStatus")
	}
	if got.OrderSn != "order-9" || got.Status != "TRADE_SUCCESS" || got.PayType != "alipay" || got.TradeNo != "trade-1" || got.PayTime <= 0 {
		t.Fatalf("UpdateOrderStatus() got %+v", got)
	}
	if confirm == nil || confirm.OrderSn != "order-9" || len(confirm.GoodsInfo) != 1 || confirm.GoodsInfo[0].GoodsId != 101 || confirm.GoodsInfo[0].Num != 2 {
		t.Fatalf("Confirm() got %+v", confirm)
	}
	if len(calls) != 3 || calls[0] != "detail" || calls[1] != "confirm" || calls[2] != "update" {
		t.Fatalf("call order = %v, want [detail confirm update]", calls)
	}
}

func TestSimulatePayCallbackCanCloseOrder(t *testing.T) {
	var got *opb.OrderStatus
	var release *ipb.SellInfo
	var calls []string
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			detail: func(_ context.Context, _ *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
				calls = append(calls, "detail")
				return orderDetailResponse(nil, nil)
			},
			update: func(_ context.Context, in *opb.OrderStatus, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				calls = append(calls, "update")
				got = in
				return &emptypb.Empty{}, nil
			},
		},
		inventoryClient: fakeInventoryClient{
			release: func(_ context.Context, in *ipb.SellInfo, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				calls = append(calls, "release")
				release = in
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
	if release == nil || release.OrderSn != "order-9" || len(release.GoodsInfo) != 1 || release.GoodsInfo[0].GoodsId != 101 || release.GoodsInfo[0].Num != 2 {
		t.Fatalf("Release() got %+v", release)
	}
	if len(calls) != 3 || calls[0] != "detail" || calls[1] != "release" || calls[2] != "update" {
		t.Fatalf("call order = %v, want [detail release update]", calls)
	}
}

func TestSimulatePayCallbackRejectsConflictingTerminalState(t *testing.T) {
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			detail: func(_ context.Context, _ *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
				return &opb.OrderInfoDetailResponse{
					OrderInfo: &opb.OrderInfoResponse{OrderSn: "order-9", Status: orderStatusTradeClosed},
					Goods: []*opb.OrderItemResponse{
						{GoodsId: 101, Nums: 2},
					},
				}, nil
			},
		},
		inventoryClient: fakeInventoryClient{},
	})

	err := svc.SimulatePayCallback(context.Background(), &PayCallbackRequest{
		UserID:  9,
		OrderSn: "order-9",
		TradeNo: "trade-1",
		Success: true,
	})
	if !errors.IsCode(err, code.ErrOrderStatusInvalid) {
		t.Fatalf("SimulatePayCallback() error = %v, want code %d", err, code.ErrOrderStatusInvalid)
	}
}

type fakeDataFactory struct {
	inventoryClient ipb.InventoryClient
	orderClient     opb.OrderClient
}

func (f fakeDataFactory) Goods() gpb.GoodsClient {
	return nil
}

func (f fakeDataFactory) Orders() opb.OrderClient {
	return f.orderClient
}

func (f fakeDataFactory) Inventory() ipb.InventoryClient {
	return f.inventoryClient
}

func (f fakeDataFactory) Users() data.UserData {
	return nil
}

type fakeOrderClient struct {
	opb.OrderClient
	detail func(context.Context, *opb.OrderRequest, ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error)
	update func(context.Context, *opb.OrderStatus, ...grpc.CallOption) (*emptypb.Empty, error)
}

func (f fakeOrderClient) OrderDetail(ctx context.Context, in *opb.OrderRequest, opts ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
	if f.detail != nil {
		return f.detail(ctx, in, opts...)
	}
	return orderDetailResponse(ctx, in, opts...)
}

func (f fakeOrderClient) UpdateOrderStatus(ctx context.Context, in *opb.OrderStatus, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.update != nil {
		return f.update(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

type fakeInventoryClient struct {
	ipb.InventoryClient
	confirm func(context.Context, *ipb.SellInfo, ...grpc.CallOption) (*emptypb.Empty, error)
	release func(context.Context, *ipb.SellInfo, ...grpc.CallOption) (*emptypb.Empty, error)
}

func (f fakeInventoryClient) Confirm(ctx context.Context, in *ipb.SellInfo, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.confirm != nil {
		return f.confirm(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

func (f fakeInventoryClient) Release(ctx context.Context, in *ipb.SellInfo, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.release != nil {
		return f.release(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

func orderDetailResponse(_ context.Context, _ *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
	return &opb.OrderInfoDetailResponse{
		OrderInfo: &opb.OrderInfoResponse{
			OrderSn: "order-9",
			Status:  orderStatusWaitBuyerPay,
		},
		Goods: []*opb.OrderItemResponse{
			{GoodsId: 101, Nums: 2},
		},
	}, nil
}
