package db

import (
	"bytes"
	"context"
	stderrors "errors"
	"goshop/app/pkg/bizcode"
	"goshop/pkg/errcode"
	"strings"
	"time"

	"goshop/app/pkg/authz"
	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (u *users) RecordLogin(ctx context.Context, id uint64, at time.Time) error {
	if id == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}
	result := u.db.WithContext(ctx).Model(&dv1.UserDO{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("last_login_at", at.UTC())
	if result.Error != nil {
		return wrapDatabaseError(result.Error, "user database operation")
	}
	if result.RowsAffected == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}
	return nil
}

func (u *users) CreateSession(ctx context.Context, session *dv1.UserSessionDO) error {
	if session == nil || session.UserID == 0 || session.ID == "" || len(session.RefreshTokenHash) != 32 {
		return errors.NewCode(errcode.ErrValidation, "invalid session")
	}
	if strings.TrimSpace(session.PrincipalType) == "" {
		session.PrincipalType = string(authz.PrincipalCustomer)
	}
	var blocked int64
	if err := u.db.WithContext(ctx).Model(&dv1.DeviceBlacklistDO{}).Where("user_id = ? AND device_id = ?", session.UserID, session.DeviceID).Count(&blocked).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	if blocked > 0 {
		return errors.NewCode(bizcode.ErrUserAccountInactive, "device is blocked")
	}
	if err := u.db.WithContext(ctx).Create(session).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	return nil
}

func (u *users) ListUserSessions(ctx context.Context, userID uint64, offset, limit int) ([]dv1.UserSessionRecordDO, int64, error) {
	if userID == 0 {
		return nil, 0, errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	base := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).Where("user_id = ? AND principal_type = ?", userID, string(authz.PrincipalCustomer))
	var total int64
	if err := base.Count(&total).Error; err != nil {
		if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
			return nil, 0, err
		}
		return nil, 0, wrapDatabaseError(err, "user database operation")
	}
	rows := make([]dv1.UserSessionRecordDO, 0, limit)
	if err := base.Select("id, device_id, device_name, client_ip, location, created_at, last_used_at, expires_at, revoked_at").Order("last_used_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
			return nil, 0, err
		}
		return nil, 0, wrapDatabaseError(err, "user database operation")
	}
	return rows, total, nil
}

func (u *users) AddDeviceBlacklist(ctx context.Context, userID int32, deviceID string, at time.Time) error {
	deviceID = strings.TrimSpace(deviceID)
	if userID <= 0 || deviceID == "" {
		return errors.NewCode(errcode.ErrValidation, "user id and device id are required")
	}
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		entry := &dv1.DeviceBlacklistDO{UserID: userID, DeviceID: deviceID, CreatedAt: at.UTC()}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(entry).Error; err != nil {
			return wrapDatabaseError(err, "user database operation")
		}
		if err := tx.Model(&dv1.UserSessionDO{}).Where("user_id = ? AND device_id = ? AND revoked_at IS NULL", userID, deviceID).Update("revoked_at", at.UTC()).Error; err != nil {
			return wrapDatabaseError(err, "user database operation")
		}
		return nil
	})
}

func (u *users) DeleteDeviceBlacklist(ctx context.Context, userID int32, deviceID string) error {
	if userID <= 0 || strings.TrimSpace(deviceID) == "" {
		return errors.NewCode(errcode.ErrValidation, "user id and device id are required")
	}
	if err := u.db.WithContext(ctx).Delete(&dv1.DeviceBlacklistDO{}, "user_id = ? AND device_id = ?", userID, strings.TrimSpace(deviceID)).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	return nil
}

func (u *users) ListDeviceBlacklist(ctx context.Context, userID int32, offset, limit int) ([]dv1.DeviceBlacklistDO, int64, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	base := u.db.WithContext(ctx).Model(&dv1.DeviceBlacklistDO{})
	if userID > 0 {
		base = base.Where("user_id = ?", userID)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "user database operation")
	}
	items := make([]dv1.DeviceBlacklistDO, 0, limit)
	if err := base.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "user database operation")
	}
	return items, total, nil
}

func (u *users) RotateSession(ctx context.Context, sessionID string, currentHash, nextHash []byte, expiresAt, usedAt time.Time) (*dv1.UserSessionDO, error) {
	var session dv1.UserSessionDO
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", sessionID).First(&session).Error; err != nil {
			return err
		}
		if session.RevokedAt != nil || !session.ExpiresAt.After(usedAt) {
			return gorm.ErrRecordNotFound
		}
		if !bytes.Equal(session.RefreshTokenHash, currentHash) {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&session).Updates(map[string]interface{}{
			"refresh_token_hash": nextHash,
			"last_used_at":       usedAt.UTC(),
			"expires_at":         expiresAt.UTC(),
		}).Error
	})
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewCode(bizcode.ErrUserAccountInactive, "session is not active")
		}
		return nil, wrapDatabaseError(err, "user database operation")
	}
	session.RefreshTokenHash = append([]byte(nil), nextHash...)
	session.LastUsedAt = usedAt.UTC()
	session.ExpiresAt = expiresAt.UTC()
	return &session, nil
}

