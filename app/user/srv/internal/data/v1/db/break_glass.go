package db

import (
	"context"
	stderrors "errors"
	"time"

	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (u *users) CreateBreakGlassApproval(ctx context.Context, approval *dv1.BreakGlassApprovalDO) error {
	if approval == nil || approval.ID == "" {
		return errors.NewCode(errcode.ErrValidation, "break-glass approval is invalid")
	}
	if err := u.db.WithContext(ctx).Create(approval).Error; err != nil {
		return errors.NewCode(errcode.ErrDatabase, err.Error())
	}
	return nil
}

func (u *users) ApproveBreakGlassApproval(ctx context.Context, approvalID string, approverUserID int32, requestID string, now time.Time) (*dv1.BreakGlassApprovalDO, error) {
	var approval dv1.BreakGlassApprovalDO
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", approvalID).First(&approval).Error; err != nil {
			return err
		}
		if approval.Status != "pending" || !approval.ExpiresAt.After(now.UTC()) {
			return gorm.ErrRecordNotFound
		}
		if approval.RequesterUserID == approverUserID {
			return errors.NewCode(errcode.ErrValidation, "break-glass approver must be different from requester")
		}
		approvedAt := now.UTC()
		approval.ApproverUserID = approverUserID
		approval.Status = "approved"
		approval.RequestID = requestID
		approval.ApprovedAt = &approvedAt
		return tx.Model(&approval).Updates(map[string]interface{}{
			"approver_user_id": approverUserID,
			"status":           approval.Status,
			"request_id":       requestID,
			"approved_at":      approvedAt,
		}).Error
	})
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewCode(errcode.ErrValidation, "break-glass approval is not approvable")
		}
		return nil, err
	}
	return &approval, nil
}

func (u *users) ConsumeBreakGlassApproval(ctx context.Context, approvalID string, requesterUserID int32, requestID string, now time.Time) (*dv1.BreakGlassApprovalDO, error) {
	var approval dv1.BreakGlassApprovalDO
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", approvalID).First(&approval).Error; err != nil {
			return err
		}
		if approval.Status != "approved" || approval.RequesterUserID != requesterUserID || !approval.ExpiresAt.After(now.UTC()) || approval.UsedAt != nil {
			return gorm.ErrRecordNotFound
		}
		usedAt := now.UTC()
		approval.Status = "used"
		approval.RequestID = requestID
		approval.UsedAt = &usedAt
		return tx.Model(&approval).Updates(map[string]interface{}{
			"status":     approval.Status,
			"request_id": requestID,
			"used_at":    usedAt,
		}).Error
	})
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewCode(errcode.ErrValidation, "break-glass approval is not usable")
		}
		return nil, err
	}
	return &approval, nil
}
