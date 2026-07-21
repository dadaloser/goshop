package v1

import (
	"context"
	v1 "goshop/app/inventory/srv/internal/data/v1"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"sort"
	"strings"

	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis"

	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/inventory/srv/internal/domain/dto"

	"goshop/pkg/log"
)

const (
	orderLockPrefix = "order_"

	stockSellStatusReserved  int32 = 1 //已预留
	stockSellStatusReleased  int32 = 2 //已释放
	stockSellStatusConfirmed int32 = 3 //已确认
)

type InventorySrv interface {
	//设置库存
	Create(ctx context.Context, inv *dto.InventoryDTO) error

	//根据商品的id查询库存
	Get(ctx context.Context, goodsID uint64) (*dto.InventoryDTO, error)

	//根据订单号查询库存预留记录
	GetOrderDetail(ctx context.Context, orderSn string) (*do.StockSellDetailDO, error)

	//扣减库存
	Sell(ctx context.Context, orderSn string, detail []do.GoodsDetail) error

	//归还库存
	Reback(ctx context.Context, orderSn string, detail []do.GoodsDetail) error

	//支付成功确认库存
	Confirm(ctx context.Context, orderSn string, detail []do.GoodsDetail) error

	//支付失败或关闭释放库存
	Release(ctx context.Context, orderSn string, detail []do.GoodsDetail) error
}

type inventoryService struct {
	data v1.DataFactory

	redisOptions *options.RedisOptions

	pool redsyncredis.Pool

	testTxn txExecutor
}

func (is *inventoryService) beginTxn() txExecutor {
	if is != nil && is.testTxn != nil {
		return is.testTxn
	}
	return gormTxn{db: is.data.Begin()}
}

func (is *inventoryService) Create(ctx context.Context, inv *dto.InventoryDTO) error {
	if inv == nil || inv.Goods <= 0 || inv.Stocks < 0 || inv.Total < 0 || inv.Available < 0 || inv.Locked < 0 || inv.Sold < 0 {
		return errors.WithCode(code2.ErrValidation, "inventory is invalid")
	}
	return is.data.Inventories().Create(ctx, &inv.InventoryDO)
}

