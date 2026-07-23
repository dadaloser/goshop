package domain

import "time"

const (
	StatusPending  = "PENDING"
	StatusApproved = "APPROVED"
	StatusRejected = "REJECTED"
)

type Review struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    int32
	OrderSN   string
	GoodsID   int32
	Rating    int32
	Content   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Append    ReviewAppend `gorm:"foreignKey:ReviewID"`
	Reply     ReviewReply  `gorm:"foreignKey:ReviewID"`
}

func (Review) TableName() string { return "reviews" }

type ReviewAppend struct {
	ID        uint64 `gorm:"primaryKey"`
	ReviewID  uint64
	Content   string
	CreatedAt time.Time
}

func (ReviewAppend) TableName() string { return "review_appends" }

type ReviewReply struct {
	ID          uint64 `gorm:"primaryKey"`
	ReviewID    uint64
	ActorUserID int32
	Content     string
	CreatedAt   time.Time
}

func (ReviewReply) TableName() string { return "review_replies" }

type Audit struct {
	ID                                              uint64 `gorm:"primaryKey"`
	ReviewID                                        uint64
	ActorUserID                                     int32
	Action, FromStatus, ToStatus, RequestID, Reason string
	CreatedAt                                       time.Time
}

func (Audit) TableName() string { return "review_audit_logs" }

type OutboxEvent struct {
	ID                uint64 `gorm:"primaryKey"`
	EventKey          string
	GoodsID           int32
	EventType, Status string
	RetryCount        int32
	NextAttemptAt     int64
	LastError         string
	CreatedAt         time.Time
	CompletedAt       *time.Time
}

func (OutboxEvent) TableName() string { return "review_outbox_events" }

type Rating struct {
	GoodsID                  int32 `gorm:"primaryKey"`
	ApprovedCount, RatingSum int64
	AverageMilli             int32
	RebuiltAt                time.Time
}

func (Rating) TableName() string { return "review_product_ratings" }
