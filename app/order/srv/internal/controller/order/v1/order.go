package order

import (
	"context"
	pb "goshop/api/order/v1"
	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/order/srv/internal/domain/dto"
	"goshop/app/order/srv/internal/service/v1"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/log"

	"google.golang.org/protobuf/types/known/emptypb"
)

type orderServer struct {
	pb.UnimplementedOrderServer

	srv service.ServiceFactory
}

func NewOrderServer(srv service.ServiceFactory) *orderServer {
	return &orderServer{srv: srv}
}

func (os *orderServer) CartItemList(ctx context.Context, info *pb.UserInfo) (*pb.CartItemListResponse, error) {
	list, err := os.srv.Orders().CartItemList(ctx, uint64(info.Id), metav1.ListMeta{}, []string{})
	if err != nil {
		return nil, err
	}
	ret := &pb.CartItemListResponse{Total: int32(list.TotalCount)}
	for _, item := range list.Items {
		ret.Data = append(ret.Data, shopCartToResponse(item))
	}
	return ret, nil
}

func (os *orderServer) CreateCartItem(ctx context.Context, request *pb.CartItemRequest) (*pb.ShopCartInfoResponse, error) {
	cart, err := os.srv.Orders().CreateCartItem(ctx, &dto.ShopCartDTO{
		ShoppingCartDO: do.ShoppingCartDO{
			User:    request.UserId,
			Goods:   request.GoodsId,
			Nums:    request.Nums,
			Checked: request.Checked,
		},
	})
	if err != nil {
		return nil, err
	}
	return shopCartToResponse(cart), nil
}

func (os *orderServer) UpdateCartItem(ctx context.Context, request *pb.CartItemRequest) (*emptypb.Empty, error) {
	err := os.srv.Orders().UpdateCartItem(ctx, &dto.ShopCartDTO{
		ShoppingCartDO: do.ShoppingCartDO{
			User:    request.UserId,
			Goods:   request.GoodsId,
			Nums:    request.Nums,
			Checked: request.Checked,
		},
	})
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (os *orderServer) DeleteCartItem(ctx context.Context, request *pb.CartItemRequest) (*emptypb.Empty, error) {
	if err := os.srv.Orders().DeleteCartItem(ctx, uint64(request.Id)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// 这个是给分布式事务saga调用的，目前没为api提供的目的
func (os *orderServer) CreateOrder(ctx context.Context, request *pb.OrderRequest) (*emptypb.Empty, error) {
	orderGoods := make([]*do.OrderGoods, len(request.OrderItems))
	for i, item := range request.OrderItems {
		orderGoods[i] = &do.OrderGoods{
			Goods: item.GoodsId,
			Nums:  item.Nums,
		}
	}

	err := os.srv.Orders().Create(ctx, &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			User:         request.UserId,
			Address:      request.Address,
			SignerName:   request.Name,
			SingerMobile: request.Mobile,
			Post:         request.Post,
			OrderSn:      request.OrderSn,
			OrderGoods:   orderGoods,
		},
	})
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (os *orderServer) CreateOrderCom(ctx context.Context, request *pb.OrderRequest) (*emptypb.Empty, error) {
	err := os.srv.Orders().CreateCom(ctx, &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			OrderSn: request.OrderSn,
		},
	})
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

//// 订单号的生成， 订单号-雪花算法，目前的订单号生成算法有问题： 不是递增
//func generateOrderSn(userId int32) string {
//	//订单号的生成规则
//	/*
//		年月日时分秒+用户id+2位随机数
//	*/
//	now := time.Now()
//	rand.Seed(time.Now().UnixNano())
//	orderSn := fmt.Sprintf("%d%d%d%d%d%d%d%d",
//		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Nanosecond(),
//		userId, rand.Intn(90)+10,
//	)
//	return orderSn
//}

/*
订单提交的时候应该是先生成订单号
订单号会单独做一个接口，订单查询，以及一系列的关联我们应该采用order_sn，不要再去采用id去关联
*/
func (os *orderServer) SubmitOrder(ctx context.Context, request *pb.OrderRequest) (*emptypb.Empty, error) {
	//从购物车中得到选中的商品
	orderDTO := dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			User:         request.UserId,
			Address:      request.Address,
			SignerName:   request.Name,
			SingerMobile: request.Mobile,
			Post:         request.Post,
			OrderSn:      request.OrderSn,
		},
	}
	err := os.srv.Orders().Submit(ctx, &orderDTO)
	if err != nil {
		log.Errorf("新建订单失败: %v", err)
		return nil, err
	}
	//另外一款解决ioc的库，wire
	return &emptypb.Empty{}, nil
}

func (os *orderServer) OrderList(ctx context.Context, request *pb.OrderFilterRequest) (*pb.OrderListResponse, error) {
	list, err := os.srv.Orders().List(ctx, uint64(request.UserId), metav1.ListMeta{
		Page:     int(request.Pages),
		PageSize: int(request.PagePerNums),
	}, []string{"add_time desc"})
	if err != nil {
		return nil, err
	}
	ret := &pb.OrderListResponse{Total: int32(list.TotalCount)}
	for _, item := range list.Items {
		ret.Data = append(ret.Data, orderToResponse(item))
	}
	return ret, nil
}

func (os *orderServer) OrderDetail(ctx context.Context, request *pb.OrderRequest) (*pb.OrderInfoDetailResponse, error) {
	order, err := os.srv.Orders().Get(ctx, uint64(request.UserId), request.OrderSn)
	if err != nil {
		return nil, err
	}
	return &pb.OrderInfoDetailResponse{
		OrderInfo: orderToResponse(order),
		Goods:     orderGoodsToResponse(order.OrderGoods),
	}, nil
}

func (os *orderServer) UpdateOrderStatus(ctx context.Context, status *pb.OrderStatus) (*emptypb.Empty, error) {
	orderDTO := &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			OrderSn: status.OrderSn,
			Status:  status.Status,
		},
	}
	if status.Id > 0 {
		orderDTO.ID = status.Id
	}
	if err := os.srv.Orders().Update(ctx, orderDTO); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

var _ pb.OrderServer = &orderServer{}

func shopCartToResponse(cart *dto.ShopCartDTO) *pb.ShopCartInfoResponse {
	return &pb.ShopCartInfoResponse{
		Id:      cart.ID,
		UserId:  cart.User,
		GoodsId: cart.Goods,
		Nums:    cart.Nums,
		Checked: cart.Checked,
	}
}

func orderToResponse(order *dto.OrderDTO) *pb.OrderInfoResponse {
	return &pb.OrderInfoResponse{
		Id:      order.ID,
		UserId:  order.User,
		OrderSn: order.OrderSn,
		PayType: order.PayType,
		Status:  order.Status,
		Post:    order.Post,
		Total:   order.OrderMount,
		Address: order.Address,
		Name:    order.SignerName,
		Mobile:  order.SingerMobile,
		AddTime: order.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func orderGoodsToResponse(goods []*do.OrderGoods) []*pb.OrderItemResponse {
	ret := make([]*pb.OrderItemResponse, 0, len(goods))
	for _, item := range goods {
		ret = append(ret, &pb.OrderItemResponse{
			Id:         item.ID,
			OrderId:    item.Order,
			GoodsId:    item.Goods,
			GoodsName:  item.GoodsName,
			GoodsImage: item.GoodsImage,
			GoodsPrice: item.GoodsPrice,
			Nums:       item.Nums,
		})
	}
	return ret
}
