package do

import "time"

// AccountDeletionInboxDO makes request processing idempotent by request ID.
type AccountDeletionInboxDO struct {
	RequestID string    `gorm:"column:request_id;type:char(36);primaryKey"`
	UserID    int32     `gorm:"column:user_id;not null;index"`
	Decision  string    `gorm:"column:decision;type:varchar(16);not null"`
	Reason    string    `gorm:"column:reason;type:varchar(255);not null"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (*AccountDeletionInboxDO) TableName() string { return "order_account_deletion_inbox" }

type AccountDeletionOutboxDO struct {
	ID          string     `gorm:"column:id;type:char(36);primaryKey"`
	RequestID   string     `gorm:"column:request_id;type:char(36);not null;uniqueIndex"`
	EventType   string     `gorm:"column:event_type;type:varchar(64);not null"`
	UserID      int32      `gorm:"column:user_id;not null;index"`
	Payload     []byte     `gorm:"column:payload;type:json;not null"`
	Status      string     `gorm:"column:status;type:varchar(16);not null;index"`
	RetryCount  int        `gorm:"column:retry_count;not null"`
	AvailableAt time.Time  `gorm:"column:available_at;type:datetime(3);not null"`
	LockedAt    *time.Time `gorm:"column:locked_at;type:datetime(3)"`
	PublishedAt *time.Time `gorm:"column:published_at;type:datetime(3)"`
	LastError   string     `gorm:"column:last_error;type:varchar(500);not null"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:datetime(3);not null"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;type:datetime(3);not null"`
}

func (*AccountDeletionOutboxDO) TableName() string { return "order_account_deletion_outbox" }
