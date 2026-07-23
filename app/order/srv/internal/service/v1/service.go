package service

import (
	"context"
	"goshop/app/order/srv/internal/boundary"
	v1 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/pkg/options"
	reviewsrv "goshop/app/review/srv"
	"time"

	"golang.org/x/sync/errgroup"
)

type ServiceFactory interface {
	Orders() OrderSrv
	RunBackground(ctx context.Context) error
}

type LifecycleConfig struct {
	PollInterval       time.Duration
	TimeoutCloseAfter  time.Duration
	FinishAfterPayment time.Duration
	BatchSize          int
}

type service struct {
	data      v1.DataFactory
	dtmopts   *options.DtmOptions
	upstream  upstream
	now       func() time.Time
	lifecycle LifecycleConfig
	reviews   *reviewsrv.Service
}

type upstream struct {
	goods     boundary.GoodsGateway
	inventory boundary.InventoryGateway
}

func (s *service) Orders() OrderSrv {
	return newOrderService(s)
}

func (s *service) RunBackground(ctx context.Context) error {
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error { return s.runLifecycleWorker(groupCtx) })
	if s.reviews != nil {
		group.Go(func() error { return s.reviews.RunOutbox(groupCtx) })
	}
	return group.Wait()
}

func (s *service) SetReviewService(reviews *reviewsrv.Service) { s.reviews = reviews }
func (s *service) ReviewService() *reviewsrv.Service           { return s.reviews }

var _ ServiceFactory = &service{}

func NewService(data v1.DataFactory, dtmopts *options.DtmOptions, goods boundary.GoodsGateway, inventory boundary.InventoryGateway, lifecycle LifecycleConfig) *service {
	return &service{
		data:      data,
		dtmopts:   dtmopts,
		upstream:  upstream{goods: goods, inventory: inventory},
		now:       time.Now,
		lifecycle: lifecycle.normalize(),
	}
}

func (c LifecycleConfig) normalize() LifecycleConfig {
	if c.PollInterval <= 0 {
		c.PollInterval = orderLifecyclePollInterval
	}
	if c.TimeoutCloseAfter <= 0 {
		c.TimeoutCloseAfter = orderTimeoutCloseAfter
	}
	if c.FinishAfterPayment <= 0 {
		c.FinishAfterPayment = orderFinishAfterPayment
	}
	if c.BatchSize <= 0 {
		c.BatchSize = orderLifecycleBatchSize
	}
	return c
}
