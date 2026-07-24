package v1

import (
	"context"
	"testing"
	"time"

	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

type fakeSessionUserStore struct {
	dv1.UserStore
	recordLogin   func(context.Context, uint64, time.Time) error
	createSession func(context.Context, *dv1.UserSessionDO) error
	rotateSession func(context.Context, string, []byte, []byte, time.Time, time.Time) (*dv1.UserSessionDO, error)
	revokeSession func(context.Context, uint64, string, time.Time) error
	revokeAll     func(context.Context, uint64, time.Time) error
	sessionActive func(context.Context, uint64, string, time.Time) (bool, error)
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
	if _, err := svc.CreateSession(context.Background(), SessionDTO{}); !errors.IsCode(err, code2.ErrDatabase) {
		t.Fatalf("CreateSession() error = %v, want code %d", err, code2.ErrDatabase)
	}
	if _, err := svc.RefreshSession(context.Background(), "session-1", nil, nil, time.Now()); !errors.IsCode(err, code2.ErrDatabase) {
		t.Fatalf("RefreshSession() error = %v, want code %d", err, code2.ErrDatabase)
	}
	if err := svc.RevokeSession(context.Background(), 1, "session-1"); !errors.IsCode(err, code2.ErrDatabase) {
		t.Fatalf("RevokeSession() error = %v, want code %d", err, code2.ErrDatabase)
	}
	if err := svc.RevokeAllSessions(context.Background(), 1); !errors.IsCode(err, code2.ErrDatabase) {
		t.Fatalf("RevokeAllSessions() error = %v, want code %d", err, code2.ErrDatabase)
	}
	if _, err := svc.ValidateSession(context.Background(), 1, "session-1"); !errors.IsCode(err, code2.ErrDatabase) {
		t.Fatalf("ValidateSession() error = %v, want code %d", err, code2.ErrDatabase)
	}
	if err := svc.RecordLogin(context.Background(), 1, time.Now()); !errors.IsCode(err, code2.ErrDatabase) {
		t.Fatalf("RecordLogin() error = %v, want code %d", err, code2.ErrDatabase)
	}
}

func TestRequireSessionID(t *testing.T) {
	if err := requireSessionID(""); !errors.IsCode(err, code.ErrUserAccountInactive) {
		t.Fatalf("requireSessionID(\"\") error = %v", err)
	}
	if err := requireSessionID("session-1"); err != nil {
		t.Fatalf("requireSessionID(session-1) error = %v", err)
	}
}
