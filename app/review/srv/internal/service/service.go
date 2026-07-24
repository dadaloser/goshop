package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	opb "goshop/api/order/v1"
	"goshop/app/review/srv/internal/data"
	"goshop/app/review/srv/internal/domain"
	"goshop/pkg/log"
)

var ErrInvalid = errors.New("invalid review request")
var ErrPurchaseRequired = errors.New("completed purchase required")

type Repository interface {
	Create(context.Context, *domain.Review) error
	Get(context.Context, uint64) (*domain.Review, error)
	Append(context.Context, int32, uint64, string) error
	Moderate(context.Context, uint64, string, int32, string, string) error
	Reply(context.Context, uint64, int32, string, string) error
	List(context.Context, int32, int32, string, int, int) ([]domain.Review, int64, error)
	RebuildRating(context.Context, int32, int32, string) (*domain.Rating, error)
	GetRating(context.Context, int32) (*domain.Rating, error)
	ProcessOutbox(context.Context, int) error
}
type PurchaseVerifier interface {
	VerifyCompleted(context.Context, int32, string, int32) error
}
type Service struct {
	repo     Repository
	verifier PurchaseVerifier
	outbox   OutboxWorkerConfig
}

type OutboxWorkerConfig struct {
	PollInterval time.Duration
	BatchSize    int
}

type Option func(*Service)

func WithOutboxWorker(pollInterval time.Duration, batchSize int) Option {
	return func(s *Service) {
		s.outbox = OutboxWorkerConfig{
			PollInterval: pollInterval,
			BatchSize:    batchSize,
		}.normalize()
	}
}

func New(repo Repository, verifier PurchaseVerifier, opts ...Option) *Service {
	svc := &Service{
		repo:     repo,
		verifier: verifier,
		outbox:   OutboxWorkerConfig{}.normalize(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc
}

func (s *Service) Create(ctx context.Context, user int32, orderSN string, goods, rating int32, content string) (*domain.Review, error) {
	orderSN = strings.TrimSpace(orderSN)
	content = strings.TrimSpace(content)
	if user <= 0 || goods <= 0 || rating < 1 || rating > 5 || orderSN == "" || content == "" || len(content) > 2000 {
		return nil, ErrInvalid
	}
	if s.verifier == nil || s.verifier.VerifyCompleted(ctx, user, orderSN, goods) != nil {
		return nil, ErrPurchaseRequired
	}
	v := &domain.Review{UserID: user, OrderSN: orderSN, GoodsID: goods, Rating: rating, Content: content}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, v.ID)
}
func (s *Service) Append(ctx context.Context, user int32, id uint64, content string) (*domain.Review, error) {
	content = strings.TrimSpace(content)
	if user <= 0 || id == 0 || content == "" || len(content) > 2000 {
		return nil, ErrInvalid
	}
	if err := s.repo.Append(ctx, user, id, content); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}
func (s *Service) Moderate(ctx context.Context, id uint64, decision string, actor int32, requestID, reason string) (*domain.Review, error) {
	decision = strings.ToUpper(strings.TrimSpace(decision))
	if id == 0 || actor <= 0 || strings.TrimSpace(requestID) == "" || (decision != domain.StatusApproved && decision != domain.StatusRejected) {
		return nil, ErrInvalid
	}
	if err := s.repo.Moderate(ctx, id, decision, actor, requestID, reason); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}
func (s *Service) Reply(ctx context.Context, id uint64, actor int32, content, requestID string) (*domain.Review, error) {
	content = strings.TrimSpace(content)
	if id == 0 || actor <= 0 || content == "" || len(content) > 2000 || strings.TrimSpace(requestID) == "" {
		return nil, ErrInvalid
	}
	if err := s.repo.Reply(ctx, id, actor, content, requestID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}
func (s *Service) List(ctx context.Context, goods, user int32, status string, page, size int) ([]domain.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	return s.repo.List(ctx, goods, user, status, (page-1)*size, size)
}
func (s *Service) GetRating(ctx context.Context, goods int32) (*domain.Rating, error) {
	if goods <= 0 {
		return nil, ErrInvalid
	}
	return s.repo.GetRating(ctx, goods)
}
func (s *Service) RebuildRating(ctx context.Context, goods, actor int32, requestID string) (*domain.Rating, error) {
	if goods <= 0 || actor <= 0 || strings.TrimSpace(requestID) == "" {
		return nil, ErrInvalid
	}
	return s.repo.RebuildRating(ctx, goods, actor, requestID)
}
func (s *Service) RunOutbox(ctx context.Context) error {
	cfg := s.outbox.normalize()
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := s.repo.ProcessOutbox(ctx, cfg.BatchSize); err != nil {
			log.Errorf("process review rating outbox: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (c OutboxWorkerConfig) normalize() OutboxWorkerConfig {
	if c.PollInterval <= 0 {
		c.PollInterval = 2 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	return c
}

type OrderVerifier struct{ client opb.OrderClient }

func NewOrderVerifier(client opb.OrderClient) *OrderVerifier { return &OrderVerifier{client: client} }
func (v *OrderVerifier) VerifyCompleted(ctx context.Context, user int32, orderSN string, goods int32) error {
	if v == nil || v.client == nil {
		return ErrPurchaseRequired
	}
	detail, err := v.client.GetOrderBySn(ctx, &opb.OrderLookupRequest{OrderSn: orderSN})
	if err != nil {
		return ErrPurchaseRequired
	}
	if detail.GetOrderInfo().GetUserId() != user || detail.GetOrderInfo().GetStatus() != "TRADE_FINISHED" {
		return ErrPurchaseRequired
	}
	for _, item := range detail.GetGoods() {
		if item.GetGoodsId() == goods {
			return nil
		}
	}
	return ErrPurchaseRequired
}

type DBOrderVerifier struct{ db *gorm.DB }

func NewDBOrderVerifier(db *gorm.DB) *DBOrderVerifier { return &DBOrderVerifier{db: db} }
func (v *DBOrderVerifier) VerifyCompleted(ctx context.Context, user int32, orderSN string, goods int32) error {
	if v == nil || v.db == nil {
		return ErrPurchaseRequired
	}
	var count int64
	err := v.db.WithContext(ctx).Table("orderinfo AS o").
		Joins("JOIN ordergoods AS g ON g.`order` = o.id").
		Where("o.user = ? AND o.order_sn = ? AND o.status = ? AND g.goods = ?", user, orderSN, "TRADE_FINISHED", goods).
		Count(&count).Error
	if err != nil || count == 0 {
		return ErrPurchaseRequired
	}
	return nil
}

var _ Repository = (*data.Store)(nil)
