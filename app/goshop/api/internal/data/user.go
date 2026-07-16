package data

import (
	"context"

	"goshop/pkg/common/time"
)

type User struct {
	ID       uint64    `json:"id"`
	Username string    `json:"username"`
	Mobile   string    `json:"mobile"`
	Email    string    `json:"email"`
	NickName string    `json:"nick_name"`
	Birthday time.Time `gorm:"type:datetime"`
	Gender   string    `json:"gender"`
	Role     int32     `json:"role"`
	Status   string    `json:"status"`
	PassWord string    `json:"password"`
}

type UserList struct {
	TotalCount int64   `json:"totalCount,omitempty"`
	Items      []*User `json:"items"`
}

type UserData interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, userID uint64) error
	Get(ctx context.Context, userID uint64) (User, error)
	GetByMobile(ctx context.Context, mobile string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	CheckPassWord(ctx context.Context, password, encryptedPwd string) error
}
