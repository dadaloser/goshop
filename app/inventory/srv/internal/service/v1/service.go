package v1

import (
	"fmt"
	v1 "goshop/app/inventory/srv/internal/data/v1"
	"goshop/app/pkg/options"

	goredislib "github.com/redis/go-redis/v9"

	redsyncredis "github.com/go-redsync/redsync/v4/redis"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
)

type ServiceFactory interface {
	Inventory() InventorySrv
}

type service struct {
	data v1.DataFactory

	redisOptions *options.RedisOptions
	pool         redsyncredis.Pool
}

func (s *service) Inventory() InventorySrv {
	//TODO implement me
	return newInventoryService(s)
}

func NewService(store v1.DataFactory, redisOptions *options.RedisOptions) *service {
	client := goredislib.NewClient(&goredislib.Options{
		Addr: fmt.Sprintf("%s:%d", redisOptions.Host, redisOptions.Port),
	})
	//pool := goredis.NewPool(client)
	//or, pool := redigo.NewPool(...)
	pool := goredis.NewPool(client)

	return &service{data: store, redisOptions: redisOptions, pool: pool}
}

var _ ServiceFactory = &service{}
