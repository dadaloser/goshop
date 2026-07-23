package do

import "time"

type RefundRequestDO struct {
	ID            uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderSN       string    `gorm:"column:order_sn;type:varchar(64);not null;index"`
	ActorUserID   int32     `gorm:"column:actor_user_id;not null"`
	AmountFen     int64     `gorm:"column:amount_fen;not null"`
	Reason        string    `gorm:"column:reason;type:varchar(255);not null"`
	Status        string    `gorm:"column:status;type:varchar(24);not null"`
	CorrelationID string    `gorm:"column:correlation_id;type:char(36);not null;uniqueIndex"`
	CreatedAt     time.Time `gorm:"column:created_at;type:datetime(3);not null"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:datetime(3);not null"`
}

func (*RefundRequestDO) TableName() string { return "order_refund_requests" }
