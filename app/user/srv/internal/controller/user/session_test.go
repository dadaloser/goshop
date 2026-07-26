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
	recordLogin              func(context.Context, uint64, time.Time) error
	createSession            func(context.Context, srvv1.SessionDTO) (*srvv1.SessionDTO, error)
	refreshSession           func(context.Context, string, []byte, []byte, time.Time) (*srvv1.SessionDTO, error)
	revokeSession            func(context.Context, uint64, string) error
	revokeAllSessions        func(context.Context, uint64) error
	validateSession          func(context.Context, uint64, string) (bool, error)
	listStaffSessions        func(context.Context, srvv1.StaffSessionFilterDTO) (*srvv1.StaffSessionDTOList, error)
	revokeStaffSession       func(context.Context, string) error
	revokeStaffUserSessions  func(context.Context, uint64) error
	createBreakGlassApproval func(context.Context, int32, string, string, time.Time) (*srvv1.BreakGlassApprovalDTO, error)
	approveBreakGlass        func(context.Context, string, int32, string) (*srvv1.BreakGlassApprovalDTO, error)
	consumeBreakGlass        func(context.Context, string, int32, string) (*srvv1.BreakGlassApprovalDTO, error)
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

func (f fakeUserSessionService) ListStaffSessions(ctx context.Context, filters srvv1.StaffSessionFilterDTO) (*srvv1.StaffSessionDTOList, error) {
	return f.listStaffSessions(ctx, filters)
}

func (f fakeUserSessionService) RevokeStaffSession(ctx context.Context, sessionID string) error {
	return f.revokeStaffSession(ctx, sessionID)
}

func (f fakeUserSessionService) RevokeStaffUserSessions(ctx context.Context, userID uint64) error {
	return f.revokeStaffUserSessions(ctx, userID)
}

func (f fakeUserSessionService) CreateBreakGlassApproval(ctx context.Context, requesterUserID int32, reason, requestID string, expiresAt time.Time) (*srvv1.BreakGlassApprovalDTO, error) {
	return f.createBreakGlassApproval(ctx, requesterUserID, reason, requestID, expiresAt)
}

func (f fakeUserSessionService) ApproveBreakGlassApproval(ctx context.Context, approvalID string, approverUserID int32, requestID string) (*srvv1.BreakGlassApprovalDTO, error) {
	return f.approveBreakGlass(ctx, approvalID, approverUserID, requestID)
}

