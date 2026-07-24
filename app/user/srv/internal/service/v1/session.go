package v1

import (
	"context"
	"time"

	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"github.com/google/uuid"
)

type SessionDTO struct {
	ID               string
	UserID           int32
	RefreshTokenHash []byte
	DeviceID         string
	DeviceName       string
	ExpiresAt        time.Time
}

type sessionStore interface {
	RecordLogin(ctx context.Context, id uint64, at time.Time) error
	CreateSession(ctx context.Context, session *dv1.UserSessionDO) error
	RotateSession(ctx context.Context, sessionID string, currentHash, nextHash []byte, expiresAt, usedAt time.Time) (*dv1.UserSessionDO, error)
	RevokeSession(ctx context.Context, userID uint64, sessionID string, at time.Time) error
	RevokeAllSessions(ctx context.Context, userID uint64, at time.Time) error
	SessionActive(ctx context.Context, userID uint64, sessionID string, at time.Time) (bool, error)
}

func (u *userService) RecordLogin(ctx context.Context, userID uint64, at time.Time) error {
	store, err := u.sessions()
	if err != nil {
		return err
	}
	return store.RecordLogin(ctx, userID, at)
}

func (u *userService) CreateSession(ctx context.Context, session SessionDTO) (*SessionDTO, error) {
	now := time.Now().UTC()
	model := &dv1.UserSessionDO{
		ID: uuid.NewString(), UserID: session.UserID,
		RefreshTokenHash: append([]byte(nil), session.RefreshTokenHash...),
		DeviceID:         session.DeviceID, DeviceName: session.DeviceName,
		CreatedAt: now, LastUsedAt: now, ExpiresAt: session.ExpiresAt.UTC(),
	}
	store, err := u.sessions()
	if err != nil {
		return nil, err
	}
	if err := store.CreateSession(ctx, model); err != nil {
		return nil, err
	}
	session.ID = model.ID
	return &session, nil
}

func (u *userService) RefreshSession(ctx context.Context, sessionID string, currentHash, nextHash []byte, expiresAt time.Time) (*SessionDTO, error) {
	store, err := u.sessions()
	if err != nil {
		return nil, err
	}
	model, err := store.RotateSession(ctx, sessionID, currentHash, nextHash, expiresAt, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &SessionDTO{ID: model.ID, UserID: model.UserID, DeviceID: model.DeviceID, DeviceName: model.DeviceName, ExpiresAt: model.ExpiresAt}, nil
}

func (u *userService) RevokeSession(ctx context.Context, userID uint64, sessionID string) error {
	store, err := u.sessions()
	if err != nil {
		return err
	}
	return store.RevokeSession(ctx, userID, sessionID, time.Now().UTC())
}

func (u *userService) RevokeAllSessions(ctx context.Context, userID uint64) error {
	store, err := u.sessions()
	if err != nil {
		return err
	}
	return store.RevokeAllSessions(ctx, userID, time.Now().UTC())
}

func (u *userService) ValidateSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	store, err := u.sessions()
	if err != nil {
		return false, err
	}
	return store.SessionActive(ctx, userID, sessionID, time.Now().UTC())
}

func (u *userService) sessions() (sessionStore, error) {
	store, ok := u.userStore.(sessionStore)
	if !ok {
		return nil, errors.WithCode(code2.ErrDatabase, "session store is not configured")
	}
	return store, nil
}

func copySessionTokenHash(hash []byte) []byte {
	if len(hash) == 0 {
		return nil
	}
	return append([]byte(nil), hash...)
}

func requireSessionID(sessionID string) error {
	if sessionID == "" {
		return errors.WithCode(code.ErrUserAccountInactive, "session is not active")
	}
	return nil
}