func (u *users) RevokeSession(ctx context.Context, userID uint64, sessionID string, at time.Time) error {
	result := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", sessionID, userID).
		Update("revoked_at", at.UTC())
	if result.Error != nil {
		return wrapDatabaseError(result.Error, "user database operation")
	}
	return nil
}

func (u *users) RevokeAllSessions(ctx context.Context, userID uint64, at time.Time) error {
	if err := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", at.UTC()).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	return nil
}

func (u *users) SessionActive(ctx context.Context, userID uint64, sessionID string, at time.Time) (bool, error) {
	var count int64
	err := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?", sessionID, userID, at.UTC()).
		Count(&count).Error
	if err != nil {
		return false, wrapDatabaseError(err, "user database operation")
	}
	return count == 1, nil
}

func (u *users) ListStaffSessions(ctx context.Context, filters dv1.StaffSessionFilters) ([]dv1.StaffSessionRecordDO, int64, error) {
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	base := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).Where("principal_type = ?", string(authz.PrincipalStaff))
	if filters.UserID > 0 {
		base = base.Where("user_id = ?", filters.UserID)
	}
	if filters.ActiveOnly {
		base = base.Where("revoked_at IS NULL AND expires_at > ?", time.Now().UTC())
	}
	if filters.CreatedAfter != nil {
		base = base.Where("created_at >= ?", filters.CreatedAfter.UTC())
	}
	if filters.CreatedBefore != nil {
		base = base.Where("created_at <= ?", filters.CreatedBefore.UTC())
	}
	if filters.LastUsedAfter != nil {
		base = base.Where("last_used_at >= ?", filters.LastUsedAfter.UTC())
	}
	if filters.LastUsedBefore != nil {
		base = base.Where("last_used_at <= ?", filters.LastUsedBefore.UTC())
	}
	if role := strings.ToLower(strings.TrimSpace(filters.Role)); role != "" {
		subQuery := u.db.WithContext(ctx).
			Table("user_roles AS ur").
			Select("DISTINCT ur.user_id").
			Joins("JOIN roles AS r ON r.id = ur.role_id").
			Where("LOWER(r.name) = ?", role)
		base = base.Where("user_id IN (?)", subQuery)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "user database operation")
	}
	rows := make([]dv1.StaffSessionRecordDO, 0, filters.Limit)
	if err := base.Select("id, user_id, principal_type, device_id, device_name, created_at, last_used_at, expires_at, revoked_at").
		Order("last_used_at DESC").
		Offset(filters.Offset).
		Limit(filters.Limit).
		Find(&rows).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "user database operation")
	}
	if len(rows) == 0 {
		return rows, total, nil
	}
	userIDs := make([]int32, 0, len(rows))
	indexByUser := map[int32][]int{}
	for i, row := range rows {
		if _, ok := indexByUser[row.UserID]; !ok {
			userIDs = append(userIDs, row.UserID)
		}
		indexByUser[row.UserID] = append(indexByUser[row.UserID], i)
	}
	type roleRow struct {
		UserID int32
		Name   string
	}
	roleRows := make([]roleRow, 0, len(userIDs))
	if err := u.db.WithContext(ctx).
		Table("user_roles AS ur").
		Select("ur.user_id AS user_id, r.name AS name").
		Joins("JOIN roles AS r ON r.id = ur.role_id").
		Where("ur.user_id IN ?", userIDs).
		Order("r.name ASC").
		Scan(&roleRows).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "user database operation")
	}
	for _, role := range roleRows {
		for _, idx := range indexByUser[role.UserID] {
			rows[idx].Roles = append(rows[idx].Roles, role.Name)
		}
	}
	return rows, total, nil
}

func (u *users) RevokeStaffSession(ctx context.Context, sessionID string, at time.Time) error {
	if sessionID == "" {
		return errors.NewCode(errcode.ErrValidation, "session id is required")
	}
	if err := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).
		Where("id = ? AND principal_type = ? AND revoked_at IS NULL", sessionID, string(authz.PrincipalStaff)).
		Update("revoked_at", at.UTC()).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	return nil
}

func (u *users) RevokeStaffUserSessions(ctx context.Context, userID uint64, at time.Time) error {
	if userID == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "user not found")
	}
	if err := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).
		Where("user_id = ? AND principal_type = ? AND revoked_at IS NULL", userID, string(authz.PrincipalStaff)).
		Update("revoked_at", at.UTC()).Error; err != nil {
		return wrapDatabaseError(err, "user database operation")
	}
	return nil
}
