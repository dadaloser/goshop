package do

import "goshop/app/pkg/gorm"

type OrderStatusLogDO struct {
	gorm.BaseModel

	OrderID    int32  `gorm:"type:int;index"`
	OrderSn    string `gorm:"type:varchar(30);index"`
	FromStatus string `gorm:"type:varchar(20)"`
	ToStatus   string `gorm:"type:varchar(20);index"`
	Reason     string `gorm:"type:varchar(128)"`
	Source     string `gorm:"type:varchar(64)"`
	Operator   string `gorm:"type:varchar(64)"`
}

func (OrderStatusLogDO) TableName() string {
	return "order_status_logs"
}
