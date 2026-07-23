package data

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"goshop/app/review/srv/internal/domain"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrConflict = stderrors.New("review conflict")
var ErrNotFound = stderrors.New("review not found")
var ErrInvalidState = stderrors.New("invalid review state")

type Store struct{ db *gorm.DB }

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) Create(ctx context.Context, review *domain.Review) error {
	review.Status = domain.StatusPending
	if err := s.db.WithContext(ctx).Create(review).Error; err != nil {
		return translate(err)
	}
	return nil
}
func (s *Store) Get(ctx context.Context, id uint64) (*domain.Review, error) {
	var value domain.Review
	if err := s.db.WithContext(ctx).Preload("Append").Preload("Reply").First(&value, id).Error; err != nil {
		return nil, translate(err)
	}
	return &value, nil
}
func (s *Store) Append(ctx context.Context, userID int32, reviewID uint64, content string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var review domain.Review
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&review, reviewID).Error; err != nil {
			return translate(err)
		}
		if review.UserID != userID {
			return ErrNotFound
		}
		if err := tx.Create(&domain.ReviewAppend{ReviewID: reviewID, Content: content}).Error; err != nil {
			return translate(err)
		}
		return nil
	})
}
func (s *Store) Moderate(ctx context.Context, reviewID uint64, decision string, actor int32, requestID, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var review domain.Review
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&review, reviewID).Error; err != nil {
			return translate(err)
		}
		if review.Status != domain.StatusPending {
			return ErrInvalidState
		}
		if err := tx.Model(&review).Update("status", decision).Error; err != nil {
			return err
		}
		if err := tx.Create(&domain.Audit{ReviewID: reviewID, ActorUserID: actor, Action: "MODERATE", FromStatus: review.Status, ToStatus: decision, RequestID: requestID, Reason: reason}).Error; err != nil {
			return err
		}
		key := fmt.Sprintf("review:%d:moderate:%s", reviewID, decision)
		return tx.Create(&domain.OutboxEvent{EventKey: key, GoodsID: review.GoodsID, EventType: "RATING_REBUILD", Status: "PENDING"}).Error
	})
}
func (s *Store) Reply(ctx context.Context, reviewID uint64, actor int32, content, requestID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var review domain.Review
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&review, reviewID).Error; err != nil {
			return translate(err)
		}
		if review.Status != domain.StatusApproved {
			return ErrInvalidState
		}
		if err := tx.Create(&domain.ReviewReply{ReviewID: reviewID, ActorUserID: actor, Content: content}).Error; err != nil {
			return translate(err)
		}
		return tx.Create(&domain.Audit{ReviewID: reviewID, ActorUserID: actor, Action: "REPLY", FromStatus: review.Status, ToStatus: review.Status, RequestID: requestID}).Error
	})
}
func (s *Store) List(ctx context.Context, goodsID, userID int32, status string, offset, limit int) ([]domain.Review, int64, error) {
	q := s.db.WithContext(ctx).Model(&domain.Review{})
	if goodsID > 0 {
		q = q.Where("goods_id = ?", goodsID)
	}
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var values []domain.Review
	if err := q.Preload("Append").Preload("Reply").Order("id DESC").Offset(offset).Limit(limit).Find(&values).Error; err != nil {
		return nil, 0, err
	}
	return values, total, nil
}
func (s *Store) RebuildRating(ctx context.Context, goodsID, actor int32, requestID string) (*domain.Rating, error) {
	var count, sum int64
	row := s.db.WithContext(ctx).Model(&domain.Review{}).Select("COUNT(*), COALESCE(SUM(rating),0)").Where("goods_id = ? AND status = ?", goodsID, domain.StatusApproved).Row()
	if err := row.Scan(&count, &sum); err != nil {
		return nil, err
	}
	avg := int32(0)
	if count > 0 {
		avg = int32(sum * 1000 / count)
	}
	value := &domain.Rating{GoodsID: goodsID, ApprovedCount: count, RatingSum: sum, AverageMilli: avg, RebuiltAt: time.Now().UTC()}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "goods_id"}}, DoUpdates: clause.AssignmentColumns([]string{"approved_count", "rating_sum", "average_milli", "rebuilt_at"})}).Create(value).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(&domain.Audit{ActorUserID: actor, Action: "REBUILD_RATING", RequestID: requestID, Reason: fmt.Sprintf("goods:%d", goodsID)}).Error; err != nil {
		return nil, err
	}
	return value, nil
}
func (s *Store) GetRating(ctx context.Context, goodsID int32) (*domain.Rating, error) {
	var v domain.Rating
	if err := s.db.WithContext(ctx).First(&v, "goods_id = ?", goodsID).Error; err != nil {
		return nil, translate(err)
	}
	return &v, nil
}
func (s *Store) ProcessOutbox(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 50
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var events []domain.OutboxEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status = ? AND next_attempt_at <= ?", "PENDING", time.Now().Unix()).Order("id").Limit(limit).Find(&events).Error; err != nil {
			return err
		}
		for _, event := range events {
			var count, sum int64
			if err := tx.Model(&domain.Review{}).Select("COUNT(*), COALESCE(SUM(rating),0)").Where("goods_id = ? AND status = ?", event.GoodsID, domain.StatusApproved).Row().Scan(&count, &sum); err != nil {
				return err
			}
			avg := int32(0)
			if count > 0 {
				avg = int32(sum * 1000 / count)
			}
			rating := domain.Rating{GoodsID: event.GoodsID, ApprovedCount: count, RatingSum: sum, AverageMilli: avg, RebuiltAt: time.Now().UTC()}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "goods_id"}}, DoUpdates: clause.AssignmentColumns([]string{"approved_count", "rating_sum", "average_milli", "rebuilt_at"})}).Create(&rating).Error; err != nil {
				return err
			}
			now := time.Now().UTC()
			if err := tx.Model(&event).Updates(map[string]any{"status": "DONE", "completed_at": &now, "last_error": ""}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func translate(err error) error {
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	var me *mysql.MySQLError
	if stderrors.As(err, &me) && me.Number == 1062 {
		return ErrConflict
	}
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return ErrConflict
	}
	return err
}
