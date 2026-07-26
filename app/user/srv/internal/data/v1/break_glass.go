package v1

import "time"

type BreakGlassApprovalDO struct {
	ID              string     `gorm:"column:id;type:char(36);primaryKey"`
	RequesterUserID int32      `gorm:"column:requester_user_id;not null"`
	ApproverUserID  int32      `gorm:"column:approver_user_id;not null"`
	Status          string     `gorm:"column:status;type:varchar(24);not null"`
	Reason          string     `gorm:"column:reason;type:varchar(255);not null"`
	RequestID       string     `gorm:"column:request_id;type:varchar(128);not null"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:datetime(3);not null"`
	ApprovedAt      *time.Time `gorm:"column:approved_at;type:datetime(3)"`
	ExpiresAt       time.Time  `gorm:"column:expires_at;type:datetime(3);not null"`
	UsedAt          *time.Time `gorm:"column:used_at;type:datetime(3)"`
}

func (*BreakGlassApprovalDO) TableName() string { return "break_glass_approvals" }
