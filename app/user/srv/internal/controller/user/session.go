package user

import (
	"context"
	"time"

	upbv1 "goshop/api/user/v1"
	srvv1 "goshop/app/user/srv/internal/service/v1"

	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *userServer) RecordLogin(ctx context.Context, req *upbv1.RecordLoginRequest) (*emptypb.Empty, error) {
	if err := s.srv.RecordLogin(ctx, uint64(req.GetUserId()), time.Unix(int64(req.GetLoggedInAt()), 0)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *userServer) CreateSession(ctx context.Context, req *upbv1.CreateSessionRequest) (*upbv1.SessionResponse, error) {
	session, err := s.srv.CreateSession(ctx, srvv1.SessionDTO{UserID: req.GetUserId(), PrincipalType: req.GetPrincipalType(), DeviceID: req.GetDeviceId(), DeviceName: req.GetDeviceName(), ClientIP: req.GetClientIp(), Location: req.GetLocation(), RefreshTokenHash: req.GetRefreshTokenHash(), ExpiresAt: time.Unix(int64(req.GetExpiresAt()), 0)})
	if err != nil {
		return nil, err
	}
	return sessionResponse(session), nil
}

func (s *userServer) RefreshSession(ctx context.Context, req *upbv1.RefreshSessionRequest) (*upbv1.SessionResponse, error) {
	session, err := s.srv.RefreshSession(ctx, req.GetSessionId(), req.GetCurrentTokenHash(), req.GetNextTokenHash(), time.Unix(int64(req.GetExpiresAt()), 0))
	if err != nil {
		return nil, err
	}
	return sessionResponse(session), nil
}

func (s *userServer) RevokeSession(ctx context.Context, req *upbv1.RevokeSessionRequest) (*emptypb.Empty, error) {
	if err := s.srv.RevokeSession(ctx, uint64(req.GetUserId()), req.GetSessionId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *userServer) RevokeAllSessions(ctx context.Context, req *upbv1.IdRequest) (*emptypb.Empty, error) {
	if err := s.srv.RevokeAllSessions(ctx, uint64(req.GetId())); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *userServer) ValidateSession(ctx context.Context, req *upbv1.ValidateSessionRequest) (*upbv1.SessionValidationResponse, error) {
	active, err := s.srv.ValidateSession(ctx, uint64(req.GetUserId()), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	return &upbv1.SessionValidationResponse{Active: active}, nil
}

func (s *userServer) ListUserSessions(ctx context.Context, req *upbv1.ListUserSessionsRequest) (*upbv1.ListUserSessionsResponse, error) {
	result, err := s.srv.ListUserSessions(ctx, uint64(req.GetUserId()), int(req.GetPn()), int(req.GetPSize()))
	if err != nil {
		return nil, err
	}
	resp := &upbv1.ListUserSessionsResponse{Total: int32(result.TotalCount), Items: make([]*upbv1.UserSessionRecord, 0, len(result.Items))}
	now := time.Now().UTC()
	for _, item := range result.Items {
		record := &upbv1.UserSessionRecord{Id: item.ID, DeviceId: item.DeviceID, DeviceName: item.DeviceName, ClientIp: item.ClientIP, Location: item.Location, CreatedAt: uint64(item.CreatedAt.Unix()), LastUsedAt: uint64(item.LastUsedAt.Unix()), ExpiresAt: uint64(item.ExpiresAt.Unix()), Active: item.RevokedAt == nil && item.ExpiresAt.After(now)}
		if item.RevokedAt != nil {
			record.RevokedAt = uint64(item.RevokedAt.Unix())
		}
		resp.Items = append(resp.Items, record)
	}
	return resp, nil
}
func (s *userServer) AddDeviceBlacklist(ctx context.Context, req *upbv1.DeviceBlacklistRequest) (*emptypb.Empty, error) {
	if err := s.srv.AddDeviceBlacklist(ctx, req.GetUserId(), req.GetDeviceId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
func (s *userServer) DeleteDeviceBlacklist(ctx context.Context, req *upbv1.DeviceBlacklistRequest) (*emptypb.Empty, error) {
	if err := s.srv.DeleteDeviceBlacklist(ctx, req.GetUserId(), req.GetDeviceId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
func (s *userServer) ListDeviceBlacklist(ctx context.Context, req *upbv1.ListDeviceBlacklistRequest) (*upbv1.ListDeviceBlacklistResponse, error) {
	result, err := s.srv.ListDeviceBlacklist(ctx, req.GetUserId(), int(req.GetPn()), int(req.GetPSize()))
	if err != nil {
		return nil, err
	}
	resp := &upbv1.ListDeviceBlacklistResponse{Total: int32(result.TotalCount), Items: make([]*upbv1.DeviceBlacklistRecord, 0, len(result.Items))}
	for _, item := range result.Items {
		resp.Items = append(resp.Items, &upbv1.DeviceBlacklistRecord{UserId: item.UserID, DeviceId: item.DeviceID, CreatedAt: uint64(item.CreatedAt.Unix())})
	}
	return resp, nil
}

func (s *userServer) ListStaffSessions(ctx context.Context, req *upbv1.ListStaffSessionsRequest) (*upbv1.ListStaffSessionsResponse, error) {
	result, err := s.srv.ListStaffSessions(ctx, srvv1.StaffSessionFilterDTO{
		UserID:         req.GetUserId(),
		Role:           req.GetRole(),
		ActiveOnly:     req.GetActiveOnly(),
		CreatedAfter:   optionalUnix(req.GetCreatedAfter()),
		CreatedBefore:  optionalUnix(req.GetCreatedBefore()),
		LastUsedAfter:  optionalUnix(req.GetLastUsedAfter()),
		LastUsedBefore: optionalUnix(req.GetLastUsedBefore()),
		Page:           int(req.GetPn()),
		PageSize:       int(req.GetPSize()),
	})
	if err != nil {
		return nil, err
	}
	resp := &upbv1.ListStaffSessionsResponse{Total: int32(result.TotalCount), Items: make([]*upbv1.StaffSessionRecord, 0, len(result.Items))}
	now := time.Now().UTC()
	for _, item := range result.Items {
		record := &upbv1.StaffSessionRecord{
			Id:            item.ID,
			UserId:        item.UserID,
			DeviceId:      item.DeviceID,
			DeviceName:    item.DeviceName,
			PrincipalType: item.PrincipalType,
			CreatedAt:     uint64(item.CreatedAt.Unix()),
			LastUsedAt:    uint64(item.LastUsedAt.Unix()),
			ExpiresAt:     uint64(item.ExpiresAt.Unix()),
			Roles:         append([]string(nil), item.Roles...),
			Active:        item.RevokedAt == nil && item.ExpiresAt.After(now),
		}
		if item.RevokedAt != nil {
			record.RevokedAt = uint64(item.RevokedAt.Unix())
		}
		resp.Items = append(resp.Items, record)
	}
	return resp, nil
}

func (s *userServer) RevokeStaffSession(ctx context.Context, req *upbv1.RevokeStaffSessionRequest) (*emptypb.Empty, error) {
	if err := s.srv.RevokeStaffSession(ctx, req.GetSessionId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *userServer) RevokeStaffUserSessions(ctx context.Context, req *upbv1.RevokeStaffUserSessionsRequest) (*emptypb.Empty, error) {
	if err := s.srv.RevokeStaffUserSessions(ctx, uint64(req.GetUserId())); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *userServer) CreateBreakGlassApproval(ctx context.Context, req *upbv1.CreateBreakGlassApprovalRequest) (*upbv1.BreakGlassApproval, error) {
	approval, err := s.srv.CreateBreakGlassApproval(ctx, req.GetRequesterUserId(), req.GetReason(), req.GetRequestId(), time.Unix(int64(req.GetExpiresAt()), 0))
	if err != nil {
		return nil, err
	}
	return breakGlassApprovalResponse(approval), nil
}

func (s *userServer) ApproveBreakGlassApproval(ctx context.Context, req *upbv1.ApproveBreakGlassApprovalRequest) (*upbv1.BreakGlassApproval, error) {
	approval, err := s.srv.ApproveBreakGlassApproval(ctx, req.GetApprovalId(), req.GetApproverUserId(), req.GetRequestId())
	if err != nil {
		return nil, err
	}
	return breakGlassApprovalResponse(approval), nil
}

func (s *userServer) ConsumeBreakGlassApproval(ctx context.Context, req *upbv1.ConsumeBreakGlassApprovalRequest) (*upbv1.BreakGlassApproval, error) {
	approval, err := s.srv.ConsumeBreakGlassApproval(ctx, req.GetApprovalId(), req.GetRequesterUserId(), req.GetRequestId())
	if err != nil {
		return nil, err
	}
	return breakGlassApprovalResponse(approval), nil
}

func sessionResponse(session *srvv1.SessionDTO) *upbv1.SessionResponse {
	if session == nil {
		return &upbv1.SessionResponse{}
	}
	return &upbv1.SessionResponse{Id: session.ID, UserId: session.UserID, DeviceId: session.DeviceID, DeviceName: session.DeviceName, ExpiresAt: uint64(session.ExpiresAt.Unix())}
}

func optionalUnix(value uint64) *time.Time {
	if value == 0 {
		return nil
	}
	parsed := time.Unix(int64(value), 0).UTC()
	return &parsed
}

func breakGlassApprovalResponse(approval *srvv1.BreakGlassApprovalDTO) *upbv1.BreakGlassApproval {
	if approval == nil {
		return &upbv1.BreakGlassApproval{}
	}
	resp := &upbv1.BreakGlassApproval{
		Id:              approval.ID,
		RequesterUserId: approval.RequesterUserID,
		ApproverUserId:  approval.ApproverUserID,
		Status:          approval.Status,
		Reason:          approval.Reason,
		RequestId:       approval.RequestID,
		CreatedAt:       uint64(approval.CreatedAt.Unix()),
		ExpiresAt:       uint64(approval.ExpiresAt.Unix()),
	}
	if approval.ApprovedAt != nil {
		resp.ApprovedAt = uint64(approval.ApprovedAt.Unix())
	}
	if approval.UsedAt != nil {
		resp.UsedAt = uint64(approval.UsedAt.Unix())
	}
	return resp
}
