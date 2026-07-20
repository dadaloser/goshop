package v1

type RoleDO struct {
	ID          uint64 `gorm:"primarykey"`
	Name        string `gorm:"uniqueIndex:idx_role_name;type:varchar(64);not null"`
	Description string `gorm:"type:varchar(255)"`
}

func (r *RoleDO) TableName() string {
	return "roles"
}

type UserRoleDO struct {
	ID     uint64 `gorm:"primarykey"`
	UserID int32  `gorm:"index:idx_user_role_user;not null"`
	RoleID uint64 `gorm:"index:idx_user_role_role;not null"`
}

func (r *UserRoleDO) TableName() string {
	return "user_roles"
}

type RolePermissionDO struct {
	ID         uint64 `gorm:"primarykey"`
	RoleID     uint64 `gorm:"index:idx_role_permission_role;not null"`
	Permission string `gorm:"type:varchar(128);not null"`
}

func (r *RolePermissionDO) TableName() string {
	return "role_permissions"
}

type UserAuthDO struct {
	UserDO
	StaffRoles  []string
	Permissions []string
}
