package v1

import (
	"context"
	"goshop/app/pkg/bizcode"
	"testing"
	"time"

	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"
)

type fakeSessionUserStore struct {
	dv1.UserStore
	recordLogin            func(context.Context, uint64, time.Time) error
	createSession          func(context.Context, *dv1.UserSessionDO) error
	rotateSession          func(context.Context, string, []byte, []byte, time.Time, time.Time) (*dv1.UserSessionDO, error)
	revokeSession          func(context.Context, uint64, string, time.Time) error
	revokeAll              func(context.Context, uint64, time.Time) error
	sessionActive          func(context.Context, uint64, string, time.Time) (bool, error)
	listStaffSessions      func(context.Context, dv1.StaffSessionFilters) ([]dv1.StaffSessionRecordDO, int64, error)
	revokeStaffSession     func(context.Context, string, time.Time) error
	revokeStaffUserSession func(context.Context, uint64, time.Time) error
}

func (f fakeSessionUserStore) RecordLogin(ctx context.Context, userID uint64, at time.Time) error {
	return f.recordLogin(ctx, userID, at)
}

func (f fakeSessionUserStore) CreateSession(ctx context.Context, session *dv1.UserSessionDO) error {
	return f.createSession(ctx, session)
}

func (f fakeSessionUserStore) RotateSession(ctx context.Context, sessionID string, currentHash, nextHash []byte, expiresAt, usedAt time.Time) (*dv1.UserSessionDO, error) {
	return f.rotateSession(ctx, sessionID, currentHash, nextHash, expiresAt, usedAt)
}

func (f fakeSessionUserStore) RevokeSession(ctx context.Context, userID uint64, sessionID string, at time.Time) error {
	return f.revokeSession(ctx, userID, sessionID, at)
}

func (f fakeSessionUserStore) RevokeAllSessions(ctx context.Context, userID uint64, at time.Time) error {
	return f.revokeAll(ctx, userID, at)
}

func (f fakeSessionUserStore) SessionActive(ctx context.Context, userID uint64, sessionID string, at time.Time) (bool, error) {
	return f.sessionActive(ctx, userID, sessionID, at)
}

func (f fakeSessionUserStore) ListUserSessions(context.Context, uint64, int, int) ([]dv1.UserSessionRecordDO, int64, error) {
	return nil, 0, nil
}
func (f fakeSessionUserStore) AddDeviceBlacklist(context.Context, int32, string, time.Time) error {
	return nil
}
func (f fakeSessionUserStore) DeleteDeviceBlacklist(context.Context, int32, string) error { return nil }
func (f fakeSessionUserStore) ListDeviceBlacklist(context.Context, int, int) ([]dv1.DeviceBlacklistDO, int64, error) {
	return nil, 0, nil
}

func (f fakeSessionUserStore) ListStaffSessions(ctx context.Context, filters dv1.StaffSessionFilters) ([]dv1.StaffSessionRecordDO, int64, error) {
	if f.listStaffSessions == nil {
		return nil, 0, nil
	}
	return f.listStaffSessions(ctx, filters)
}

func (f fakeSessionUserStore) RevokeStaffSession(ctx context.Context, sessionID string, at time.Time) error {
	if f.revokeStaffSession == nil {
		return nil
	}
	return f.revokeStaffSession(ctx, sessionID, at)
}

func (f fakeSessionUserStore) RevokeStaffUserSessions(ctx context.Context, userID uint64, at time.Time) error {
	if f.revokeStaffUserSession == nil {
		return nil
	}
	return f.revokeStaffUserSession(ctx, userID, at)
}

