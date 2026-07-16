package db

import (
	"context"
	"strings"
	"time"

	"goshop/app/pkg/authz"
	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"

	"gorm.io/gorm"
)

type users struct {
	db *gorm.DB
}

func NewUsers(db *gorm.DB) dv1.UserStore {
	return &users{db: db}
}

// GetByMobile
//
//	@Description: 根据手机号获取用户信息
//	@receiver u
//	@param ctx
//	@param mobile: 手机号
//	@return *dv1.UserDO
//	@return error
func (u *users) GetByMobile(ctx context.Context, mobile string) (*dv1.UserDO, error) {
	mobile = strings.TrimSpace(mobile)
	if mobile == "" {
		return nil, errors.WithCode(code.ErrUserNotFound, "user not found")
	}

	user := dv1.UserDO{}

	//err是gorm的error这种error我们尽量不要抛出去
	err := u.db.WithContext(ctx).Where("mobile=?", mobile).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrUserNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return &user, nil
}

func (u *users) GetByUsername(ctx context.Context, username string) (*dv1.UserDO, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.WithCode(code.ErrUserNotFound, "user not found")
	}

	user := dv1.UserDO{}
	err := u.db.WithContext(ctx).
		Where("username = ? OR mobile = ? OR email = ?", username, username, username).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrUserNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return &user, nil
}

// GetByID
//
//	@Description: 根据id获取用户信息
//	@receiver u
//	@param ctx
//	@param id: 用户id
//	@return *dv1.UserDO
//	@return error
func (u *users) GetByID(ctx context.Context, id uint64) (*dv1.UserDO, error) {
	if id == 0 {
		return nil, errors.WithCode(code.ErrUserNotFound, "user not found")
	}

	user := dv1.UserDO{}
	err := u.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrUserNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return &user, nil
}

// Create
//
//	@Description: 创建用户
//	@receiver u
//	@param ctx
//	@param user: 用户DO
//	@return error
func (u *users) Create(ctx context.Context, user *dv1.UserDO) error {
	if user == nil {
		return errors.WithCode(code2.ErrValidation, "user is required")
	}

	tx := u.db.WithContext(ctx).Create(user)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

// Update
//
//	@Description: 更新用户信息
//	@receiver u
//	@param ctx
//	@param user
//	@return error
func (u *users) Update(ctx context.Context, user *dv1.UserDO) error {
	if user == nil || user.ID <= 0 {
		return errors.WithCode(code.ErrUserNotFound, "user not found")
	}

	updates := map[string]interface{}{
		"nick_name": user.NickName,
		"gender":    user.Gender,
		"birthday":  user.Birthday,
		"email":     user.Email,
	}
	if user.Username != nil {
		updates["username"] = user.Username
	}

	tx := u.db.WithContext(ctx).Model(&dv1.UserDO{}).
		Where("id = ?", user.ID).
		Updates(updates)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrUserNotFound, "user not found")
	}
	return nil
}

func (u *users) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.WithCode(code.ErrUserNotFound, "user not found")
	}

	now := time.Now()
	tx := u.db.WithContext(ctx).Model(&dv1.UserDO{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"is_deleted":     true,
			"deleted_at":     now,
			"account_status": string(authz.AccountStatusDeleted),
		})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrUserNotFound, "user not found")
	}
	return nil
}

func newUsers(db *gorm.DB) *users {
	return &users{db: db}
}

var _ dv1.UserStore = &users{}

// List
//
//	@Description: 获取用户列表, 凡是列表页返回的时候都应该返回总共有多少个
//	@receiver u
//	@param ctx
//	@param orderBy
//	@param opts
//	@return *dv1.UserDOList
//	@return error
func (u *users) List(ctx context.Context, orderBy []string, opts metav1.ListMeta) (*dv1.UserDOList, error) {
	//实现gorm查询
	ret := &dv1.UserDOList{}

	//分页
	var limit, offset int
	if opts.PageSize == 0 {
		limit = 10
	} else {
		limit = opts.PageSize
	}

	if opts.Page > 0 {
		offset = (opts.Page - 1) * limit
	}

	countQuery := u.db.WithContext(ctx).Model(&dv1.UserDO{})
	if err := countQuery.Count(&ret.TotalCount).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	//排序
	query := u.db.WithContext(ctx).Model(&dv1.UserDO{})
	query = applyOrderBy(query, orderBy, userOrderColumns)

	d := query.Offset(offset).Limit(limit).Find(&ret.Items)
	if d.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, d.Error.Error())
	}
	return ret, nil
}
