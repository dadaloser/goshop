package srv

import (
	"time"

	"gorm.io/gorm"
	"goshop/app/pkg/options"

	"goshop/app/review/srv/internal/data"
	"goshop/app/review/srv/internal/data/db"
	"goshop/app/review/srv/internal/service"
)

type Service = service.Service
type GRPCServer = service.GRPCServer

func NewStore(db *gorm.DB) *data.Store { return data.NewStore(db) }
func New(store service.Repository, verifier service.PurchaseVerifier, opts ...service.Option) *Service {
	return service.New(store, verifier, opts...)
}
func WithOutboxWorker(pollInterval time.Duration, batchSize int) service.Option {
	return service.WithOutboxWorker(pollInterval, batchSize)
}
func NewDBOrderVerifier(db *gorm.DB) *service.DBOrderVerifier { return service.NewDBOrderVerifier(db) }
func NewGRPCServer(value *Service) *GRPCServer                { return service.NewGRPCServer(value) }
func GetDBFactoryOr(mysqlOpts *options.MySQLOptions) (*gorm.DB, error) {
	return db.GetDBFactoryOr(mysqlOpts)
}
