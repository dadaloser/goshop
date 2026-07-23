package srv

import (
	"gorm.io/gorm"

	"goshop/app/review/srv/internal/data"
	"goshop/app/review/srv/internal/service"
)

type Service = service.Service
type GRPCServer = service.GRPCServer

func NewStore(db *gorm.DB) *data.Store { return data.NewStore(db) }
func New(store service.Repository, verifier service.PurchaseVerifier) *Service {
	return service.New(store, verifier)
}
func NewDBOrderVerifier(db *gorm.DB) *service.DBOrderVerifier { return service.NewDBOrderVerifier(db) }
func NewGRPCServer(value *Service) *GRPCServer                { return service.NewGRPCServer(value) }
