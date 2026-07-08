package do

import (
	gorm2 "goshop/app/pkg/gorm"
)

const (
	OutboxStatusPending    = "PENDING"
	OutboxStatusProcessing = "PROCESSING"
	OutboxStatusDone       = "DONE"
	OutboxStatusDead       = "DEAD"

	OutboxTopicGoodsSync = "goods.search.sync"

	OutboxActionUpsert = "UPSERT"
	OutboxActionDelete = "DELETE"
)

type OutboxEventDO struct {
	gorm2.BaseModel

	Topic          string `gorm:"type:varchar(64);not null;index:idx_outbox_status_topic"`
	AggregateType  string `gorm:"type:varchar(32);not null"`
	AggregateID    int32  `gorm:"type:int;not null;index:idx_outbox_aggregate"`
	Action         string `gorm:"type:varchar(16);not null"`
	Payload        string `gorm:"type:text;not null"`
	Status         string `gorm:"type:varchar(16);not null;index:idx_outbox_status_topic"`
	RetryCount     int32  `gorm:"type:int;not null;default:0"`
	MaxRetryCount  int32  `gorm:"type:int;not null;default:5"`
	LastError      string `gorm:"type:text"`
	NextAttemptAt  int64  `gorm:"type:bigint;not null;default:0;index:idx_outbox_status_topic"`
	ProcessingLock string `gorm:"type:varchar(64)"`
}

func (OutboxEventDO) TableName() string {
	return "outbox_events"
}
