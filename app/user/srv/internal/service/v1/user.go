package v1

import (
	"context"
	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/common/auth"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"
)

type UserDTO struct {
	//正常要重新写,不是直接引用
	dv1.UserDO
}

type UserSrv interface {
	List(ctx context.Context, orderBy []string, opts metav1.ListMeta) (*UserDTOList, error)
	Create(ctx context.Context, user *UserDTO) error
	Update(ctx context.Context, user *UserDTO) error
	GetByID(ctx context.Context, ID uint64) (*UserDTO, error)
	GetByMobile(ctx context.Context, mobile string) (*UserDTO, error)
	GetByUsername(ctx context.Context, username string) (*UserDTO, error)
}

type userService struct {
	userStore dv1.UserStore
}

func (u *userService) Create(ctx context.Context, user *UserDTO) error {
	//先判断用户是否存在
	if _, err := u.userStore.GetByMobile(ctx, user.Mobile); err == nil {
		return errors.WithCode(code.ErrUserAlreadyExists, "用户已经存在")
	} else if !errors.IsCode(err, code.ErrUserNotFound) {
		return err
	}

	if user.Email != "" {
		if _, err := u.userStore.GetByUsername(ctx, user.Email); err == nil {
			return errors.WithCode(code.ErrUserAlreadyExists, "邮箱已经存在")
		} else if !errors.IsCode(err, code.ErrUserNotFound) {
			return err
		}
	}

	encryptedPassword, err := auth.Encrypt(user.Password)
	if err != nil {
		return errors.WithCode(code2.ErrEncrypt, "加密密码失败")
	}
	user.Password = encryptedPassword
	return u.userStore.Create(ctx, &user.UserDO)
}

func (u *userService) Update(ctx context.Context, user *UserDTO) error {
	//先查询用户是否存在
	_, err := u.userStore.GetByID(ctx, uint64(user.ID))
	if err != nil {
		return err
	}

	return u.userStore.Update(ctx, &user.UserDO)
}

func (u *userService) GetByID(ctx context.Context, ID uint64) (*UserDTO, error) {
	userDO, err := u.userStore.GetByID(ctx, ID)
	if err != nil {
		return nil, err
	}

	return &UserDTO{*userDO}, nil
}

func (u *userService) GetByMobile(ctx context.Context, mobile string) (*UserDTO, error) {
	userDO, err := u.userStore.GetByMobile(ctx, mobile)
	if err != nil {
		return nil, err
	}

	return &UserDTO{*userDO}, nil
}

func (u *userService) GetByUsername(ctx context.Context, username string) (*UserDTO, error) {
	userDO, err := u.userStore.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return &UserDTO{*userDO}, nil
}

func NewUserService(us dv1.UserStore) UserSrv {
	return &userService{
		userStore: us,
	}
}

var _ UserSrv = &userService{}

type UserDTOList struct {
	TotalCount int64      `json:"totalCount,omitempty"` //总数
	Items      []*UserDTO `json:"data"`                 //数据
}

func (u *userService) List(ctx context.Context, orderBy []string, opts metav1.ListMeta) (*UserDTOList, error) {
	//这里是业务逻辑1
	/*
		1. data层的接口必须先写好
		2. 我期望测试的时候每次测试底层的data层的数据按照我期望的返回
			1. 先手动去插入一些数据
			2. 去删除一些数据
		3. 如果data层的方法有bug， 坑爹， 我们的代码想要具备好的可测试性
	*/

	doList, err := u.userStore.List(ctx, orderBy, opts)
	if err != nil {
		return nil, err
	}

	//业务逻辑2
	//代码不方便写单元测试用例
	var userDTOList UserDTOList
	for _, value := range doList.Items {
		projectDTO := UserDTO{*value}
		userDTOList.Items = append(userDTOList.Items, &projectDTO)
	}

	//业务逻辑3
	return &userDTOList, nil
}
