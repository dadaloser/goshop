package do

import "time"

type PaymentEventDO struct {
	ID                uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Provider          string     `gorm:"column:provider;type:varchar(32);not null"`
	EventID           string     `gorm:"column:event_id;type:varchar(128);not null"`
	OrderSN           string     `gorm:"column:order_sn;type:varchar(64);not null;index"`
	TradeNo           string     `gorm:"column:trade_no;type:varchar(128);not null"`
	EventType         string     `gorm:"column:event_type;type:varchar(32);not null"`
	OrderAmountFen    int64      `gorm:"column:order_amount_fen;not null"`
	ProviderAmountFen int64      `gorm:"column:provider_amount_fen;not null"`
	RefundAmountFen   int64      `gorm:"column:refund_amount_fen;not null"`
	Status            string     `gorm:"column:status;type:varchar(24);not null"`
	ErrorDetail       string     `gorm:"column:error_detail;type:varchar(255);not null"`
	ReceivedAt        time.Time  `gorm:"column:received_at;type:datetime(3);not null"`
	CompletedAt       *time.Time `gorm:"column:completed_at;type:datetime(3)"`
}

func (*PaymentEventDO) TableName() string { return "payment_events" }
