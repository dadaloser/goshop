package v1

import (
	"context"
	"goshop/app/order/srv/internal/domain/do"
	metav1 "goshop/pkg/common/meta/v1"
	"time"

	"gorm.io/gorm"
)

type OrderStore interface {
	Get(ctx context.Context, orderSn string) (*do.OrderInfoDO, error)

	List(ctx context.Context, userID uint64, meta metav1.ListMeta, orderBy []string) (*do.OrderInfoDOList, error)

	Create(ctx context.Context, txn *gorm.DB, order *do.OrderInfoDO) error

	DeleteByOrderSn(ctx context.Context, txn *gorm.DB, orderSn string) error

	Update(ctx context.Context, txn *gorm.DB, order *do.OrderInfoDO) error

	ListCloseCandidates(ctx context.Context, statuses []string, createdBefore time.Time, limit int) ([]*do.OrderInfoDO, error)

	ListFinishCandidates(ctx context.Context, status string, paidBefore time.Time, limit int) ([]*do.OrderInfoDO, error)
}
