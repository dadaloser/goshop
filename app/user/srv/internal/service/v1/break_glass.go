package v1

import (
	"context"
	"strings"
	"time"

	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/pkg/errcode"
	"goshop/pkg/errors"

	"github.com/google/uuid"
)

type breakGlassApprovalStore interface {
	CreateBreakGlassApproval(ctx context.Context, approval *dv1.BreakGlassApprovalDO) error
	ApproveBreakGlassApproval(ctx context.Context, approvalID string, approverUserID int32, requestID string, now time.Time) (*dv1.BreakGlassApprovalDO, error)
	ConsumeBreakGlassApproval(ctx context.Context, approvalID string, requesterUserID int32, requestID string, now time.Time) (*dv1.BreakGlassApprovalDO, error)
}

func (u *userService) CreateBreakGlassApproval(ctx context.Context, requesterUserID int32, reason, requestID string, expiresAt time.Time) (*BreakGlassApprovalDTO, error) {
	store, ok := u.userStore.(breakGlassApprovalStore)
	if !ok {
		return nil, errors.NewCode(errcode.ErrDatabase, "break-glass approval store is not configured")
	}
	reason = strings.TrimSpace(reason)
	requestID = strings.TrimSpace(requestID)
	if requesterUserID <= 0 || reason == "" || requestID == "" || !expiresAt.After(time.Now().UTC()) {
		return nil, errors.NewCode(errcode.ErrValidation, "invalid break-glass approval request")
	}
	model := &dv1.BreakGlassApprovalDO{
		ID:              uuid.NewString(),
		RequesterUserID: requesterUserID,
		Status:          "pending",
		Reason:          reason,
		RequestID:       requestID,
		CreatedAt:       time.Now().UTC(),
		ExpiresAt:       expiresAt.UTC(),
	}
	if err := store.CreateBreakGlassApproval(ctx, model); err != nil {
		return nil, err
	}
	return breakGlassApprovalDTO(model), nil
}

func (u *userService) ApproveBreakGlassApproval(ctx context.Context, approvalID string, approverUserID int32, requestID string) (*BreakGlassApprovalDTO, error) {
	store, ok := u.userStore.(breakGlassApprovalStore)
	if !ok {
		return nil, errors.NewCode(errcode.ErrDatabase, "break-glass approval store is not configured")
	}
	model, err := store.ApproveBreakGlassApproval(ctx, strings.TrimSpace(approvalID), approverUserID, strings.TrimSpace(requestID), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return breakGlassApprovalDTO(model), nil
}

func (u *userService) ConsumeBreakGlassApproval(ctx context.Context, approvalID string, requesterUserID int32, requestID string) (*BreakGlassApprovalDTO, error) {
	store, ok := u.userStore.(breakGlassApprovalStore)
	if !ok {
		return nil, errors.NewCode(errcode.ErrDatabase, "break-glass approval store is not configured")
	}
	model, err := store.ConsumeBreakGlassApproval(ctx, strings.TrimSpace(approvalID), requesterUserID, strings.TrimSpace(requestID), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return breakGlassApprovalDTO(model), nil
}

func breakGlassApprovalDTO(model *dv1.BreakGlassApprovalDO) *BreakGlassApprovalDTO {
	if model == nil {
		return &BreakGlassApprovalDTO{}
	}
	return &BreakGlassApprovalDTO{
		ID:              model.ID,
		RequesterUserID: model.RequesterUserID,
		ApproverUserID:  model.ApproverUserID,
		Status:          model.Status,
		Reason:          model.Reason,
		RequestID:       model.RequestID,
		CreatedAt:       model.CreatedAt,
		ApprovedAt:      model.ApprovedAt,
		ExpiresAt:       model.ExpiresAt,
		UsedAt:          model.UsedAt,
	}
}
