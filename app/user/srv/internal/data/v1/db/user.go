package db

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"goshop/app/pkg/accountdeletion"
	"goshop/app/pkg/bizcode"
	"slices"
	"strings"
	"time"

	"goshop/app/pkg/authz"
	dv1 "goshop/app/user/srv/internal/data/v1"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errcode"
	"goshop/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/google/uuid"
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
		return nil, errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}

	user := dv1.UserDO{}

	//err是gorm的error这种error我们尽量不要抛出去
	err := u.db.WithContext(ctx).Where("mobile=?", mobile).First(&user).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewCode(bizcode.ErrUserNotFound, err.Error())
		}
		return nil, wrapDatabaseError(err, "user database operation")
	}
	return &user, nil
}

func (u *users) GetByUsername(ctx context.Context, username string) (*dv1.UserDO, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}

	user := dv1.UserDO{}
	err := u.db.WithContext(ctx).
		Where("username = ? OR mobile = ? OR email = ?", username, username, username).
		First(&user).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewCode(bizcode.ErrUserNotFound, err.Error())
		}
		return nil, wrapDatabaseError(err, "user database operation")
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
		return nil, errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}

	user := dv1.UserDO{}
	err := u.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewCode(bizcode.ErrUserNotFound, err.Error())
		}
		return nil, wrapDatabaseError(err, "user database operation")
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
		return nil, wrapDatabaseError(err, "user database operation")
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
		return nil, errors.NewCode(errcode.ErrValidation, "staff role name is required")
	}

	role := dv1.RoleDO{
		Name:        roleName,
		Description: description,
	}
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, wrapDatabaseError(tx.Error, "user database operation")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Create(&role).Error; err != nil {
		tx.Rollback()
		return nil, wrapDatabaseError(err, "user database operation")
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
		return nil, wrapDatabaseError(err, "user database operation")
	}

	role.Permissions = append([]string(nil), permissions...)
	role.Domains = append([]string(nil), domains...)
	return &role, nil
}

func (u *users) UpdateRole(ctx context.Context, roleName, description string, permissions, domains []string) (*dv1.RoleDO, error) {
	roleName = strings.ToLower(strings.TrimSpace(roleName))
	if roleName == "" {
		return nil, errors.NewCode(errcode.ErrValidation, "staff role name is required")
	}

	var role dv1.RoleDO
	if err := u.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewCode(bizcode.ErrUserNotFound, "staff role not found")
		}
		return nil, wrapDatabaseError(err, "user database operation")
	}

	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, wrapDatabaseError(tx.Error, "user database operation")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Model(&dv1.RoleDO{}).Where("id = ?", role.ID).Update("description", description).Error; err != nil {
		tx.Rollback()
		return nil, wrapDatabaseError(err, "user database operation")
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
		return nil, wrapDatabaseError(err, "user database operation")
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
		return errors.NewCode(errcode.ErrValidation, "staff role name is required")
	}
	if authz.IsValidStaffRole(roleName) {
		return errors.NewCode(errcode.ErrValidation, "built-in staff roles cannot be deleted")
	}

	var role dv1.RoleDO
	if err := u.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewCode(bizcode.ErrUserNotFound, "staff role not found")
		}
		return wrapDatabaseError(err, "user database operation")
	}

	var bindingCount int64
	if err := u.db.WithContext(ctx).Model(&dv1.UserRoleDO{}).Where("role_id = ?", role.ID).Count(&bindingCount).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	if bindingCount > 0 {
		return errors.NewCode(errcode.ErrValidation, "staff role is still assigned")
	}

	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return wrapDatabaseError(tx.Error, "user database operation")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Where("role_id = ?", role.ID).Delete(&dv1.RolePermissionDO{}).Error; err != nil {
		tx.Rollback()
		return wrapDatabaseError(err, "user database operation")
	}
	if err := tx.Where("role_id = ?", role.ID).Delete(&dv1.RoleDomainDO{}).Error; err != nil {
		tx.Rollback()
		return wrapDatabaseError(err, "user database operation")
	}
	if err := tx.Delete(&dv1.RoleDO{}, role.ID).Error; err != nil {
		tx.Rollback()
		return wrapDatabaseError(err, "user database operation")
	}
	if err := tx.Commit().Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
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
		return nil, wrapDatabaseError(tx.Error, "user database operation")
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
		return nil, wrapDatabaseError(err, "user database operation")
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
		return errors.NewCode(errcode.ErrValidation, "user is required")
	}

	tx := u.db.WithContext(ctx).Create(user)
	if tx.Error != nil {
		return wrapDatabaseError(tx.Error, "user database operation")
	}
	return nil
}

