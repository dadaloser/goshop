package db

import (
	"bytes"
	"context"
	stderrors "errors"
	"time"

	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (u *users) RecordLogin(ctx context.Context, id uint64, at time.Time) error {
	if id == 0 {
		return errors.WithCode(code.ErrUserNotFound, "user not found")
	}
	result := u.db.WithContext(ctx).Model(&dv1.UserDO{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("last_login_at", at.UTC())
	if result.Error != nil {
		return errors.WithCode(code2.ErrDatabase, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.WithCode(code.ErrUserNotFound, "user not found")
	}
	return nil
}

func (u *users) CreateSession(ctx context.Context, session *dv1.UserSessionDO) error {
	if session == nil || session.UserID == 0 || session.ID == "" || len(session.RefreshTokenHash) != 32 {
		return errors.WithCode(code2.ErrValidation, "invalid session")
	}
	if err := u.db.WithContext(ctx).Create(session).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
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
			return nil, errors.WithCode(code.ErrUserAccountInactive, "session is not active")
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
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
		return errors.WithCode(code2.ErrDatabase, result.Error.Error())
	}
	return nil
}

func (u *users) RevokeAllSessions(ctx context.Context, userID uint64, at time.Time) error {
	if err := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", at.UTC()).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (u *users) SessionActive(ctx context.Context, userID uint64, sessionID string, at time.Time) (bool, error) {
	var count int64
	err := u.db.WithContext(ctx).Model(&dv1.UserSessionDO{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?", sessionID, userID, at.UTC()).
		Count(&count).Error
	if err != nil {
		return false, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return count == 1, nil
}
