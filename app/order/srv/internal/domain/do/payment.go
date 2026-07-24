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

type PaymentReconciliationRunDO struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Provider      string     `gorm:"column:provider;type:varchar(32);not null"`
	WindowStart   time.Time  `gorm:"column:window_start;type:datetime(3);not null"`
	WindowEnd     time.Time  `gorm:"column:window_end;type:datetime(3);not null"`
	StartedAt     time.Time  `gorm:"column:started_at;type:datetime(3);not null"`
	FinishedAt    *time.Time `gorm:"column:finished_at;type:datetime(3)"`
	CheckedCount  int        `gorm:"column:checked_count;not null"`
	MismatchCount int        `gorm:"column:mismatch_count;not null"`
	Status        string     `gorm:"column:status;type:varchar(24);not null"`
}

func (*PaymentReconciliationRunDO) TableName() string { return "payment_reconciliation_runs" }

type PaymentReconciliationItemDO struct {
	ID                uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	RunID             uint64    `gorm:"column:run_id;not null;index"`
	ProviderEventID   string    `gorm:"column:provider_event_id;type:varchar(128);not null"`
	OrderSN           string    `gorm:"column:order_sn;type:varchar(64);not null"`
	TradeNo           string    `gorm:"column:trade_no;type:varchar(128);not null"`
	EventType         string    `gorm:"column:event_type;type:varchar(32);not null"`
	ProviderAmountFen int64     `gorm:"column:provider_amount_fen;not null"`
	LocalAmountFen    int64     `gorm:"column:local_amount_fen;not null"`
	Result            string    `gorm:"column:result;type:varchar(24);not null"`
	Detail            string    `gorm:"column:detail;type:varchar(255);not null"`
	CreatedAt         time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (*PaymentReconciliationItemDO) TableName() string { return "payment_reconciliation_items" }
