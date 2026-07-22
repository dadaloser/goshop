package db

import (
	"context"
	"fmt"
	"slices"
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

func (u *users) GetAuthByUsername(ctx context.Context, username string) (*dv1.UserAuthDO, error) {
	user, err := u.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return u.buildAuthUser(ctx, user)
}

func (u *users) GetAuthByID(ctx context.Context, id uint64) (*dv1.UserAuthDO, error) {
	user, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return u.buildAuthUser(ctx, user)
}

func (u *users) ListRoles(ctx context.Context) ([]dv1.RoleDO, error) {
	var roles []dv1.RoleDO
	if err := u.db.WithContext(ctx).Order("name ASC").Find(&roles).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	definitions := make(map[string]authz.RoleDefinition)
	for _, definition := range authz.BuiltinRoleDefinitions() {
		definitions[string(definition.Name)] = definition
	}
	for i := range roles {
		permissions, err := u.listRolePermissions(ctx, roles[i].ID)
		if err != nil {
			return nil, err
		}
		domains, err := u.listRoleDomains(ctx, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = permissions
		roles[i].Domains = domains
		if definition, ok := definitions[roles[i].Name]; ok {
			roles[i].Builtin = true
			if len(roles[i].Domains) == 0 {
				roles[i].Domains = make([]string, 0, len(definition.Domains))
				for _, domain := range definition.Domains {
					roles[i].Domains = append(roles[i].Domains, string(domain))
				}
			}
		}
	}
	return roles, nil
}

func (u *users) CreateRole(ctx context.Context, roleName, description string, permissions, domains []string) (*dv1.RoleDO, error) {
	roleName = strings.ToLower(strings.TrimSpace(roleName))
	if roleName == "" {
		return nil, errors.WithCode(code2.ErrValidation, "staff role name is required")
	}

	role := dv1.RoleDO{
		Name:        roleName,
		Description: description,
	}
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Create(&role).Error; err != nil {
		tx.Rollback()
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if err := u.replaceRolePermissionsTx(tx, role.ID, permissions); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := u.replaceRoleDomainsTx(tx, role.ID, domains); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	role.Permissions = append([]string(nil), permissions...)
	role.Domains = append([]string(nil), domains...)
	return &role, nil
}

func (u *users) UpdateRole(ctx context.Context, roleName, description string, permissions, domains []string) (*dv1.RoleDO, error) {
	roleName = strings.ToLower(strings.TrimSpace(roleName))
	if roleName == "" {
		return nil, errors.WithCode(code2.ErrValidation, "staff role name is required")
	}

	var role dv1.RoleDO
	if err := u.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrUserNotFound, "staff role not found")
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Model(&dv1.RoleDO{}).Where("id = ?", role.ID).Update("description", description).Error; err != nil {
		tx.Rollback()
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if err := u.replaceRolePermissionsTx(tx, role.ID, permissions); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := u.replaceRoleDomainsTx(tx, role.ID, domains); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	role.Description = description
	role.Permissions = append([]string(nil), permissions...)
	role.Domains = append([]string(nil), domains...)
	role.Builtin = authz.IsValidStaffRole(role.Name)
	return &role, nil
}

func (u *users) DeleteRole(ctx context.Context, roleName string) error {
	roleName = strings.ToLower(strings.TrimSpace(roleName))
	if roleName == "" {
		return errors.WithCode(code2.ErrValidation, "staff role name is required")
	}
	if authz.IsValidStaffRole(roleName) {
		return errors.WithCode(code2.ErrValidation, "built-in staff roles cannot be deleted")
	}

	var role dv1.RoleDO
	if err := u.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.WithCode(code.ErrUserNotFound, "staff role not found")
		}
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}

	var bindingCount int64
	if err := u.db.WithContext(ctx).Model(&dv1.UserRoleDO{}).Where("role_id = ?", role.ID).Count(&bindingCount).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if bindingCount > 0 {
		return errors.WithCode(code2.ErrValidation, "staff role is still assigned")
	}

	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Where("role_id = ?", role.ID).Delete(&dv1.RolePermissionDO{}).Error; err != nil {
		tx.Rollback()
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if err := tx.Where("role_id = ?", role.ID).Delete(&dv1.RoleDomainDO{}).Error; err != nil {
		tx.Rollback()
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if err := tx.Delete(&dv1.RoleDO{}, role.ID).Error; err != nil {
		tx.Rollback()
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if err := tx.Commit().Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (u *users) ReplaceUserRoles(ctx context.Context, userID uint64, roleNames []string, actor *dv1.AuditActor) (*dv1.UserAuthDO, error) {
	user, err := u.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	previousRoles, err := u.listStaffRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	normalizedRoles := normalizeRoleNames(roleNames)
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	roles, err := u.loadRoles(tx, normalizedRoles)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err = u.replaceUserRolesTx(tx, user.ID, roles); err != nil {
		tx.Rollback()
		return nil, err
	}
	if !slices.Equal(previousRoles, normalizedRoles) {
		if err = u.appendAuditLogTx(tx, &dv1.UserAuditLogDO{
			UserID:             user.ID,
			ActorUserID:        actorUserID(actor),
			ActorPrincipalType: actorPrincipalType(actor),
			Action:             dv1.UserAuditActionRolesReplaced,
			Detail:             buildRolesAuditDetail(previousRoles, normalizedRoles),
		}); err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if err = tx.Commit().Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	return u.buildAuthUser(ctx, user)
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

func (u *users) CreateStaff(ctx context.Context, user *dv1.UserDO, roleNames []string, actor *dv1.AuditActor) (*dv1.UserAuthDO, error) {
	if user == nil {
		return nil, errors.WithCode(code2.ErrValidation, "user is required")
	}

	normalizedRoles := normalizeRoleNames(roleNames)
	if len(normalizedRoles) == 0 {
		return nil, errors.WithCode(code2.ErrValidation, "staff roles are required")
	}

	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	roles, err := u.loadRoles(tx, normalizedRoles)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if err = tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if err = u.replaceUserRolesTx(tx, user.ID, roles); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err = u.appendAuditLogTx(tx, &dv1.UserAuditLogDO{
		UserID:             user.ID,
		ActorUserID:        actorUserID(actor),
		ActorPrincipalType: actorPrincipalType(actor),
		Action:             dv1.UserAuditActionStaffCreated,
		ToStatus:           user.Status,
		Detail:             buildRolesAuditDetail(nil, normalizedRoles),
	}); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err = tx.Commit().Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	return u.buildAuthUser(ctx, user)
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

func (u *users) UpdateStatus(ctx context.Context, id uint64, status string, actor *dv1.AuditActor) error {
	if id == 0 {
		return errors.WithCode(code.ErrUserNotFound, "user not found")
	}

	user, err := u.GetByID(ctx, id)
	if err != nil {
		return err
	}
	previousStatus := user.Status
	if previousStatus == status {
		return nil
	}

	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	result := tx.Model(&dv1.UserDO{}).Where("id = ?", id).Update("account_status", status)
	if result.Error != nil {
		tx.Rollback()
		return errors.WithCode(code2.ErrDatabase, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return errors.WithCode(code.ErrUserNotFound, "user not found")
	}
	if err = u.appendAuditLogTx(tx, &dv1.UserAuditLogDO{
		UserID:             user.ID,
		ActorUserID:        actorUserID(actor),
		ActorPrincipalType: actorPrincipalType(actor),
		Action:             dv1.UserAuditActionStatusUpdated,
		FromStatus:         previousStatus,
		ToStatus:           status,
	}); err != nil {
		tx.Rollback()
		return err
	}
	if err = tx.Commit().Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
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

func (u *users) ListAuditLogs(ctx context.Context, userID uint64, filters dv1.UserAuditLogFilters, opts metav1.ListMeta) (*dv1.UserAuditLogDOList, error) {
	if userID == 0 {
		return nil, errors.WithCode(code.ErrUserNotFound, "user not found")
	}

	ret := &dv1.UserAuditLogDOList{}
	limit := opts.PageSize
	if limit <= 0 {
		limit = 10
	}
	offset := 0
	if opts.Page > 0 {
		offset = (opts.Page - 1) * limit
	}

	query := u.db.WithContext(ctx).Model(&dv1.UserAuditLogDO{}).Where("user_id = ?", userID)
	if action := strings.TrimSpace(filters.Action); action != "" {
		query = query.Where("action = ?", action)
	}
	if filters.ActorUserID > 0 {
		query = query.Where("actor_user_id = ?", filters.ActorUserID)
	}
	if principalType := strings.TrimSpace(filters.ActorPrincipalType); principalType != "" {
		query = query.Where("actor_principal_type = ?", principalType)
	}
	if filters.CreatedAfter != nil {
		query = query.Where("add_time >= ?", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		query = query.Where("add_time <= ?", *filters.CreatedBefore)
	}
	if err := query.Count(&ret.TotalCount).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if err := query.Order("add_time DESC, id DESC").Offset(offset).Limit(limit).Find(&ret.Items).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return ret, nil
}

func (u *users) CreateAdminAuditLog(ctx context.Context, logEntry *dv1.AdminAuditLogDO) error {
	if logEntry == nil {
		return errors.WithCode(code2.ErrValidation, "admin audit log is required")
	}
	if strings.TrimSpace(logEntry.Action) == "" {
		return errors.WithCode(code2.ErrValidation, "admin audit action is required")
	}
	if strings.TrimSpace(logEntry.ActorPrincipalType) == "" {
		logEntry.ActorPrincipalType = string(authz.PrincipalInternalService)
	}
	if err := u.db.WithContext(ctx).Create(logEntry).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
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

func (u *users) buildAuthUser(ctx context.Context, user *dv1.UserDO) (*dv1.UserAuthDO, error) {
	if user == nil {
		return nil, errors.WithCode(code.ErrUserNotFound, "user not found")
	}

	roles, err := u.listStaffRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	permissions, err := u.listPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &dv1.UserAuthDO{
		UserDO:      *user,
		StaffRoles:  roles,
		Permissions: permissions,
	}, nil
}

func (u *users) listStaffRoles(ctx context.Context, userID int32) ([]string, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id %d", userID)
	}

	var roles []string
	err := u.db.WithContext(ctx).
		Table((&dv1.RoleDO{}).TableName()).
		Distinct((&dv1.RoleDO{}).TableName()+".name").
		Joins("JOIN "+(&dv1.UserRoleDO{}).TableName()+" ON "+(&dv1.UserRoleDO{}).TableName()+".role_id = "+(&dv1.RoleDO{}).TableName()+".id").
		Where((&dv1.UserRoleDO{}).TableName()+".user_id = ?", userID).
		Order((&dv1.RoleDO{}).TableName()+".name ASC").
		Pluck((&dv1.RoleDO{}).TableName()+".name", &roles).Error
	if err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return roles, nil
}

func (u *users) listPermissions(ctx context.Context, userID int32) ([]string, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id %d", userID)
	}

	var permissions []string
	err := u.db.WithContext(ctx).
		Table((&dv1.RolePermissionDO{}).TableName()).
		Distinct((&dv1.RolePermissionDO{}).TableName()+".permission").
		Joins("JOIN "+(&dv1.RoleDO{}).TableName()+" ON "+(&dv1.RoleDO{}).TableName()+".id = "+(&dv1.RolePermissionDO{}).TableName()+".role_id").
		Joins("JOIN "+(&dv1.UserRoleDO{}).TableName()+" ON "+(&dv1.UserRoleDO{}).TableName()+".role_id = "+(&dv1.RoleDO{}).TableName()+".id").
		Where((&dv1.UserRoleDO{}).TableName()+".user_id = ?", userID).
		Order((&dv1.RolePermissionDO{}).TableName()+".permission ASC").
		Pluck((&dv1.RolePermissionDO{}).TableName()+".permission", &permissions).Error
	if err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return permissions, nil
}

func (u *users) listRolePermissions(ctx context.Context, roleID uint64) ([]string, error) {
	if roleID == 0 {
		return nil, errors.WithCode(code2.ErrValidation, "staff role not found")
	}

	var permissions []string
	err := u.db.WithContext(ctx).
		Model(&dv1.RolePermissionDO{}).
		Where("role_id = ?", roleID).
		Order("permission ASC").
		Pluck("permission", &permissions).Error
	if err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return permissions, nil
}

func (u *users) listRoleDomains(ctx context.Context, roleID uint64) ([]string, error) {
	if roleID == 0 {
		return nil, errors.WithCode(code2.ErrValidation, "staff role not found")
	}

	var domains []string
	err := u.db.WithContext(ctx).
		Model(&dv1.RoleDomainDO{}).
		Where("role_id = ?", roleID).
		Order("domain ASC").
		Pluck("domain", &domains).Error
	if err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return domains, nil
}

func (u *users) loadRoles(tx *gorm.DB, normalizedRoles []string) ([]dv1.RoleDO, error) {
	if len(normalizedRoles) == 0 {
		return nil, nil
	}

	var roles []dv1.RoleDO
	if err := tx.Where("name IN ?", normalizedRoles).Order("name ASC").Find(&roles).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if len(roles) != len(normalizedRoles) {
		return nil, errors.WithCode(code2.ErrValidation, "staff roles contain unknown values")
	}
	loadedNames := make([]string, 0, len(roles))
	for _, role := range roles {
		loadedNames = append(loadedNames, role.Name)
	}
	slices.Sort(loadedNames)
	if !slices.Equal(loadedNames, normalizedRoles) {
		return nil, errors.WithCode(code2.ErrValidation, "staff roles contain unknown values")
	}
	return roles, nil
}

func (u *users) replaceUserRolesTx(tx *gorm.DB, userID int32, roles []dv1.RoleDO) error {
	if err := tx.Where("user_id = ?", userID).Delete(&dv1.UserRoleDO{}).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	for _, role := range roles {
		binding := dv1.UserRoleDO{
			UserID: userID,
			RoleID: role.ID,
		}
		if err := tx.Create(&binding).Error; err != nil {
			return errors.WithCode(code2.ErrDatabase, err.Error())
		}
	}
	return nil
}

func (u *users) replaceRolePermissionsTx(tx *gorm.DB, roleID uint64, permissions []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&dv1.RolePermissionDO{}).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	for _, permission := range permissions {
		record := dv1.RolePermissionDO{
			RoleID:     roleID,
			Permission: permission,
		}
		if err := tx.Create(&record).Error; err != nil {
			return errors.WithCode(code2.ErrDatabase, err.Error())
		}
	}
	return nil
}

func (u *users) replaceRoleDomainsTx(tx *gorm.DB, roleID uint64, domains []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&dv1.RoleDomainDO{}).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	for _, domain := range domains {
		record := dv1.RoleDomainDO{
			RoleID: roleID,
			Domain: domain,
		}
		if err := tx.Create(&record).Error; err != nil {
			return errors.WithCode(code2.ErrDatabase, err.Error())
		}
	}
	return nil
}

func (u *users) appendAuditLogTx(tx *gorm.DB, logEntry *dv1.UserAuditLogDO) error {
	if logEntry == nil || logEntry.UserID <= 0 {
		return nil
	}
	if strings.TrimSpace(logEntry.ActorPrincipalType) == "" {
		logEntry.ActorPrincipalType = string(authz.PrincipalInternalService)
	}
	if err := tx.Create(logEntry).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func actorUserID(actor *dv1.AuditActor) int32 {
	if actor == nil {
		return 0
	}
	return actor.UserID
}

func actorPrincipalType(actor *dv1.AuditActor) string {
	if actor == nil {
		return ""
	}
	return strings.TrimSpace(actor.PrincipalType)
}

func buildRolesAuditDetail(previousRoles, nextRoles []string) string {
	return fmt.Sprintf("roles:%s->%s", strings.Join(previousRoles, ","), strings.Join(nextRoles, ","))
}

func normalizeRoleNames(roleNames []string) []string {
	normalized := make([]string, 0, len(roleNames))
	seen := make(map[string]struct{}, len(roleNames))
	for _, roleName := range roleNames {
		value := strings.ToLower(strings.TrimSpace(roleName))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	slices.Sort(normalized)
	return normalized
}

func builtinRoleDefinitionByName(name string) (authz.RoleDefinition, bool) {
	for _, definition := range authz.BuiltinRoleDefinitions() {
		if string(definition.Name) == strings.ToLower(strings.TrimSpace(name)) {
			return definition, true
		}
	}
	return authz.RoleDefinition{}, false
}
