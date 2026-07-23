package v1

import "time"

type AdminAuditLogDO struct {
	ID                 uint64    `gorm:"primarykey"`
	TargetUserID       int32     `gorm:"index:idx_admin_audit_logs_target"`
	ActorUserID        int32     `gorm:"index:idx_admin_audit_logs_actor"`
	ActorPrincipalType string    `gorm:"type:varchar(32);not null"`
	Action             string    `gorm:"type:varchar(64);not null"`
	Detail             string    `gorm:"type:text"`
	CorrelationID      *string   `gorm:"column:correlation_id;type:char(36);uniqueIndex"`
	RequestID          string    `gorm:"column:request_id;type:varchar(128);not null"`
	TargetType         string    `gorm:"column:target_type;type:varchar(32);not null"`
	TargetID           string    `gorm:"column:target_id;type:varchar(128);not null"`
	Domain             string    `gorm:"column:domain;type:varchar(32);not null"`
	StoreID            string    `gorm:"column:store_id;type:varchar(64);not null"`
	TeamID             string    `gorm:"column:team_id;type:varchar(64);not null"`
	CreatedAt          time.Time `gorm:"column:add_time;autoCreateTime"`
}

func (a *AdminAuditLogDO) TableName() string {
	return "admin_audit_logs"
}

const (
	AdminAuditActionStaffLoginSucceeded     = "staff_login_succeeded"
	AdminAuditActionBreakGlassSessionIssued = "break_glass_session_issued"
	AdminAuditActionGoodsCreated            = "goods_created"
	AdminAuditActionGoodsUpdated            = "goods_updated"
	AdminAuditActionGoodsDeleted            = "goods_deleted"
	AdminAuditActionInventoryAdjusted       = "inventory_adjusted"
	AdminAuditActionOrderClosed             = "order_closed"
	AdminAuditActionOrderRefundRequested    = "order_refund_requested"
)