func (u *users) CreateStaff(ctx context.Context, user *dv1.UserDO, roleNames []string, actor *dv1.AuditActor) (*dv1.UserAuthDO, error) {
	if user == nil {
		return nil, errors.NewCode(errcode.ErrValidation, "user is required")
	}

	normalizedRoles := normalizeRoleNames(roleNames)
	if len(normalizedRoles) == 0 {
		return nil, errors.NewCode(errcode.ErrValidation, "staff roles are required")
	}

	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, wrapDatabaseError(tx.Error, "user database operation")
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
		return nil, wrapDatabaseError(err, "user database operation")
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
		return nil, wrapDatabaseError(err, "user database operation")
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
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
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
		return wrapDatabaseError(tx.Error, "user database operation")
	}
	if tx.RowsAffected == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}
	return nil
}

func (u *users) UpdateStatus(ctx context.Context, id uint64, status string, actor *dv1.AuditActor) error {
	if id == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
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
		return wrapDatabaseError(tx.Error, "user database operation")
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
		return wrapDatabaseError(result.Error, "user database operation")
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
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
		return wrapDatabaseError(err, "user database operation")
	}
	return nil
}

func (u *users) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
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
		return wrapDatabaseError(tx.Error, "user database operation")
	}
	if tx.RowsAffected == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}
	return nil
}

// RequestAccountDeletion starts a reversible account-deletion workflow.
// The actual deletion decision is made asynchronously by downstream services.
// Keeping the account row intact is essential: a rejected request must be able
// to restore the account without recreating identities or foreign keys.
func (u *users) RequestAccountDeletion(ctx context.Context, id uint64, at time.Time) error {
	if id == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}

	eventID := uuid.NewString()
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user dv1.UserDO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", id).
			First(&user).Error; err != nil {
			return err
		}

		if user.Status == string(authz.AccountStatusDeletionPending) {
			return nil // duplicate requests are idempotent
		}
		if authz.NormalizeAccountStatus(user.Status) != authz.AccountStatusActive {
			return errors.NewCode(bizcode.ErrUserAccountInactive, "account is not active")
		}

		if err := tx.Model(&dv1.UserDO{}).Where("id = ?", id).
			Update("account_status", string(authz.AccountStatusDeletionPending)).Error; err != nil {
			return err
		}
		if err := tx.Model(&dv1.UserSessionDO{}).
			Where("user_id = ? AND revoked_at IS NULL", id).
			Update("revoked_at", at.UTC()).Error; err != nil {
			return err
		}
		payload, err := json.Marshal(accountdeletion.Requested{EventID: eventID, UserID: id, RequestedAt: at.UTC()})
		if err != nil {
			return err
		}
		if err := tx.Create(&dv1.AccountDeletionOutboxEventDO{
			ID:          eventID,
			EventType:   accountdeletion.SubjectRequested,
			UserID:      user.ID,
			Payload:     payload,
			Status:      "PENDING",
			AvailableAt: at.UTC(),
			CreatedAt:   at.UTC(),
			UpdatedAt:   at.UTC(),
		}).Error; err != nil {
			return err
		}
		return u.appendAuditLogTx(tx, &dv1.UserAuditLogDO{
			UserID:             user.ID,
			ActorUserID:        user.ID,
			ActorPrincipalType: string(authz.PrincipalCustomer),
			Action:             dv1.UserAuditActionAccountDeletionRequested,
			FromStatus:         user.Status,
			ToStatus:           string(authz.AccountStatusDeletionPending),
		})
	})
	if err == nil {
		return nil
	}
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}
	// Keep business errors generated inside the transaction intact.
	if errors.IsCode(err, bizcode.ErrUserAccountInactive) {
		return err
	}
	return wrapDatabaseError(err, "user database operation")
}

