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

	stockSellStatusReserved int32 = 1
	stockSellStatusReleased int32 = 2
)

type InventorySrv interface {
	//设置库存
	Create(ctx context.Context, inv *dto.InventoryDTO) error

	//根据商品的id查询库存
	Get(ctx context.Context, goodsID uint64) (*dto.InventoryDTO, error)

	//扣减库存
	Sell(ctx context.Context, orderSn string, detail []do.GoodsDetail) error

	//归还库存
	Reback(ctx context.Context, orderSn string, detail []do.GoodsDetail) error
}

type inventoryService struct {
	data v1.DataFactory

	redisOptions *options.RedisOptions

	pool redsyncredis.Pool
}

func (is *inventoryService) Create(ctx context.Context, inv *dto.InventoryDTO) error {
	return is.data.Inventories().Create(ctx, &inv.InventoryDO)
}

func (is *inventoryService) Get(ctx context.Context, goodsID uint64) (*dto.InventoryDTO, error) {
	inv, err := is.data.Inventories().Get(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	return &dto.InventoryDTO{InventoryDO: *inv}, nil
}

func (is *inventoryService) Sell(ctx context.Context, ordersn string, details []do.GoodsDetail) error {
	if err := validateStockOperation(ordersn, details); err != nil {
		return err
	}

	log.Infof("订单%s扣减库存", ordersn)

	rs := redsync.New(is.pool)
	mutex := rs.NewMutex(orderLockPrefix + ordersn)
	if err := mutex.LockContext(ctx); err != nil {
		log.Errorf("订单%s获取锁失败", ordersn)
		return err
	}
	defer unlockOrderMutex(mutex, ordersn)

	//实际上批量扣减库存的时候， 我们经常会先按照商品的id排序，然后从小大小逐个扣减库存，这样可以减少锁的竞争
	//如果无序的话 那么就有可能订单a 扣减 1,3,4 订单B 扣减 3,2,1
	var detail = do.GoodsDetailList(details)
	sort.Sort(detail)

	txn := is.data.Begin()
	committed := false
	defer func() {
		if err := recover(); err != nil { //注意事务的回滚要放在defer中， 这样才能保证无论什么异常都能回滚,还要加recover防止死锁
			_ = txn.Rollback()
			log.Error("事务进行中出现异常，回滚")
			return
		}
		if !committed {
			_ = txn.Rollback()
		}
	}()

	existing, err := is.data.Inventories().GetSellDetail(ctx, txn, ordersn)
	if err == nil {
		switch existing.Status {
		case stockSellStatusReserved:
			log.Infof("订单%s库存已经扣减, 忽略重复扣减", ordersn)
			return nil
		case stockSellStatusReleased:
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
		err = is.data.Inventories().Reduce(ctx, txn, uint64(goodsInfo.Goods), int(goodsInfo.Num))
		if err != nil {
			log.Errorf("订单%s扣减库存失败", ordersn)
			return err
		}
	}

	err = is.data.Inventories().CreateStockSellDetail(ctx, txn, &sellDetail)
	if err != nil {
		log.Errorf("订单%s创建扣减库存记录失败", ordersn)
		return err
	}

	if err := txn.Commit().Error; err != nil {
		log.Errorf("订单%s提交扣减库存事务失败", ordersn)
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	committed = true
	return nil
}

func (is *inventoryService) Reback(ctx context.Context, ordersn string, details []do.GoodsDetail) error {
	if err := validateStockOperation(ordersn, details); err != nil {
		return err
	}

	log.Infof("订单%s归还库存", ordersn)

	rs := redsync.New(is.pool)

	//库存归还的时候有不少细节
	//1. 主动取消 2. 网络问题引起的重试 3. 超时取消 4. 退款取消
	//会出现重复归还的情况，所以我们要先查询一下扣减库存记录的状态，如果已经是归还状态了，那么就不需要再归还了
	//需要分布式锁
	mutex := rs.NewMutex(orderLockPrefix + ordersn)
	if err := mutex.LockContext(ctx); err != nil {
		log.Errorf("订单%s获取锁失败", ordersn)
		return err
	}
	defer unlockOrderMutex(mutex, ordersn)

	txn := is.data.Begin()
	committed := false
	defer func() {
		if err := recover(); err != nil {
			_ = txn.Rollback()
			log.Error("事务进行中出现异常，回滚")
			return
		}
		if !committed {
			_ = txn.Rollback()
		}
	}()

	sellDetail, err := is.data.Inventories().GetSellDetail(ctx, txn, ordersn)
	if err != nil {
		if errors.IsCode(err, code.ErrInvSellDetailNotFound) {
			detail := do.GoodsDetailList(details)
			sort.Sort(detail)
			created, err := is.data.Inventories().CreateStockSellDetailIfAbsent(ctx, txn, &do.StockSellDetailDO{
				OrderSn: ordersn,
				Status:  stockSellStatusReleased,
				Detail:  detail,
			})
			if err != nil {
				log.Errorf("订单%s创建空回滚记录失败", ordersn)
				return err
			}
			if err := txn.Commit().Error; err != nil {
				log.Errorf("订单%s提交空回滚事务失败", ordersn)
				return errors.WithCode(code2.ErrDatabase, err.Error())
			}
			committed = true
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

	var detail = sellDetail.Detail
	if len(detail) == 0 {
		log.Errorf("订单%s扣减库存记录明细为空", ordersn)
		return errors.WithCode(code2.ErrDatabase, "inventory sell detail is empty")
	}
	sort.Sort(detail)

	for _, goodsInfo := range detail {
		err = is.data.Inventories().Increase(ctx, txn, uint64(goodsInfo.Goods), int(goodsInfo.Num))
		if err != nil {
			log.Errorf("订单%s归还库存失败", ordersn)
			return err
		}
	}

	err = is.data.Inventories().UpdateStockSellDetailStatus(ctx, txn, ordersn, stockSellStatusReleased)
	if err != nil {
		log.Errorf("订单%s更新扣减库存记录失败", ordersn)
		return err
	}

	if err := txn.Commit().Error; err != nil {
		log.Errorf("订单%s提交归还库存事务失败", ordersn)
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	committed = true
	return nil
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

func unlockOrderMutex(mutex *redsync.Mutex, orderSn string) {
	if _, err := mutex.Unlock(); err != nil {
		log.Errorf("订单%s释放锁出现异常: %v", orderSn, err)
	}
}

func newInventoryService(s *service) *inventoryService {
	return &inventoryService{data: s.data, redisOptions: s.redisOptions, pool: s.pool}
}

var _ InventorySrv = &inventoryService{}