func (f fakeUserSessionService) ConsumeBreakGlassApproval(ctx context.Context, approvalID string, requesterUserID int32, requestID string) (*srvv1.BreakGlassApprovalDTO, error) {
	return f.consumeBreakGlass(ctx, approvalID, requesterUserID, requestID)
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

func TestUserServerStaffSessionAndBreakGlassRPCs(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	revokedAt := now.Add(-time.Minute)
	approvedAt := now.Add(2 * time.Minute)
	usedAt := now.Add(3 * time.Minute)
	service := fakeUserSessionService{
		listStaffSessions: func(_ context.Context, filters srvv1.StaffSessionFilterDTO) (*srvv1.StaffSessionDTOList, error) {
			if filters.UserID != 7 || filters.Role != "ops_delegate" || !filters.ActiveOnly || filters.Page != 2 || filters.PageSize != 5 {
				t.Fatalf("ListStaffSessions() filters = %+v", filters)
			}
			if filters.CreatedAfter == nil || filters.CreatedAfter.Unix() != now.Unix() {
				t.Fatalf("ListStaffSessions() CreatedAfter = %v, want %v", filters.CreatedAfter, now)
			}
			if filters.LastUsedBefore == nil || filters.LastUsedBefore.Unix() != now.Add(time.Hour).Unix() {
				t.Fatalf("ListStaffSessions() LastUsedBefore = %v, want %v", filters.LastUsedBefore, now.Add(time.Hour))
			}
			return &srvv1.StaffSessionDTOList{
				TotalCount: 2,
				Items: []srvv1.StaffSessionDTO{
					{
						ID:            "staff-session-1",
						UserID:        7,
						PrincipalType: "staff",
						DeviceID:      "device-1",
						DeviceName:    "MacBook",
						CreatedAt:     now,
						LastUsedAt:    now.Add(time.Minute),
						ExpiresAt:     now.Add(time.Hour),
						Roles:         []string{"ops_delegate"},
					},
					{
						ID:            "staff-session-2",
						UserID:        7,
						PrincipalType: "staff",
						DeviceID:      "device-2",
						DeviceName:    "iPad",
						CreatedAt:     now.Add(-time.Hour),
						LastUsedAt:    now,
						ExpiresAt:     now.Add(time.Hour),
						RevokedAt:     &revokedAt,
						Roles:         []string{"ops_delegate", "review"},
					},
				},
			}, nil
		},
		revokeStaffSession: func(_ context.Context, sessionID string) error {
			if sessionID != "staff-session-1" {
				t.Fatalf("RevokeStaffSession() sessionID = %q, want staff-session-1", sessionID)
			}
			return nil
		},
		revokeStaffUserSessions: func(_ context.Context, userID uint64) error {
			if userID != 7 {
				t.Fatalf("RevokeStaffUserSessions() userID = %d, want 7", userID)
			}
			return nil
		},
		createBreakGlassApproval: func(_ context.Context, requesterUserID int32, reason, requestID string, expiresAt time.Time) (*srvv1.BreakGlassApprovalDTO, error) {
			if requesterUserID != 7 || reason != "emergency access" || requestID != "req-1" || expiresAt.Unix() != now.Add(4*time.Hour).Unix() {
				t.Fatalf("CreateBreakGlassApproval() got requester=%d reason=%q requestID=%q expiresAt=%v", requesterUserID, reason, requestID, expiresAt)
			}
			return &srvv1.BreakGlassApprovalDTO{
				ID:              "approval-1",
				RequesterUserID: requesterUserID,
				Status:          "pending",
				Reason:          reason,
				RequestID:       requestID,
				CreatedAt:       now,
				ExpiresAt:       expiresAt,
			}, nil
		},
		approveBreakGlass: func(_ context.Context, approvalID string, approverUserID int32, requestID string) (*srvv1.BreakGlassApprovalDTO, error) {
			if approvalID != "approval-1" || approverUserID != 8 || requestID != "req-2" {
				t.Fatalf("ApproveBreakGlassApproval() got approvalID=%q approver=%d requestID=%q", approvalID, approverUserID, requestID)
			}
			return &srvv1.BreakGlassApprovalDTO{
				ID:              approvalID,
				RequesterUserID: 7,
				ApproverUserID:  approverUserID,
				Status:          "approved",
				Reason:          "emergency access",
				RequestID:       requestID,
				CreatedAt:       now,
				ApprovedAt:      &approvedAt,
				ExpiresAt:       now.Add(4 * time.Hour),
			}, nil
		},
		consumeBreakGlass: func(_ context.Context, approvalID string, requesterUserID int32, requestID string) (*srvv1.BreakGlassApprovalDTO, error) {
			if approvalID != "approval-1" || requesterUserID != 7 || requestID != "req-3" {
				t.Fatalf("ConsumeBreakGlassApproval() got approvalID=%q requester=%d requestID=%q", approvalID, requesterUserID, requestID)
			}
			return &srvv1.BreakGlassApprovalDTO{
				ID:              approvalID,
				RequesterUserID: requesterUserID,
				ApproverUserID:  8,
				Status:          "used",
				Reason:          "emergency access",
				RequestID:       requestID,
				CreatedAt:       now,
				ApprovedAt:      &approvedAt,
				ExpiresAt:       now.Add(4 * time.Hour),
				UsedAt:          &usedAt,
			}, nil
		},
	}
	server := &userServer{srv: service}

	listResp, err := server.ListStaffSessions(context.Background(), &upbv1.ListStaffSessionsRequest{
		UserId:         7,
		Role:           "ops_delegate",
		ActiveOnly:     true,
		CreatedAfter:   uint64(now.Unix()),
		LastUsedBefore: uint64(now.Add(time.Hour).Unix()),
		Pn:             2,
		PSize:          5,
	})
	if err != nil {
		t.Fatalf("ListStaffSessions() error = %v", err)
	}
	if listResp.GetTotal() != 2 || len(listResp.GetItems()) != 2 {
		t.Fatalf("ListStaffSessions() response = %+v, want total=2 len=2", listResp)
	}
	if !listResp.GetItems()[0].GetActive() || listResp.GetItems()[1].GetActive() {
		t.Fatalf("ListStaffSessions() active flags = %+v", listResp.GetItems())
	}
	if listResp.GetItems()[1].GetRevokedAt() != uint64(revokedAt.Unix()) {
		t.Fatalf("ListStaffSessions() revoked_at = %d, want %d", listResp.GetItems()[1].GetRevokedAt(), revokedAt.Unix())
	}

	if _, err := server.RevokeStaffSession(context.Background(), &upbv1.RevokeStaffSessionRequest{SessionId: "staff-session-1"}); err != nil {
		t.Fatalf("RevokeStaffSession() error = %v", err)
	}
	if _, err := server.RevokeStaffUserSessions(context.Background(), &upbv1.RevokeStaffUserSessionsRequest{UserId: 7}); err != nil {
		t.Fatalf("RevokeStaffUserSessions() error = %v", err)
	}

	created, err := server.CreateBreakGlassApproval(context.Background(), &upbv1.CreateBreakGlassApprovalRequest{
		RequesterUserId: 7,
		Reason:          "emergency access",
		RequestId:       "req-1",
		ExpiresAt:       uint64(now.Add(4 * time.Hour).Unix()),
	})
	if err != nil {
		t.Fatalf("CreateBreakGlassApproval() error = %v", err)
	}
	if created.GetId() != "approval-1" || created.GetStatus() != "pending" {
		t.Fatalf("CreateBreakGlassApproval() response = %+v", created)
	}

	approved, err := server.ApproveBreakGlassApproval(context.Background(), &upbv1.ApproveBreakGlassApprovalRequest{
		ApprovalId:     "approval-1",
		ApproverUserId: 8,
		RequestId:      "req-2",
	})
	if err != nil {
		t.Fatalf("ApproveBreakGlassApproval() error = %v", err)
	}
	if approved.GetApprovedAt() != uint64(approvedAt.Unix()) {
		t.Fatalf("ApproveBreakGlassApproval() approved_at = %d, want %d", approved.GetApprovedAt(), approvedAt.Unix())
	}

	consumed, err := server.ConsumeBreakGlassApproval(context.Background(), &upbv1.ConsumeBreakGlassApprovalRequest{
		ApprovalId:      "approval-1",
		RequesterUserId: 7,
		RequestId:       "req-3",
	})
	if err != nil {
		t.Fatalf("ConsumeBreakGlassApproval() error = %v", err)
	}
	if consumed.GetUsedAt() != uint64(usedAt.Unix()) {
		t.Fatalf("ConsumeBreakGlassApproval() used_at = %d, want %d", consumed.GetUsedAt(), usedAt.Unix())
	}
}

func TestSessionResponseHelpers(t *testing.T) {
	if got := sessionResponse(nil); got.GetId() != "" {
		t.Fatalf("sessionResponse(nil) = %+v, want zero value", got)
	}
	if got := breakGlassApprovalResponse(nil); got.GetId() != "" {
		t.Fatalf("breakGlassApprovalResponse(nil) = %+v, want zero value", got)
	}
	if got := optionalUnix(0); got != nil {
		t.Fatalf("optionalUnix(0) = %v, want nil", got)
	}
}
