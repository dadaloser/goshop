package order

import (
	"context"
	"testing"

	pb "goshop/api/order/v1"
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
