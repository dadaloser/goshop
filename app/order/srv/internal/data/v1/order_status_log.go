package v1

import (
	"context"
	"goshop/app/order/srv/internal/domain/do"

	"gorm.io/gorm"
)

type OrderStatusLogStore interface {
	Create(ctx context.Context, txn *gorm.DB, entry *do.OrderStatusLogDO) error
}
