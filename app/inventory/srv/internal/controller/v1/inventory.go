package v1

import (
	"context"
	invpb "goshop/api/inventory/v1"
	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/inventory/srv/internal/domain/dto"
	v1 "goshop/app/inventory/srv/internal/service/v1"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type inventoryServer struct {
	invpb.UnimplementedInventoryServer
	srv v1.ServiceFactory
}

// 设置库存
func (is *inventoryServer) SetInv(ctx context.Context, info *invpb.GoodsInvInfo) (*emptypb.Empty, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "inventory request is required")
	}

	invDTO := &dto.InventoryDTO{}
	invDTO.Goods = info.GoodsId
	invDTO.Stocks = info.Num
	invDTO.Total = info.Total
	invDTO.Available = info.Available
	invDTO.Locked = info.Locked
	invDTO.Sold = info.Sold
	err := is.srv.Inventory().Create(ctx, invDTO)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (is *inventoryServer) SetStock(ctx context.Context, info *invpb.GoodsInvInfo) (*emptypb.Empty, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "inventory request is required")
	}
	inv := &dto.InventoryDTO{}
	inv.Goods = info.GoodsId
	inv.Stocks = info.Num
	inv.Total = info.Total
	inv.Available = info.Available
	inv.Locked = info.Locked
	inv.Sold = info.Sold
	err := is.srv.Inventory().Adjust(ctx, inv, &do.InventoryAdjustmentDO{ActorUserID: info.ActorUserId, CorrelationID: info.CorrelationId, RequestID: info.RequestId, Reason: info.Reason})
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (is *inventoryServer) ListAdjustments(ctx context.Context, req *invpb.InventoryAdjustmentListRequest) (*invpb.InventoryAdjustmentListResponse, error) {
	items, total, err := is.srv.Inventory().ListAdjustments(ctx, uint64(req.GetGoodsId()), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	resp := &invpb.InventoryAdjustmentListResponse{Total: int32(total), Data: make([]*invpb.InventoryAdjustment, 0, len(items))}
	for _, item := range items {
		resp.Data = append(resp.Data, &invpb.InventoryAdjustment{Id: int64(item.ID), GoodsId: item.GoodsID, BeforeAvailable: item.BeforeAvailable, AfterAvailable: item.AfterAvailable, ActorUserId: item.ActorUserID, CorrelationId: item.CorrelationID, RequestId: item.RequestID, Reason: item.Reason, CreatedAt: item.CreatedAt.Unix()})
	}
	return resp, nil
}

func (is *inventoryServer) InvDetail(ctx context.Context, info *invpb.GoodsInvInfo) (*invpb.GoodsInvInfo, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "inventory request is required")
	}

	inv, err := is.srv.Inventory().Get(ctx, uint64(info.GoodsId))
	if err != nil {
		return nil, err
	}
	return &invpb.GoodsInvInfo{
		GoodsId:   inv.Goods,
		Num:       inv.Stocks,
		Total:     inv.Total,
		Available: inv.Available,
		Locked:    inv.Locked,
		Sold:      inv.Sold,
	}, nil
}

func (is *inventoryServer) GetStock(ctx context.Context, info *invpb.GoodsInvInfo) (*invpb.GoodsInvInfo, error) {
	return is.InvDetail(ctx, info)
}

func (is *inventoryServer) GetSellDetail(ctx context.Context, info *invpb.OrderInfo) (*invpb.SellDetailInfo, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "inventory order request is required")
	}

	detail, err := is.srv.Inventory().GetOrderDetail(ctx, info.OrderSn)
	if err != nil {
		return nil, err
	}

	resp := &invpb.SellDetailInfo{
		OrderSn:    detail.OrderSn,
		Status:     detail.Status,
		StatusName: stockSellStatusName(detail.Status),
	}
	for _, item := range detail.Detail {
		resp.GoodsInfo = append(resp.GoodsInfo, &invpb.GoodsInvInfo{
			GoodsId: item.Goods,
			Num:     item.Num,
		})
	}
	return resp, nil
}

func (is *inventoryServer) Sell(ctx context.Context, info *invpb.SellInfo) (*emptypb.Empty, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "inventory sell request is required")
	}

	var detail []do.GoodsDetail
	for _, value := range info.GoodsInfo {
		detail = append(detail, do.GoodsDetail{Goods: value.GoodsId, Num: value.Num})
	}
	err := is.srv.Inventory().Sell(ctx, info.OrderSn, detail)
	if err != nil {
		if errors.IsCode(err, code.ErrInvNotEnough) {
			return nil, status.Error(codes.Aborted, err.Error())
		}
		return nil, err
	}
	//time.Sleep(5 * time.Second)
	//return nil, status.Errorf(codes.Aborted, " err.Error()") //测试
	return &emptypb.Empty{}, nil
}

func (is *inventoryServer) Reserve(ctx context.Context, info *invpb.SellInfo) (*emptypb.Empty, error) {
	return is.Sell(ctx, info)
}

func (is *inventoryServer) Reback(ctx context.Context, info *invpb.SellInfo) (*emptypb.Empty, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "inventory reback request is required")
	}

	log.Infof("订单%s归还库存", info.OrderSn)
	var detail []do.GoodsDetail
	for _, v := range info.GoodsInfo {
		detail = append(detail, do.GoodsDetail{Goods: v.GoodsId, Num: v.Num})
	}
	err := is.srv.Inventory().Reback(ctx, info.OrderSn, detail)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (is *inventoryServer) Confirm(ctx context.Context, info *invpb.SellInfo) (*emptypb.Empty, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "inventory confirm request is required")
	}

	var detail []do.GoodsDetail
	for _, value := range info.GoodsInfo {
		detail = append(detail, do.GoodsDetail{Goods: value.GoodsId, Num: value.Num})
	}
	if err := is.srv.Inventory().Confirm(ctx, info.OrderSn, detail); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (is *inventoryServer) Release(ctx context.Context, info *invpb.SellInfo) (*emptypb.Empty, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "inventory release request is required")
	}

	var detail []do.GoodsDetail
	for _, value := range info.GoodsInfo {
		detail = append(detail, do.GoodsDetail{Goods: value.GoodsId, Num: value.Num})
	}
	if err := is.srv.Inventory().Release(ctx, info.OrderSn, detail); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func NewInventoryServer(srv v1.ServiceFactory) *inventoryServer {
	return &inventoryServer{srv: srv}
}

func stockSellStatusName(status int32) string {
	switch status {
	case 1:
		return "reserved"
	case 2:
		return "released"
	case 3:
		return "confirmed"
	default:
		return "unknown"
	}
}

var (
	_ invpb.InventoryServer = &inventoryServer{}
)
