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

func (u *users) Create(ctx context.Context, user *dv1.UserDO) error {
	return errors.New("mock users Create not implemented")
}

func (u *users) Update(ctx context.Context, user *dv1.UserDO) error {
	return errors.New("mock users Update not implemented")
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
