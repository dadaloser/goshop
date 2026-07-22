package v1

import "time"

type AdminAuditLogDO struct {
	ID                 uint64    `gorm:"primarykey"`
	TargetUserID       int32     `gorm:"index:idx_admin_audit_logs_target"`
	ActorUserID        int32     `gorm:"index:idx_admin_audit_logs_actor"`
	ActorPrincipalType string    `gorm:"type:varchar(32);not null"`
	Action             string    `gorm:"type:varchar(64);not null"`
	Detail             string    `gorm:"type:text"`
	CreatedAt          time.Time `gorm:"column:add_time;autoCreateTime"`
}

func (a *AdminAuditLogDO) TableName() string {
	return "admin_audit_logs"
}

const (
	AdminAuditActionStaffLoginSucceeded     = "staff_login_succeeded"
	AdminAuditActionBreakGlassSessionIssued = "break_glass_session_issued"
)
