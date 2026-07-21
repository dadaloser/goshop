package mock

import (
	"context"
	"errors"
	dv1 "goshop/app/user/srv/internal/data/v1"

	metav1 "goshop/pkg/common/meta/v1"
)

type users struct {
	users []*dv1.UserDO
}

func (u *users) GetByMobile(ctx context.Context, mobile string) (*dv1.UserDO, error) {
	return nil, errors.New("mock users GetByMobile not implemented")
}

func (u *users) GetByUsername(ctx context.Context, username string) (*dv1.UserDO, error) {
	return nil, errors.New("mock users GetByUsername not implemented")
}

func (u *users) GetByID(ctx context.Context, id uint64) (*dv1.UserDO, error) {
	return nil, errors.New("mock users GetByID not implemented")
}

func (u *users) GetAuthByUsername(ctx context.Context, username string) (*dv1.UserAuthDO, error) {
	return nil, errors.New("mock users GetAuthByUsername not implemented")
}

func (u *users) GetAuthByID(ctx context.Context, id uint64) (*dv1.UserAuthDO, error) {
	return nil, errors.New("mock users GetAuthByID not implemented")
}

func (u *users) ListRoles(ctx context.Context) ([]dv1.RoleDO, error) {
	return nil, errors.New("mock users ListRoles not implemented")
}

func (u *users) CreateRole(ctx context.Context, roleName, description string, permissions, domains []string) (*dv1.RoleDO, error) {
	return nil, errors.New("mock users CreateRole not implemented")
}

func (u *users) UpdateRole(ctx context.Context, roleName, description string, permissions, domains []string) (*dv1.RoleDO, error) {
	return nil, errors.New("mock users UpdateRole not implemented")
}

func (u *users) DeleteRole(ctx context.Context, roleName string) error {
	return errors.New("mock users DeleteRole not implemented")
}

func (u *users) ReplaceUserRoles(ctx context.Context, userID uint64, roleNames []string, actor *dv1.AuditActor) (*dv1.UserAuthDO, error) {
	return nil, errors.New("mock users ReplaceUserRoles not implemented")
}

func (u *users) ListAuditLogs(ctx context.Context, userID uint64, filters dv1.UserAuditLogFilters, opts metav1.ListMeta) (*dv1.UserAuditLogDOList, error) {
	return nil, errors.New("mock users ListAuditLogs not implemented")
}

func (u *users) Create(ctx context.Context, user *dv1.UserDO) error {
	return errors.New("mock users Create not implemented")
}

func (u *users) CreateStaff(ctx context.Context, user *dv1.UserDO, roleNames []string, actor *dv1.AuditActor) (*dv1.UserAuthDO, error) {
	return nil, errors.New("mock users CreateStaff not implemented")
}

func (u *users) Update(ctx context.Context, user *dv1.UserDO) error {
	return errors.New("mock users Update not implemented")
}

func (u *users) UpdateStatus(ctx context.Context, id uint64, status string, actor *dv1.AuditActor) error {
	return errors.New("mock users UpdateStatus not implemented")
}

func (u *users) Delete(ctx context.Context, id uint64) error {
	return errors.New("mock users Delete not implemented")
}

func NewUsers() *users {
	return &users{}
}

func (u *users) List(ctx context.Context, fields []string, opts metav1.ListMeta) (*dv1.UserDOList, error) {
	var users []*dv1.UserDO
	return &dv1.UserDOList{
		TotalCount: 1,
		Items:      users,
	}, nil
}
