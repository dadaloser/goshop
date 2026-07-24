package user

import (
	"context"
	"testing"
	"time"

	upbv1 "goshop/api/user/v1"
	srvv1 "goshop/app/user/srv/internal/service/v1"
)

type fakeUserSessionService struct {
	srvv1.UserSrv
	recordLogin       func(context.Context, uint64, time.Time) error
	createSession     func(context.Context, srvv1.SessionDTO) (*srvv1.SessionDTO, error)
	refreshSession    func(context.Context, string, []byte, []byte, time.Time) (*srvv1.SessionDTO, error)
	revokeSession     func(context.Context, uint64, string) error
	revokeAllSessions func(context.Context, uint64) error
	validateSession   func(context.Context, uint64, string) (bool, error)
}

func (f fakeUserSessionService) RecordLogin(ctx context.Context, userID uint64, at time.Time) error {
	return f.recordLogin(ctx, userID, at)
}

func (f fakeUserSessionService) CreateSession(ctx context.Context, session srvv1.SessionDTO) (*srvv1.SessionDTO, error) {
	return f.createSession(ctx, session)
}

func (f fakeUserSessionService) RefreshSession(ctx context.Context, sessionID string, currentHash, nextHash []byte, expiresAt time.Time) (*srvv1.SessionDTO, error) {
	return f.refreshSession(ctx, sessionID, currentHash, nextHash, expiresAt)
}

func (f fakeUserSessionService) RevokeSession(ctx context.Context, userID uint64, sessionID string) error {
	return f.revokeSession(ctx, userID, sessionID)
}

func (f fakeUserSessionService) RevokeAllSessions(ctx context.Context, userID uint64) error {
	return f.revokeAllSessions(ctx, userID)
}

func (f fakeUserSessionService) ValidateSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	return f.validateSession(ctx, userID, sessionID)
}

func TestUserServerSessionRPCs(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	service := fakeUserSessionService{
		recordLogin: func(_ context.Context, userID uint64, at time.Time) error {
			if userID != 9 || !at.Equal(now) {
				t.Fatalf("RecordLogin() got userID=%d at=%v", userID, at)
			}
			return nil
		},
		createSession: func(_ context.Context, session srvv1.SessionDTO) (*srvv1.SessionDTO, error) {
			if session.UserID != 9 || session.DeviceID != "device-1" || session.DeviceName != "iPad" || string(session.RefreshTokenHash) != "12345678901234567890123456789012" {
				t.Fatalf("CreateSession() request = %+v", session)
			}
			session.ID = "session-1"
			return &session, nil
		},
		refreshSession: func(_ context.Context, sessionID string, currentHash, nextHash []byte, expiresAt time.Time) (*srvv1.SessionDTO, error) {
			if sessionID != "session-1" || string(currentHash) != "old-token-hash-12345678901234567" || string(nextHash) != "new-token-hash-12345678901234567" || expiresAt.Unix() != now.Add(2*time.Hour).Unix() {
				t.Fatalf("RefreshSession() got sessionID=%q current=%q next=%q expiresAt=%v", sessionID, currentHash, nextHash, expiresAt)
			}
			return &srvv1.SessionDTO{ID: sessionID, UserID: 9, DeviceID: "device-1", DeviceName: "iPad", ExpiresAt: expiresAt}, nil
		},
		revokeSession: func(_ context.Context, userID uint64, sessionID string) error {
			if userID != 9 || sessionID != "session-1" {
				t.Fatalf("RevokeSession() got userID=%d sessionID=%q", userID, sessionID)
			}
			return nil
		},
		revokeAllSessions: func(_ context.Context, userID uint64) error {
			if userID != 9 {
				t.Fatalf("RevokeAllSessions() got userID=%d", userID)
			}
			return nil
		},
		validateSession: func(_ context.Context, userID uint64, sessionID string) (bool, error) {
			if userID != 9 || sessionID != "session-1" {
				t.Fatalf("ValidateSession() got userID=%d sessionID=%q", userID, sessionID)
			}
			return true, nil
		},
	}
	server := &userServer{srv: service}

	if _, err := server.RecordLogin(context.Background(), &upbv1.RecordLoginRequest{UserId: 9, LoggedInAt: uint64(now.Unix())}); err != nil {
		t.Fatalf("RecordLogin() error = %v", err)
	}

	created, err := server.CreateSession(context.Background(), &upbv1.CreateSessionRequest{
		UserId:           9,
		DeviceId:         "device-1",
		DeviceName:       "iPad",
		RefreshTokenHash: []byte("12345678901234567890123456789012"),
		ExpiresAt:        uint64(now.Add(time.Hour).Unix()),
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if created.GetId() != "session-1" || created.GetDeviceName() != "iPad" {
		t.Fatalf("CreateSession() response = %+v", created)
	}

	refreshed, err := server.RefreshSession(context.Background(), &upbv1.RefreshSessionRequest{
		SessionId:        "session-1",
		CurrentTokenHash: []byte("old-token-hash-12345678901234567"),
		NextTokenHash:    []byte("new-token-hash-12345678901234567"),
		ExpiresAt:        uint64(now.Add(2 * time.Hour).Unix()),
	})
	if err != nil {
		t.Fatalf("RefreshSession() error = %v", err)
	}
	if refreshed.GetId() != "session-1" || refreshed.GetUserId() != 9 {
		t.Fatalf("RefreshSession() response = %+v", refreshed)
	}

	if _, err := server.RevokeSession(context.Background(), &upbv1.RevokeSessionRequest{UserId: 9, SessionId: "session-1"}); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if _, err := server.RevokeAllSessions(context.Background(), &upbv1.IdRequest{Id: 9}); err != nil {
		t.Fatalf("RevokeAllSessions() error = %v", err)
	}

	validation, err := server.ValidateSession(context.Background(), &upbv1.ValidateSessionRequest{UserId: 9, SessionId: "session-1"})
	if err != nil {
		t.Fatalf("ValidateSession() error = %v", err)
	}
	if !validation.GetActive() {
		t.Fatalf("ValidateSession() response = %+v, want active=true", validation)
	}
}