func TestUserServiceSessionLifecycle(t *testing.T) {
	now := time.Now().UTC()
	hash := []byte("12345678901234567890123456789012")
	var created *dv1.UserSessionDO
	store := fakeSessionUserStore{
		recordLogin: func(context.Context, uint64, time.Time) error { return nil },
		createSession: func(_ context.Context, session *dv1.UserSessionDO) error {
			created = session
			return nil
		},
		rotateSession: func(_ context.Context, sessionID string, currentHash, nextHash []byte, expiresAt, _ time.Time) (*dv1.UserSessionDO, error) {
			if sessionID != "session-1" {
				t.Fatalf("sessionID = %q, want session-1", sessionID)
			}
			if string(currentHash) != string(hash) || string(nextHash) != "abcdefghijklmnopqrstuvwxzy012345" {
				t.Fatalf("rotate hashes = %q -> %q", currentHash, nextHash)
			}
			return &dv1.UserSessionDO{ID: sessionID, UserID: 9, DeviceID: "device-1", DeviceName: "Pixel", ExpiresAt: expiresAt}, nil
		},
		revokeSession: func(context.Context, uint64, string, time.Time) error { return nil },
		revokeAll:     func(context.Context, uint64, time.Time) error { return nil },
		sessionActive: func(context.Context, uint64, string, time.Time) (bool, error) { return true, nil },
	}

	svc := &userService{userStore: store}

	if err := svc.RecordLogin(context.Background(), 9, now); err != nil {
		t.Fatalf("RecordLogin() error = %v", err)
	}

	createdDTO, err := svc.CreateSession(context.Background(), SessionDTO{
		UserID:           9,
		RefreshTokenHash: hash,
		DeviceID:         "device-1",
		DeviceName:       "Pixel",
		ExpiresAt:        now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if created == nil || created.ID == "" || created.UserID != 9 || created.DeviceID != "device-1" || created.DeviceName != "Pixel" {
		t.Fatalf("created session = %+v", created)
	}
	if createdDTO == nil || createdDTO.ID == "" {
		t.Fatalf("CreateSession() response = %+v", createdDTO)
	}

	refreshed, err := svc.RefreshSession(context.Background(), "session-1", hash, []byte("abcdefghijklmnopqrstuvwxzy012345"), now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("RefreshSession() error = %v", err)
	}
	if refreshed == nil || refreshed.ID != "session-1" || refreshed.UserID != 9 || refreshed.DeviceName != "Pixel" {
		t.Fatalf("RefreshSession() response = %+v", refreshed)
	}

	if err := svc.RevokeSession(context.Background(), 9, "session-1"); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if err := svc.RevokeAllSessions(context.Background(), 9); err != nil {
		t.Fatalf("RevokeAllSessions() error = %v", err)
	}
	active, err := svc.ValidateSession(context.Background(), 9, "session-1")
	if err != nil {
		t.Fatalf("ValidateSession() error = %v", err)
	}
	if !active {
		t.Fatal("ValidateSession() active=false, want true")
	}
}

func TestUserServiceSessionMethodsRequireSessionStore(t *testing.T) {
	svc := &userService{}
	if _, err := svc.CreateSession(context.Background(), SessionDTO{}); !errors.IsCode(err, errcode.ErrDatabase) {
		t.Fatalf("CreateSession() error = %v, want code %d", err, errcode.ErrDatabase)
	}
	if _, err := svc.RefreshSession(context.Background(), "session-1", nil, nil, time.Now()); !errors.IsCode(err, errcode.ErrDatabase) {
		t.Fatalf("RefreshSession() error = %v, want code %d", err, errcode.ErrDatabase)
	}
	if err := svc.RevokeSession(context.Background(), 1, "session-1"); !errors.IsCode(err, errcode.ErrDatabase) {
		t.Fatalf("RevokeSession() error = %v, want code %d", err, errcode.ErrDatabase)
	}
	if err := svc.RevokeAllSessions(context.Background(), 1); !errors.IsCode(err, errcode.ErrDatabase) {
		t.Fatalf("RevokeAllSessions() error = %v, want code %d", err, errcode.ErrDatabase)
	}
	if _, err := svc.ValidateSession(context.Background(), 1, "session-1"); !errors.IsCode(err, errcode.ErrDatabase) {
		t.Fatalf("ValidateSession() error = %v, want code %d", err, errcode.ErrDatabase)
	}
	if err := svc.RecordLogin(context.Background(), 1, time.Now()); !errors.IsCode(err, errcode.ErrDatabase) {
		t.Fatalf("RecordLogin() error = %v, want code %d", err, errcode.ErrDatabase)
	}
}

func TestRequireSessionID(t *testing.T) {
	if err := requireSessionID(""); !errors.IsCode(err, bizcode.ErrUserAccountInactive) {
		t.Fatalf("requireSessionID(\"\") error = %v", err)
	}
	if err := requireSessionID("session-1"); err != nil {
		t.Fatalf("requireSessionID(session-1) error = %v", err)
	}
}

func TestUserServiceStaffSessionOperations(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	revokedAt := now.Add(-5 * time.Minute)
	var listed bool
	var revokedSessionID string
	var revokedUserID uint64

	store := fakeSessionUserStore{
		listStaffSessions: func(_ context.Context, filters dv1.StaffSessionFilters) ([]dv1.StaffSessionRecordDO, int64, error) {
			listed = true
			if filters.UserID != 7 || filters.Role != "ops_delegate" || !filters.ActiveOnly {
				t.Fatalf("ListStaffSessions() filters = %+v", filters)
			}
			if filters.Offset != 0 || filters.Limit != 20 {
				t.Fatalf("ListStaffSessions() pagination = offset %d limit %d, want 0/20", filters.Offset, filters.Limit)
			}
			if filters.CreatedAfter == nil || filters.LastUsedBefore == nil {
				t.Fatalf("ListStaffSessions() missing time filters: %+v", filters)
			}
			return []dv1.StaffSessionRecordDO{
				{
					ID:            "staff-session-1",
					UserID:        7,
					PrincipalType: "staff",
					DeviceID:      "device-1",
					DeviceName:    "MacBook",
					CreatedAt:     now.Add(-time.Hour),
					LastUsedAt:    now,
					ExpiresAt:     now.Add(time.Hour),
					Roles:         []string{"ops_delegate"},
				},
				{
					ID:            "staff-session-2",
					UserID:        7,
					PrincipalType: "staff",
					DeviceID:      "device-2",
					DeviceName:    "iPad",
					CreatedAt:     now.Add(-2 * time.Hour),
					LastUsedAt:    now.Add(-30 * time.Minute),
					ExpiresAt:     now.Add(30 * time.Minute),
					RevokedAt:     &revokedAt,
					Roles:         []string{"ops_delegate", "review"},
				},
			}, 2, nil
		},
		revokeStaffSession: func(_ context.Context, sessionID string, at time.Time) error {
			revokedSessionID = sessionID
			if sessionID != "staff-session-1" || at.IsZero() {
				t.Fatalf("RevokeStaffSession() got sessionID=%q at=%v", sessionID, at)
			}
			return nil
		},
		revokeStaffUserSession: func(_ context.Context, userID uint64, at time.Time) error {
			revokedUserID = userID
			if userID != 7 || at.IsZero() {
				t.Fatalf("RevokeStaffUserSessions() got userID=%d at=%v", userID, at)
			}
			return nil
		},
	}

	svc := &userService{userStore: store}
	createdAfter := now.Add(-24 * time.Hour)
	lastUsedBefore := now.Add(time.Minute)
	result, err := svc.ListStaffSessions(context.Background(), StaffSessionFilterDTO{
		UserID:         7,
		Role:           "ops_delegate",
		ActiveOnly:     true,
		CreatedAfter:   &createdAfter,
		LastUsedBefore: &lastUsedBefore,
		Page:           0,
		PageSize:       200,
	})
	if err != nil {
		t.Fatalf("ListStaffSessions() error = %v", err)
	}
	if !listed {
		t.Fatal("ListStaffSessions() store was not called")
	}
	if result.TotalCount != 2 || len(result.Items) != 2 {
		t.Fatalf("ListStaffSessions() result = %+v, want total=2 len=2", result)
	}
	if got := result.Items[1].Roles[1]; got != "review" {
		t.Fatalf("ListStaffSessions() copied roles = %#v, want review preserved", result.Items[1].Roles)
	}

	if err := svc.RevokeStaffSession(context.Background(), "staff-session-1"); err != nil {
		t.Fatalf("RevokeStaffSession() error = %v", err)
	}
	if revokedSessionID != "staff-session-1" {
		t.Fatalf("RevokeStaffSession() sessionID = %q, want staff-session-1", revokedSessionID)
	}

	if err := svc.RevokeStaffUserSessions(context.Background(), 7); err != nil {
		t.Fatalf("RevokeStaffUserSessions() error = %v", err)
	}
	if revokedUserID != 7 {
		t.Fatalf("RevokeStaffUserSessions() userID = %d, want 7", revokedUserID)
	}
}
