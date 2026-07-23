package do

import "time"

type InventoryAdjustmentDO struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	GoodsID         int32     `gorm:"column:goods_id;not null;index"`
	BeforeAvailable int32     `gorm:"column:before_available;not null"`
	AfterAvailable  int32     `gorm:"column:after_available;not null"`
	ActorUserID     int32     `gorm:"column:actor_user_id;not null"`
	CorrelationID   string    `gorm:"column:correlation_id;type:char(36);not null;uniqueIndex"`
	RequestID       string    `gorm:"column:request_id;type:varchar(128);not null"`
	Reason          string    `gorm:"column:reason;type:varchar(255);not null"`
	CreatedAt       time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (*InventoryAdjustmentDO) TableName() string { return "inventory_adjustment_logs" }
