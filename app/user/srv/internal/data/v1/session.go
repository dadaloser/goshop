package v1

import "time"

// UserSessionDO stores one revocable device session. RefreshTokenHash never
// contains a usable bearer credential.
type UserSessionDO struct {
	ID               string     `gorm:"column:id;type:char(36);primaryKey"`
	UserID           int32      `gorm:"column:user_id;not null;index"`
	PrincipalType    string     `gorm:"column:principal_type;type:varchar(32);not null"`
	RefreshTokenHash []byte     `gorm:"column:refresh_token_hash;type:binary(32);not null;uniqueIndex"`
	DeviceID         string     `gorm:"column:device_id;type:varchar(128);not null"`
	DeviceName       string     `gorm:"column:device_name;type:varchar(128);not null"`
	CreatedAt        time.Time  `gorm:"column:created_at;type:datetime(3);not null"`
	LastUsedAt       time.Time  `gorm:"column:last_used_at;type:datetime(3);not null"`
	ExpiresAt        time.Time  `gorm:"column:expires_at;type:datetime(3);not null"`
	RevokedAt        *time.Time `gorm:"column:revoked_at;type:datetime(3)"`
}

func (*UserSessionDO) TableName() string { return "user_sessions" }

// AccountDeletionOutboxEventDO is a durable record published only after the
// transaction that disables the account has committed.
type AccountDeletionOutboxEventDO struct {
	ID          string     `gorm:"column:id;type:char(36);primaryKey"`
	EventType   string     `gorm:"column:event_type;type:varchar(64);not null"`
	UserID      int32      `gorm:"column:user_id;not null;index:idx_user_deletion_outbox_claim"`
	Payload     []byte     `gorm:"column:payload;type:json;not null"`
	Status      string     `gorm:"column:status;type:varchar(16);not null;index:idx_user_deletion_outbox_claim"`
	RetryCount  int        `gorm:"column:retry_count;not null"`
	AvailableAt time.Time  `gorm:"column:available_at;type:datetime(3);not null;index:idx_user_deletion_outbox_claim"`
	LockedAt    *time.Time `gorm:"column:locked_at;type:datetime(3)"`
	PublishedAt *time.Time `gorm:"column:published_at;type:datetime(3)"`
	LastError   string     `gorm:"column:last_error;type:varchar(500);not null"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:datetime(3);not null"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;type:datetime(3);not null"`
}

func (*AccountDeletionOutboxEventDO) TableName() string { return "user_account_deletion_outbox" }

type StaffSessionRecordDO struct {
	ID            string
	UserID        int32
	PrincipalType string
	DeviceID      string
	DeviceName    string
	CreatedAt     time.Time
	LastUsedAt    time.Time
	ExpiresAt     time.Time
	RevokedAt     *time.Time
	Roles         []string
}

type StaffSessionFilters struct {
	UserID         int32
	Role           string
	ActiveOnly     bool
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	LastUsedAfter  *time.Time
	LastUsedBefore *time.Time
	Offset         int
	Limit          int
}

// VerificationCodeDO describes the reviewed verification-code schema used for
// delivery audit and future database-backed channels. Usable codes remain hashed.
type VerificationCodeDO struct {
	ID              uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Channel         string     `gorm:"column:channel;type:varchar(16);not null"`
	Purpose         string     `gorm:"column:purpose;type:varchar(16);not null"`
	DestinationHash []byte     `gorm:"column:destination_hash;type:binary(32);not null"`
	CodeHash        []byte     `gorm:"column:code_hash;type:binary(32);not null"`
	Attempts        uint       `gorm:"column:attempts;not null"`
	ExpiresAt       time.Time  `gorm:"column:expires_at;type:datetime(3);not null"`
	ConsumedAt      *time.Time `gorm:"column:consumed_at;type:datetime(3)"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:datetime(3);not null"`
}

func (*VerificationCodeDO) TableName() string { return "verification_codes" }