func (u *users) ListAuditLogs(ctx context.Context, userID uint64, filters dv1.UserAuditLogFilters, opts metav1.ListMeta) (*dv1.UserAuditLogDOList, error) {
	if userID == 0 {
		return nil, errors.NewCode(bizcode.ErrUserNotFound, "user not found")
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
		return nil, wrapDatabaseError(err, "user database operation")
	}
	if err := query.Order("add_time DESC, id DESC").Offset(offset).Limit(limit).Find(&ret.Items).Error; err != nil {
		return nil, wrapDatabaseError(err, "user database operation")
	}
	return ret, nil
}

func (u *users) CreateAdminAuditLog(ctx context.Context, logEntry *dv1.AdminAuditLogDO) error {
	if logEntry == nil {
		return errors.NewCode(errcode.ErrValidation, "admin audit log is required")
	}
	if strings.TrimSpace(logEntry.Action) == "" {
		return errors.NewCode(errcode.ErrValidation, "admin audit action is required")
	}
	if strings.TrimSpace(logEntry.ActorPrincipalType) == "" {
		logEntry.ActorPrincipalType = string(authz.PrincipalInternalService)
	}
	if err := u.db.WithContext(ctx).Create(logEntry).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	return nil
}

func (u *users) ListAdminAuditLogs(ctx context.Context, filters dv1.AdminAuditLogFilters, opts metav1.ListMeta) (*dv1.AdminAuditLogDOList, error) {
	ret := &dv1.AdminAuditLogDOList{}
	limit := opts.PageSize
	if limit <= 0 {
		limit = 10
	}
	offset := 0
	if opts.Page > 0 {
		offset = (opts.Page - 1) * limit
	}

	query := u.db.WithContext(ctx).Model(&dv1.AdminAuditLogDO{})
	if filters.TargetUserID > 0 {
		query = query.Where("target_user_id = ?", filters.TargetUserID)
	}
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
		return nil, wrapDatabaseError(err, "user database operation")
	}
	if err := query.Order("add_time DESC, id DESC").Offset(offset).Limit(limit).Find(&ret.Items).Error; err != nil {
		return nil, wrapDatabaseError(err, "user database operation")
	}
	return ret, nil
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
		return nil, wrapDatabaseError(err, "user database operation")
	}

	//排序
	query := u.db.WithContext(ctx).Model(&dv1.UserDO{})
	query = applyOrderBy(query, orderBy, userOrderColumns)

	d := query.Offset(offset).Limit(limit).Find(&ret.Items)
	if d.Error != nil {
		return nil, wrapDatabaseError(d.Error, "user database operation")
	}
	return ret, nil
}

func (u *users) buildAuthUser(ctx context.Context, user *dv1.UserDO) (*dv1.UserAuthDO, error) {
	if user == nil {
		return nil, errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}

	roles, err := u.listStaffRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	permissions, err := u.listPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	var scopes []dv1.UserResourceScopeDO
	if err = u.db.WithContext(ctx).Where("user_id = ?", user.ID).Find(&scopes).Error; err != nil {
		return nil, wrapDatabaseError(err, "user database operation")
	}
	domainSet := make(map[string]struct{}, len(scopes))
	storeSet := make(map[string]struct{}, len(scopes))
	teamSet := make(map[string]struct{}, len(scopes))
	resourceScopes := make([]dv1.UserResourceScopeDO, 0, len(scopes))
	for _, scope := range scopes {
		if scope.Domain != "" {
			domainSet[scope.Domain] = struct{}{}
		}
		if scope.StoreID != "" {
			storeSet[scope.StoreID] = struct{}{}
		}
		if scope.TeamID != "" {
			teamSet[scope.TeamID] = struct{}{}
		}
		resourceScopes = append(resourceScopes, scope)
	}
	domains := make([]string, 0, len(domainSet))
	for value := range domainSet {
		domains = append(domains, value)
	}
	stores := make([]string, 0, len(storeSet))
	for value := range storeSet {
		stores = append(stores, value)
	}
	teams := make([]string, 0, len(teamSet))
	for value := range teamSet {
		teams = append(teams, value)
	}

	return &dv1.UserAuthDO{
		UserDO:          *user,
		StaffRoles:      roles,
		Permissions:     permissions,
		ResourceDomains: domains, ResourceStores: stores, ResourceTeams: teams, ResourceScopes: resourceScopes,
	}, nil
}

