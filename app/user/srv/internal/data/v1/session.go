package v1

import "time"

// UserSessionDO stores one revocable device session. RefreshTokenHash never
// contains a usable bearer credential.
type UserSessionDO struct {
	ID               string     `gorm:"column:id;type:char(36);primaryKey"`
	UserID           int32      `gorm:"column:user_id;not null;index"`
	RefreshTokenHash []byte     `gorm:"column:refresh_token_hash;type:binary(32);not null;uniqueIndex"`
	DeviceID         string     `gorm:"column:device_id;type:varchar(128);not null"`
	DeviceName       string     `gorm:"column:device_name;type:varchar(128);not null"`
	CreatedAt        time.Time  `gorm:"column:created_at;type:datetime(3);not null"`
	LastUsedAt       time.Time  `gorm:"column:last_used_at;type:datetime(3);not null"`
	ExpiresAt        time.Time  `gorm:"column:expires_at;type:datetime(3);not null"`
	RevokedAt        *time.Time `gorm:"column:revoked_at;type:datetime(3)"`
}

func (*UserSessionDO) TableName() string { return "user_sessions" }

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