func (is *inventoryService) Get(ctx context.Context, goodsID uint64) (*dto.InventoryDTO, error) {
	if goodsID == 0 {
		return nil, errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
	}

	inv, err := is.data.Inventories().Get(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	return &dto.InventoryDTO{InventoryDO: *inv}, nil
}

func (is *inventoryService) GetOrderDetail(ctx context.Context, orderSn string) (*do.StockSellDetailDO, error) {
	if strings.TrimSpace(orderSn) == "" {
		return nil, errors.WithCode(code2.ErrValidation, "order_sn不能为空")
	}
	return is.data.Inventories().GetSellDetail(ctx, nil, orderSn)
}

func (is *inventoryService) Sell(ctx context.Context, ordersn string, details []do.GoodsDetail) error {
	if err := validateStockOperation(ordersn, details); err != nil {
		return err
	}

	log.Infof("订单%s扣减库存", ordersn)

	mutex, err := is.lockOrder(ctx, ordersn)
	if err != nil {
		log.Errorf("订单%s获取锁失败", ordersn)
		return err
	}
	defer unlockOrderMutex(mutex, ordersn)

	//实际上批量扣减库存的时候， 我们经常会先按照商品的id排序，然后从小大小逐个扣减库存，这样可以减少锁的竞争
	//如果无序的话 那么就有可能订单a 扣减 1,3,4 订单B 扣减 3,2,1
	var detail = do.GoodsDetailList(details)
	sort.Sort(detail)

	return withTxnExecutor(is.beginTxn(), "sell inventory", func(txn txExecutor) error {
		existing, err := is.data.Inventories().GetSellDetail(ctx, txn.DB(), ordersn)
		if err == nil {
			switch existing.Status {
			case stockSellStatusReserved:
				log.Infof("订单%s库存已经扣减, 忽略重复扣减", ordersn)
				return nil
			case stockSellStatusReleased, stockSellStatusConfirmed:
				log.Infof("订单%s库存已经归还或取消, 忽略延迟扣减", ordersn)
				return nil
			}
		} else if !errors.IsCode(err, code.ErrInvSellDetailNotFound) {
			log.Errorf("订单%s获取扣减库存记录失败", ordersn)
			return err
		}

		sellDetail := do.StockSellDetailDO{
			OrderSn: ordersn,
			Status:  stockSellStatusReserved,
			Detail:  detail,
		}

		for _, goodsInfo := range detail {
			if err := is.data.Inventories().Reduce(ctx, txn.DB(), uint64(goodsInfo.Goods), int(goodsInfo.Num)); err != nil {
				log.Errorf("订单%s扣减库存失败", ordersn)
				return err
			}
		}

		if err := is.data.Inventories().CreateStockSellDetail(ctx, txn.DB(), &sellDetail); err != nil {
			log.Errorf("订单%s创建扣减库存记录失败", ordersn)
			return err
		}
		return nil
	})
}

func (is *inventoryService) Reback(ctx context.Context, ordersn string, details []do.GoodsDetail) error {
	if err := validateStockOperation(ordersn, details); err != nil {
		return err
	}

	log.Infof("订单%s归还库存", ordersn)

	//库存归还的时候有不少细节
	//1. 主动取消 2. 网络问题引起的重试 3. 超时取消 4. 退款取消
	//会出现重复归还的情况，所以我们要先查询一下扣减库存记录的状态，如果已经是归还状态了，那么就不需要再归还了
	//需要分布式锁
	mutex, err := is.lockOrder(ctx, ordersn)
	if err != nil {
		log.Errorf("订单%s获取锁失败", ordersn)
		return err
	}
	defer unlockOrderMutex(mutex, ordersn)

	return withTxnExecutor(is.beginTxn(), "release inventory", func(txn txExecutor) error {
		sellDetail, err := is.data.Inventories().GetSellDetail(ctx, txn.DB(), ordersn)
		if err != nil {
			if errors.IsCode(err, code.ErrInvSellDetailNotFound) {
				detail := do.GoodsDetailList(details)
				sort.Sort(detail)
				created, err := is.data.Inventories().CreateStockSellDetailIfAbsent(ctx, txn.DB(), &do.StockSellDetailDO{
					OrderSn: ordersn,
					Status:  stockSellStatusReleased,
					Detail:  detail,
				})
				if err != nil {
					log.Errorf("订单%s创建空回滚记录失败", ordersn)
					return err
				}
				if created {
					log.Infof("订单%s扣减库存记录不存在, 已记录空回滚", ordersn)
				}
				return nil
			}
			log.Errorf("订单%s获取扣减库存记录失败", ordersn)
			return err
		}

		if sellDetail.Status == stockSellStatusReleased {
			log.Infof("订单%s扣减库存记录已经归还, 忽略", ordersn)
			return nil
		}
		if sellDetail.Status == stockSellStatusConfirmed {
			log.Infof("订单%s扣减库存记录已经确认, 忽略归还", ordersn)
			return nil
		}

		detail := sellDetail.Detail
		if len(detail) == 0 {
			log.Errorf("订单%s扣减库存记录明细为空", ordersn)
			return errors.WithCode(code2.ErrDatabase, "inventory sell detail is empty")
		}
		sort.Sort(detail)

		for _, goodsInfo := range detail {
			if err := is.data.Inventories().Increase(ctx, txn.DB(), uint64(goodsInfo.Goods), int(goodsInfo.Num)); err != nil {
				log.Errorf("订单%s归还库存失败", ordersn)
				return err
			}
		}

		if err := is.data.Inventories().UpdateStockSellDetailStatus(ctx, txn.DB(), ordersn, stockSellStatusReleased); err != nil {
			log.Errorf("订单%s更新扣减库存记录失败", ordersn)
			return err
		}
		return nil
	})
}

func (is *inventoryService) Confirm(ctx context.Context, ordersn string, details []do.GoodsDetail) error {
	if err := validateStockOperation(ordersn, details); err != nil {
		return err
	}

	log.Infof("订单%s确认库存", ordersn)

	mutex, err := is.lockOrder(ctx, ordersn)
	if err != nil {
		log.Errorf("订单%s获取锁失败", ordersn)
		return err
	}
	defer unlockOrderMutex(mutex, ordersn)

	return withTxnExecutor(is.beginTxn(), "confirm inventory", func(txn txExecutor) error {
		sellDetail, err := is.data.Inventories().GetSellDetail(ctx, txn.DB(), ordersn)
		if err != nil {
			log.Errorf("订单%s获取扣减库存记录失败", ordersn)
			return err
		}

		if sellDetail.Status == stockSellStatusReleased {
			log.Infof("订单%s库存记录已释放, 忽略确认", ordersn)
			return nil
		}
		if sellDetail.Status == stockSellStatusConfirmed {
			log.Infof("订单%s库存记录已确认, 忽略重复确认", ordersn)
			return nil
		}

		detail := append(do.GoodsDetailList(nil), sellDetail.Detail...)
		if len(detail) == 0 {
			log.Errorf("订单%s扣减库存记录明细为空", ordersn)
			return errors.WithCode(code2.ErrDatabase, "inventory sell detail is empty")
		}
		sort.Sort(detail)

		for _, goodsInfo := range detail {
			if err := is.data.Inventories().ConfirmSell(ctx, txn.DB(), uint64(goodsInfo.Goods), int(goodsInfo.Num)); err != nil {
				log.Errorf("订单%s确认库存失败", ordersn)
				return err
			}
		}

		if err := is.data.Inventories().UpdateStockSellDetailStatus(ctx, txn.DB(), ordersn, stockSellStatusConfirmed); err != nil {
			log.Errorf("订单%s更新确认库存记录失败", ordersn)
			return err
		}
		return nil
	})
}

func (is *inventoryService) Release(ctx context.Context, ordersn string, details []do.GoodsDetail) error {
	return is.Reback(ctx, ordersn, details)
}

func validateStockOperation(orderSn string, details []do.GoodsDetail) error {
	if strings.TrimSpace(orderSn) == "" {
		return errors.WithCode(code2.ErrValidation, "order_sn不能为空")
	}
	if len(details) == 0 {
		return errors.WithCode(code2.ErrValidation, "库存明细不能为空")
	}
	for _, detail := range details {
		if detail.Goods <= 0 || detail.Num <= 0 {
			return errors.WithCode(code2.ErrValidation, "库存明细参数错误")
		}
	}
	return nil
}

func stockSellStatusName(status int32) string {
	switch status {
	case stockSellStatusReserved:
		return "reserved"
	case stockSellStatusReleased:
		return "released"
	case stockSellStatusConfirmed:
		return "confirmed"
	default:
		return "unknown"
	}
}

func unlockOrderMutex(mutex *redsync.Mutex, orderSn string) {
	if mutex == nil {
		return
	}
	if _, err := mutex.Unlock(); err != nil {
		log.Errorf("订单%s释放锁出现异常: %v", orderSn, err)
	}
}

func (is *inventoryService) lockOrder(ctx context.Context, orderSn string) (*redsync.Mutex, error) {
	if is == nil || is.pool == nil {
		return nil, nil
	}
	mutex := redsync.New(is.pool).NewMutex(orderLockPrefix + orderSn)
	if err := mutex.LockContext(ctx); err != nil {
		return nil, err
	}
	return mutex, nil
}

func newInventoryService(s *service) *inventoryService {
	return &inventoryService{data: s.data, redisOptions: s.redisOptions, pool: s.pool}
}

var _ InventorySrv = &inventoryService{}
