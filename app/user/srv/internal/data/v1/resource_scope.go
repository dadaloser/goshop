package v1

import "time"

type UserResourceScopeDO struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int32     `gorm:"column:user_id;not null;index"`
	Domain    string    `gorm:"column:domain;type:varchar(32);not null"`
	StoreID   string    `gorm:"column:store_id;type:varchar(64);not null"`
	TeamID    string    `gorm:"column:team_id;type:varchar(64);not null"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (*UserResourceScopeDO) TableName() string { return "user_resource_scopes" }