func (u *users) ReplaceResourceScopes(ctx context.Context, userID uint64, scopes []dv1.UserResourceScopeDO) error {
	if userID == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&dv1.UserResourceScopeDO{}).Error; err != nil {
			return err
		}
		if len(scopes) > 0 {
			if err := tx.Create(&scopes).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return wrapDatabaseError(err, "replace user resource scopes")
	}
	return nil
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
		return nil, wrapDatabaseError(err, "user database operation")
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
		return nil, wrapDatabaseError(err, "user database operation")
	}
	return permissions, nil
}

func (u *users) listRolePermissions(ctx context.Context, roleID uint64) ([]string, error) {
	if roleID == 0 {
		return nil, errors.NewCode(errcode.ErrValidation, "staff role not found")
	}

	var permissions []string
	err := u.db.WithContext(ctx).
		Model(&dv1.RolePermissionDO{}).
		Where("role_id = ?", roleID).
		Order("permission ASC").
		Pluck("permission", &permissions).Error
	if err != nil {
		return nil, wrapDatabaseError(err, "user database operation")
	}
	return permissions, nil
}

func (u *users) listRoleDomains(ctx context.Context, roleID uint64) ([]string, error) {
	if roleID == 0 {
		return nil, errors.NewCode(errcode.ErrValidation, "staff role not found")
	}

	var domains []string
	err := u.db.WithContext(ctx).
		Model(&dv1.RoleDomainDO{}).
		Where("role_id = ?", roleID).
		Order("domain ASC").
		Pluck("domain", &domains).Error
	if err != nil {
		return nil, wrapDatabaseError(err, "user database operation")
	}
	return domains, nil
}

func (u *users) loadRoles(tx *gorm.DB, normalizedRoles []string) ([]dv1.RoleDO, error) {
	if len(normalizedRoles) == 0 {
		return nil, nil
	}

	var roles []dv1.RoleDO
	if err := tx.Where("name IN ?", normalizedRoles).Order("name ASC").Find(&roles).Error; err != nil {
		return nil, wrapDatabaseError(err, "user database operation")
	}
	if len(roles) != len(normalizedRoles) {
		return nil, errors.NewCode(errcode.ErrValidation, "staff roles contain unknown values")
	}
	loadedNames := make([]string, 0, len(roles))
	for _, role := range roles {
		loadedNames = append(loadedNames, role.Name)
	}
	slices.Sort(loadedNames)
	if !slices.Equal(loadedNames, normalizedRoles) {
		return nil, errors.NewCode(errcode.ErrValidation, "staff roles contain unknown values")
	}
	return roles, nil
}

func (u *users) replaceUserRolesTx(tx *gorm.DB, userID int32, roles []dv1.RoleDO) error {
	if err := tx.Where("user_id = ?", userID).Delete(&dv1.UserRoleDO{}).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	for _, role := range roles {
		binding := dv1.UserRoleDO{
			UserID: userID,
			RoleID: role.ID,
		}
		if err := tx.Create(&binding).Error; err != nil {
			return wrapDatabaseError(err, "user database operation")
		}
	}
	return nil
}

func (u *users) replaceRolePermissionsTx(tx *gorm.DB, roleID uint64, permissions []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&dv1.RolePermissionDO{}).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	for _, permission := range permissions {
		record := dv1.RolePermissionDO{
			RoleID:     roleID,
			Permission: permission,
		}
		if err := tx.Create(&record).Error; err != nil {
			return wrapDatabaseError(err, "user database operation")
		}
	}
	return nil
}

func (u *users) replaceRoleDomainsTx(tx *gorm.DB, roleID uint64, domains []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&dv1.RoleDomainDO{}).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	for _, domain := range domains {
		record := dv1.RoleDomainDO{
			RoleID: roleID,
			Domain: domain,
		}
		if err := tx.Create(&record).Error; err != nil {
			return wrapDatabaseError(err, "user database operation")
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
		return wrapDatabaseError(err, "user database operation")
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
