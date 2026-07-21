package v1

import "time"

/*
*
用户审计日志
*/
type UserAuditLogDO struct {
	ID                 uint64    `gorm:"primarykey"`
	UserID             int32     `gorm:"index:idx_user_audit_logs_user;not null"`
	ActorUserID        int32     `gorm:"index:idx_user_audit_logs_actor"`
	ActorPrincipalType string    `gorm:"type:varchar(32);not null"`
	Action             string    `gorm:"type:varchar(64);not null"`
	FromStatus         string    `gorm:"type:varchar(16)"`
	ToStatus           string    `gorm:"type:varchar(16)"`
	Detail             string    `gorm:"type:text"`
	CreatedAt          time.Time `gorm:"column:add_time;autoCreateTime"`
}

func (u *UserAuditLogDO) TableName() string {
	return "user_audit_logs"
}

const (
	UserAuditActionStaffCreated  = "staff_user_created"
	UserAuditActionStatusUpdated = "staff_user_status_updated"
	UserAuditActionRolesReplaced = "staff_user_roles_replaced"
)
