package v1

import (
	"context"
	v1 "goshop/app/inventory/srv/internal/data/v1"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	"goshop/pkg/errors"
	"sort"

	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis"

	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/inventory/srv/internal/domain/dto"

	"goshop/pkg/log"
)

const (
	inventoryLockPrefix = "inventory_"
	orderLockPrefix     = "order_"
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
	log.Infof("订单%s扣减库存", ordersn)
	//解决了空悬挂的问题
	//先查询刚才插入的记录是否存在，如果存在则说明已经cancel就不能执行了

	rs := redsync.New(is.pool)
	//实际上批量扣减库存的时候， 我们经常会先按照商品的id排序，然后从小大小逐个扣减库存，这样可以减少锁的竞争
	//如果无序的话 那么就有可能订单a 扣减 1,3,4 订单B 扣减 3,2,1
	var detail = do.GoodsDetailList(details)
	sort.Sort(detail)

	txn := is.data.Begin()
	defer func() {
		if err := recover(); err != nil { //注意事务的回滚要放在defer中， 这样才能保证无论什么异常都能回滚,还要加recover防止死锁
			_ = txn.Rollback()
			log.Error("事务进行中出现异常，回滚")
			return
		}
	}()

	//
	sellDetail := do.StockSellDetailDO{
		OrderSn: ordersn,
		Status:  1,
		Detail:  detail,
	}

	for _, goodsInfo := range detail {
		mutex := rs.NewMutex(inventoryLockPrefix + ordersn)
		if err := mutex.Lock(); err != nil {
			log.Errorf("订单%s获取锁失败", ordersn)
		}

		inv, err := is.data.Inventories().Get(ctx, uint64(goodsInfo.Goods))
		if err != nil {
			log.Errorf("订单%s获取库存失败", ordersn)
			return err
		}

		//判断库存是否充足
		if inv.Stocks < goodsInfo.Num {
			txn.Rollback() //一定要注意回滚
			log.Errorf("商品%d库存%d不足, 现有库存: %d", goodsInfo.Goods, goodsInfo.Num, inv.Stocks)
			return errors.WithCode(code.ErrInvNotEnough, "库存不足")
		}
		inv.Stocks -= goodsInfo.Num

		err = is.data.Inventories().Reduce(ctx, txn, uint64(goodsInfo.Goods), int(goodsInfo.Num))
		if err != nil {
			txn.Rollback() //回滚
			log.Errorf("订单%s扣减库存失败", ordersn)
			return err
		}

		//释放锁
		if _, err := mutex.Unlock(); err != nil {
			txn.Rollback() //回滚
			log.Errorf("订单%s释放锁出现异常", ordersn)
		}
	}

	err := is.data.Inventories().CreateStockSellDetail(ctx, txn, &sellDetail)
	if err != nil {
		txn.Rollback() //回滚
		log.Errorf("订单%s创建扣减库存记录失败", ordersn)
		return err
	}

	txn.Commit()
	return nil
}

func (is *inventoryService) Reback(ctx context.Context, ordersn string, details []do.GoodsDetail) error {
	log.Infof("订单%s归还库存", ordersn)

	rs := redsync.New(is.pool)

	txn := is.data.Begin()
	defer func() {
		if err := recover(); err != nil {
			_ = txn.Rollback()
			log.Error("事务进行中出现异常，回滚")
			return
		}
	}()

	//库存归还的时候有不少细节
	//1. 主动取消 2. 网络问题引起的重试 3. 超时取消 4. 退款取消
	//会出现重复归还的情况，所以我们要先查询一下扣减库存记录的状态，如果已经是归还状态了，那么就不需要再归还了
	//需要分布式锁
	mutex := rs.NewMutex(orderLockPrefix + ordersn)
	if err := mutex.Lock(); err != nil {
		txn.Rollback() //回滚
		log.Errorf("订单%s获取锁失败", ordersn)
		return err
	}
	sellDetail, err := is.data.Inventories().GetSellDetail(ctx, txn, ordersn)
	if err != nil {
		txn.Rollback()
		_, err := mutex.Unlock()
		if err != nil {
			log.Errorf("订单%s释放锁出现异常", ordersn)
			return err
		}
		if errors.IsCode(err, code.ErrInvSellDetailNotFound) {
			//空回滚
			log.Errorf("订单%s扣减库存记录不存在, 忽略", ordersn)
			//我应该记录一条数据去记录，说 ordersn已经被cancel了
			return nil
		}
		log.Errorf("订单%s获取扣减库存记录失败", ordersn)
		return err
	}

	if sellDetail.Status == 2 {
		log.Infof("订单%s扣减库存记录已经归还, 忽略", ordersn)
		return nil
	}

	var detail = do.GoodsDetailList(details)
	sort.Sort(detail)

	for _, goodsInfo := range detail {
		inv, err := is.data.Inventories().Get(ctx, uint64(goodsInfo.Goods))
		if err != nil {
			txn.Rollback() //回滚
			log.Errorf("订单%s获取库存失败", ordersn)
			return err
		}
		inv.Stocks += goodsInfo.Num

		err = is.data.Inventories().Increase(ctx, txn, uint64(goodsInfo.Goods), int(goodsInfo.Num))
		if err != nil {
			txn.Rollback() //回滚
			log.Errorf("订单%s归还库存失败", ordersn)
			return err
		}
	}

	err = is.data.Inventories().UpdateStockSellDetailStatus(ctx, txn, ordersn, 2)
	if err != nil {
		txn.Rollback() //回滚
		log.Errorf("订单%s更新扣减库存记录失败", ordersn)
		return err
	}

	txn.Commit()
	return nil
}

func newInventoryService(s *service) *inventoryService {
	return &inventoryService{data: s.data, redisOptions: s.redisOptions, pool: s.pool}
}

var _ InventorySrv = &inventoryService{}
