package service

import (
	"context"
	proto2 "goshop/api/goods/v1"
	proto "goshop/api/inventory/v1"
	proto3 "goshop/api/order/v1"
	v12 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/order/srv/internal/domain/dto"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	v1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"github.com/dtm-labs/client/dtmgrpc"
)

const (
	inventory_busi = "discovery:///goshop-inventory-srv"
	goods_busi     = "discovery:///goshop-order-srv"
)

type OrderSrv interface {
	Get(ctx context.Context, orderSn string) (*dto.OrderDTO, error)
	List(ctx context.Context, userID uint64, meta v1.ListMeta, orderBy []string) (*dto.OrderDTOList, error)
	Submit(ctx context.Context, order *dto.OrderDTO) error
	Create(ctx context.Context, order *dto.OrderDTO) error
	CreateCom(ctx context.Context, order *dto.OrderDTO) error //这是create的补偿
	Update(ctx context.Context, order *dto.OrderDTO) error
}

type orderService struct {
	data    v12.DataFactory
	dtmOpts *options.DtmOptions
}

func (os *orderService) CreateCom(ctx context.Context, order *dto.OrderDTO) error {
	/*
		1. 删除orderinfo表
		2. 删除ordergoods表
		3. 删除order找到对应的购物车条目，删除购物车条目
	*/
	//其实不用回滚
	//你应该先查询订单是否已经存在，如果已经存在删除相关记录即可， 同时恢复购物车记录
	return nil
}

func (os *orderService) Create(ctx context.Context, order *dto.OrderDTO) error {
	/*
		1. 生成order_info表
		2. 生成order_goods表
		3. 根据order找到对应的购物车条目，删除购物车条目
	*/

	var goodsIds []int32
	for _, value := range order.OrderGoods {
		goodsIds = append(goodsIds, value.Goods)
	}

	//获取goods信息
	goods, err := os.data.Goods().BatchGetGoods(context.Background(), &proto2.BatchGoodsIdInfo{Id: goodsIds})
	if err != nil {
		log.Errorf("批量获取商品信息失败，goodids: %v, err:%v", goodsIds, err)
		return err
	}
	if len(goods.Data) != len(goodsIds) {
		log.Errorf("批量获取商品信息失败，goodids: %v, 返回值：%v, err:%v", goodsIds, goods.Data, err)
		return errors.WithCode(code.ErrGoodsNotFound, "商品不存在或者部分不存在")
	}

	var goodsMap = make(map[int32]*proto2.GoodsInfoResponse)
	for _, value := range goods.Data {
		goodsMap[value.Id] = value
	}

	//生成订单总金额
	var orderAmount float32
	for _, value := range order.OrderGoods {
		orderAmount += goodsMap[value.Goods].ShopPrice * float32(value.Nums)
		value.GoodsName = goodsMap[value.Goods].Name
		value.GoodsPrice = goodsMap[value.Goods].ShopPrice
		value.GoodsImage = goodsMap[value.Goods].GoodsFrontImage
	}

	txn := os.data.Begin() //开启事务
	defer func() {
		if err := recover(); err != nil {
			_ = txn.Rollback()
			log.Error("新建订单事务进行中出现异常，回滚")
			return
		}
	}()

	err = os.data.Orders().Create(ctx, txn, &order.OrderInfoDO)
	if err != nil {
		txn.Rollback()
		log.Errorf("创建订单失败，err:%v", err)
		return err //这个不是abort 也就是说会不停的重试
	}

	err = os.data.ShopCarts().DeleteByGoodsIDs(ctx, txn, uint64(order.User), goodsIds)
	if err != nil {
		txn.Rollback()
		log.Errorf("删除购物车失败，goodids:%v, err:%v", goodsIds, err)
		return err
	}

	txn.Commit()
	//这里有逻辑
	return nil
}

func (os *orderService) Get(ctx context.Context, orderSn string) (*dto.OrderDTO, error) {
	order, err := os.data.Orders().Get(ctx, orderSn)
	if err != nil {
		return nil, err
	}
	return &dto.OrderDTO{OrderInfoDO: *order}, nil
}

func (os *orderService) List(ctx context.Context, userID uint64, meta v1.ListMeta, orderBy []string) (*dto.OrderDTOList, error) {
	orders, err := os.data.Orders().List(ctx, userID, meta, orderBy)
	if err != nil {
		return nil, err
	}
	var ret dto.OrderDTOList
	ret.TotalCount = orders.TotalCount
	for _, value := range orders.Items {
		ret.Items = append(ret.Items, &dto.OrderDTO{
			*value,
		})
	}
	return &ret, nil
}

func (os *orderService) Submit(ctx context.Context, order *dto.OrderDTO) error {
	//先从购物车中获取商品信息
	list, err := os.data.ShopCarts().List(ctx, uint64(order.User), true, v1.ListMeta{}, []string{})
	if err != nil {
		log.Errorf("获取购物车信息失败，err:%v", err)
		return err
	}

	if len(list.Items) == 0 {
		log.Errorf("购物车中没有商品，无法下单")
		return errors.WithCode(code.ErrNoGoodsSelect, "没有选择商品")
	}

	var orderGoods []*do.OrderGoods
	var orderItems []*proto3.OrderItemResponse
	for _, value := range list.Items {
		orderGoods = append(orderGoods, &do.OrderGoods{
			Goods: value.Goods,
			Nums:  value.Nums,
		})

		orderItems = append(orderItems, &proto3.OrderItemResponse{
			GoodsId: value.Goods,
			Nums:    value.Nums,
		})
	}
	order.OrderGoods = orderGoods

	//基于可靠消息最终一致性的思想， saga事务来解决订单生成的问题
	var goodsInfo []*proto.GoodsInvInfo
	for _, value := range order.OrderGoods {
		goodsInfo = append(goodsInfo, &proto.GoodsInvInfo{
			GoodsId: value.Goods,
			Num:     value.Nums,
		})
	}
	req := &proto.SellInfo{
		GoodsInfo: goodsInfo,
		OrderSn:   order.OrderSn,
	}
	oReq := &proto3.OrderRequest{
		OrderSn:    order.OrderSn,
		UserId:     order.User,
		Address:    order.Address,
		Name:       order.SignerName,
		Mobile:     order.SingerMobile,
		Post:       order.Post,
		OrderItems: orderItems,
	}

	saga := dtmgrpc.NewSagaGrpc(os.dtmOpts.GrpcServer, order.OrderSn).
		Add(inventory_busi+"/Inventory/Sell", inventory_busi+"/Inventory/Reback", req).
		Add(goods_busi+"/Order/CreateOrder", goods_busi+"/Order/CreateOrderCom", oReq)
	saga.WaitResult = true
	err = saga.Submit()
	//通过OrderSn查询一下， 当前的状态如何状态一直是Submitted那么就你一直不要给前端返回， 如果是failed那么你提示给前端说下单失败，重新下单
	return err
}

func (os *orderService) Update(ctx context.Context, order *dto.OrderDTO) error {
	//TODO implement me
	panic("implement me")
}

func newOrderService(sv *service) *orderService {
	return &orderService{
		data:    sv.data,
		dtmOpts: sv.dtmopts,
	}
}

var _ OrderSrv = &orderService{}
