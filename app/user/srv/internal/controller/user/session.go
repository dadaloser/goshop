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
	session, err := s.srv.CreateSession(ctx, srvv1.SessionDTO{UserID: req.GetUserId(), DeviceID: req.GetDeviceId(), DeviceName: req.GetDeviceName(), RefreshTokenHash: req.GetRefreshTokenHash(), ExpiresAt: time.Unix(int64(req.GetExpiresAt()), 0)})
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

func sessionResponse(session *srvv1.SessionDTO) *upbv1.SessionResponse {
	if session == nil {
		return &upbv1.SessionResponse{}
	}
	return &upbv1.SessionResponse{Id: session.ID, UserId: session.UserID, DeviceId: session.DeviceID, DeviceName: session.DeviceName, ExpiresAt: uint64(session.ExpiresAt.Unix())}
}
