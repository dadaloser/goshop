package v1

import (
	"context"
	"goshop/app/pkg/bizcode"
	"time"

	"goshop/app/pkg/authz"
	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"

	"github.com/google/uuid"
)

type SessionDTO struct {
	ID               string
	UserID           int32
	PrincipalType    string
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
	ListStaffSessions(ctx context.Context, filters dv1.StaffSessionFilters) ([]dv1.StaffSessionRecordDO, int64, error)
	RevokeStaffSession(ctx context.Context, sessionID string, at time.Time) error
	RevokeStaffUserSessions(ctx context.Context, userID uint64, at time.Time) error
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
	principalType := session.PrincipalType
	if principalType == "" {
		principalType = string(authz.PrincipalCustomer)
	}
	model := &dv1.UserSessionDO{
		ID: uuid.NewString(), UserID: session.UserID, PrincipalType: principalType,
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
	session.PrincipalType = principalType
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
	return &SessionDTO{ID: model.ID, UserID: model.UserID, PrincipalType: model.PrincipalType, DeviceID: model.DeviceID, DeviceName: model.DeviceName, ExpiresAt: model.ExpiresAt}, nil
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

func (u *userService) ListStaffSessions(ctx context.Context, filters StaffSessionFilterDTO) (*StaffSessionDTOList, error) {
	store, err := u.sessions()
	if err != nil {
		return nil, err
	}
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 || filters.PageSize > 100 {
		filters.PageSize = 20
	}
	items, total, err := store.ListStaffSessions(ctx, dv1.StaffSessionFilters{
		UserID:         filters.UserID,
		Role:           filters.Role,
		ActiveOnly:     filters.ActiveOnly,
		CreatedAfter:   filters.CreatedAfter,
		CreatedBefore:  filters.CreatedBefore,
		LastUsedAfter:  filters.LastUsedAfter,
		LastUsedBefore: filters.LastUsedBefore,
		Offset:         (filters.Page - 1) * filters.PageSize,
		Limit:          filters.PageSize,
	})
	if err != nil {
		return nil, err
	}
	result := &StaffSessionDTOList{TotalCount: total, Items: make([]StaffSessionDTO, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, StaffSessionDTO{
			ID:            item.ID,
			UserID:        item.UserID,
			PrincipalType: item.PrincipalType,
			DeviceID:      item.DeviceID,
			DeviceName:    item.DeviceName,
			CreatedAt:     item.CreatedAt,
			LastUsedAt:    item.LastUsedAt,
			ExpiresAt:     item.ExpiresAt,
			RevokedAt:     item.RevokedAt,
			Roles:         append([]string(nil), item.Roles...),
		})
	}
	return result, nil
}

func (u *userService) RevokeStaffSession(ctx context.Context, sessionID string) error {
	store, err := u.sessions()
	if err != nil {
		return err
	}
	return store.RevokeStaffSession(ctx, sessionID, time.Now().UTC())
}

func (u *userService) RevokeStaffUserSessions(ctx context.Context, userID uint64) error {
	store, err := u.sessions()
	if err != nil {
		return err
	}
	return store.RevokeStaffUserSessions(ctx, userID, time.Now().UTC())
}

func (u *userService) sessions() (sessionStore, error) {
	store, ok := u.userStore.(sessionStore)
	if !ok {
		return nil, errors.NewCode(errcode.ErrDatabase, "session store is not configured")
	}
	return store, nil
}

func requireSessionID(sessionID string) error {
	if sessionID == "" {
		return errors.NewCode(bizcode.ErrUserAccountInactive, "session is not active")
	}
	return nil
}
