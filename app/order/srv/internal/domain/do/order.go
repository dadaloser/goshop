package do

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"goshop/app/pkg/gorm"
)

type GormList []string

func (g GormList) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// 实现 sql.Scanner 接口，Scan 将 value 扫描至 Jsonb
func (g *GormList) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &g)
}

type OrderInfoDO struct {
	gorm.BaseModel

	OrderGoods []*OrderGoods `gorm:"foreignKey:Order;references:ID" json:"goods"`

	User    int32  `gorm:"type:int;index"`
	OrderSn string `gorm:"type:varchar(30);index"` //订单号，我们平台自己生成的订单号
	PayType string `gorm:"type:varchar(20) comment 'alipay(支付宝)， wechat(微信)'"`

	//status大家可以考虑使用iota来做
	Status        string     `gorm:"type:varchar(20)  comment 'PAYING(待支付), TRADE_SUCCESS(成功)， TRADE_CLOSED(超时关闭), WAIT_BUYER_PAY(交易创建), TRADE_FINISHED(交易结束)'"`
	TradeNo       string     `gorm:"type:varchar(100) comment '交易号'"` //交易号就是支付宝的订单号 查账
	OrderMountFen int64      `gorm:"type:bigint;not null;default:0"`
	PayTime       *time.Time `gorm:"type:datetime"`

	Address      string `gorm:"type:varchar(100)"`
	SignerName   string `gorm:"type:varchar(20)"`
	SingerMobile string `gorm:"type:varchar(11)"`
	Post         string `gorm:"type:varchar(20)"`
}

func (OrderInfoDO) TableName() string {
	return "orderinfo"
}

type OrderGoods struct {
	gorm.BaseModel

	Order int32 `gorm:"type:int;index"`
	Goods int32 `gorm:"type:int;index"`

	//把商品的信息保存下来了 ， 字段冗余， 高并发系统中我们一般都不会遵循三范式  做镜像 记录
	GoodsName     string `gorm:"type:varchar(100);index"`
	GoodsImage    string `gorm:"type:varchar(200)"`
	GoodsPriceFen int64  `gorm:"type:bigint;not null;default:0"`
	Nums          int32  `gorm:"type:int"`
}

func (OrderGoods) TableName() string {
	return "ordergoods"
}

type OrderInfoDOList struct {
	TotalCount int64          `json:"totalCount,omitempty"`
	Items      []*OrderInfoDO `json:"items"`
}
