package do

import "time"

type RefundRequestDO struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderSN          string    `gorm:"column:order_sn;type:varchar(64);not null;index"`
	ActorUserID      int32     `gorm:"column:actor_user_id;not null"`
	AmountFen        int64     `gorm:"column:amount_fen;not null"`
	Reason           string    `gorm:"column:reason;type:varchar(255);not null"`
	Status           string    `gorm:"column:status;type:varchar(24);not null"`
	Provider         string    `gorm:"column:provider;type:varchar(32);not null"`
	ProviderRefundID string    `gorm:"column:provider_refund_id;type:varchar(128);not null"`
	FailureReason    string    `gorm:"column:failure_reason;type:varchar(255);not null"`
	CorrelationID    string    `gorm:"column:correlation_id;type:char(36);not null;uniqueIndex"`
	CreatedAt        time.Time `gorm:"column:created_at;type:datetime(3);not null"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:datetime(3);not null"`
}

func (*RefundRequestDO) TableName() string { return "order_refund_requests" }

type RefundOutboxDO struct {
	ID              uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	RefundRequestID uint64          `gorm:"column:refund_request_id;not null;uniqueIndex"`
	Status          string          `gorm:"column:status;type:varchar(24);not null"`
	Attempts        int             `gorm:"column:attempts;not null"`
	AvailableAt     time.Time       `gorm:"column:available_at;type:datetime(3);not null"`
	LockedAt        *time.Time      `gorm:"column:locked_at;type:datetime(3)"`
	LastError       string          `gorm:"column:last_error;type:varchar(255);not null"`
	CreatedAt       time.Time       `gorm:"column:created_at;type:datetime(3);not null"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;type:datetime(3);not null"`
	RefundRequest   RefundRequestDO `gorm:"foreignKey:RefundRequestID"`
}

func (*RefundOutboxDO) TableName() string { return "order_refund_outbox" }

type RefundJob struct {
	OutboxID        uint64
	RefundRequestID uint64
	OrderSN         string
	TradeNo         string
	AmountFen       int64
	Reason          string
	CorrelationID   string
	Attempts        int
}
