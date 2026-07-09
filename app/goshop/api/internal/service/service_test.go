package service

import (
	"context"
	"testing"

	gpb "goshop/api/goods/v1"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
)

func TestNilServiceFactoryReturnsSafeServices(t *testing.T) {
	var svc *service

	goodsSrv := svc.Goods()
	if goodsSrv == nil {
		t.Fatal("Goods() returned nil")
	}
	if _, err := goodsSrv.List(context.Background(), &gpb.GoodsFilterRequest{}); !errors.IsCode(err, code.ErrConnectGRPC) {
		t.Fatalf("Goods().List() error = %v, want code %d", err, code.ErrConnectGRPC)
	}

	inventorySrv := svc.Inventory()
	if inventorySrv == nil {
		t.Fatal("Inventory() returned nil")
	}
	if _, err := inventorySrv.Detail(context.Background(), 1); !errors.IsCode(err, code.ErrConnectGRPC) {
		t.Fatalf("Inventory().Detail() error = %v, want code %d", err, code.ErrConnectGRPC)
	}

	userSrv := svc.Users()
	if userSrv == nil {
		t.Fatal("Users() returned nil")
	}
	if _, err := userSrv.Get(context.Background(), 1); !errors.IsCode(err, code.ErrConnectGRPC) {
		t.Fatalf("Users().Get() error = %v, want code %d", err, code.ErrConnectGRPC)
	}

	orderSrv := svc.Orders()
	if orderSrv == nil {
		t.Fatal("Orders() returned nil")
	}
	if err := orderSrv.SimulatePayCallback(context.Background(), &orderv1.PayCallbackRequest{
		UserID:  1,
		OrderSn: "order-1",
		TradeNo: "trade-1",
		Success: true,
	}); !errors.IsCode(err, code.ErrConnectGRPC) {
		t.Fatalf("Orders().SimulatePayCallback() error = %v, want code %d", err, code.ErrConnectGRPC)
	}

	smsSrv := svc.Sms()
	if smsSrv == nil {
		t.Fatal("Sms() returned nil")
	}
	if err := smsSrv.SendSms(context.Background(), "13800138000", "template", "{}"); err == nil {
		t.Fatal("Sms().SendSms() error = nil, want non-nil")
	}
}
