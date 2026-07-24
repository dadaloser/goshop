package order

import (
	"context"
	"time"

	pb "goshop/api/order/v1"
	"goshop/app/order/srv/internal/domain/do"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"google.golang.org/protobuf/types/known/emptypb"
)

type paymentService interface {
	BeginPaymentEvent(context.Context, *do.PaymentEventDO) (*do.PaymentEventDO, *do.OrderInfoDO, bool, error)
	CompletePaymentEvent(context.Context, uint64, bool, string) error
	ListPaymentEvents(context.Context, string, int, int, bool) ([]do.PaymentEventDO, int64, int64, error)
	ClaimRefundJobs(context.Context, int, int, time.Duration) ([]do.RefundJob, error)
	CompleteRefundJob(context.Context, uint64, bool, string, string, string, string, int) error
	ReconcilePayments(context.Context, string, time.Time, time.Time, []do.PaymentEventDO) (*do.PaymentReconciliationRunDO, error)
}

func (os *orderServer) ClaimRefundJobs(ctx context.Context, req *pb.ClaimRefundJobsRequest) (*pb.ClaimRefundJobsResponse, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	jobs, err := service.ClaimRefundJobs(ctx, int(req.GetLimit()), int(req.GetMaxAttempts()), time.Duration(req.GetLockTimeoutSeconds())*time.Second)
	if err != nil {
		return nil, err
	}
	resp := &pb.ClaimRefundJobsResponse{Jobs: make([]*pb.RefundJob, 0, len(jobs))}
	for _, job := range jobs {
		resp.Jobs = append(resp.Jobs, &pb.RefundJob{Id: int64(job.OutboxID), RefundRequestId: int64(job.RefundRequestID), OrderSn: job.OrderSN, TradeNo: job.TradeNo, AmountFen: job.AmountFen, Reason: job.Reason, CorrelationId: job.CorrelationID, Attempts: int32(job.Attempts)})
	}
	return resp, nil
}

func (os *orderServer) CompleteRefundJob(ctx context.Context, req *pb.CompleteRefundJobRequest) (*emptypb.Empty, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	if err := service.CompleteRefundJob(ctx, uint64(req.GetId()), req.GetSuccess(), req.GetProvider(), req.GetProviderRefundId(), req.GetProviderStatus(), req.GetErrorDetail(), int(req.GetMaxAttempts())); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (os *orderServer) ReconcilePayments(ctx context.Context, req *pb.ReconcilePaymentsRequest) (*pb.ReconcilePaymentsResponse, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	transactions := make([]do.PaymentEventDO, 0, len(req.GetTransactions()))
	for _, item := range req.GetTransactions() {
		transactions = append(transactions, do.PaymentEventDO{Provider: req.GetProvider(), EventID: item.GetEventId(), OrderSN: item.GetOrderSn(), TradeNo: item.GetTradeNo(), EventType: item.GetEventType(), ProviderAmountFen: item.GetAmountFen(), ReceivedAt: time.Unix(item.GetOccurredAt(), 0).UTC()})
	}
	run, err := service.ReconcilePayments(ctx, req.GetProvider(), time.Unix(req.GetWindowStart(), 0), time.Unix(req.GetWindowEnd(), 0), transactions)
	if err != nil {
		return nil, err
	}
	return &pb.ReconcilePaymentsResponse{RunId: int64(run.ID), CheckedCount: int32(run.CheckedCount), MismatchCount: int32(run.MismatchCount)}, nil
}

func (os *orderServer) paymentService() (paymentService, error) {
	service, ok := os.srv.Orders().(paymentService)
	if !ok {
		return nil, errors.WithCode(code2.ErrDatabase, "payment service is not configured")
	}
	return service, nil
}

func (os *orderServer) BeginPaymentEvent(ctx context.Context, req *pb.PaymentEventRequest) (*pb.PaymentEventResponse, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	event, order, accepted, err := service.BeginPaymentEvent(ctx, &do.PaymentEventDO{Provider: req.GetProvider(), EventID: req.GetEventId(), OrderSN: req.GetOrderSn(), TradeNo: req.GetTradeNo(), EventType: req.GetEventType(), ProviderAmountFen: req.GetProviderAmountFen(), RefundAmountFen: req.GetRefundAmountFen()})
	if err != nil {
		return nil, err
	}
	return &pb.PaymentEventResponse{Id: int64(event.ID), Accepted: accepted, Completed: event.Status == "completed", OrderAmountFen: event.OrderAmountFen, OrderStatus: order.Status}, nil
}
func (os *orderServer) CompletePaymentEvent(ctx context.Context, req *pb.CompletePaymentEventRequest) (*emptypb.Empty, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	if err = service.CompletePaymentEvent(ctx, uint64(req.GetId()), req.GetSuccess(), req.GetErrorDetail()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
func (os *orderServer) ListPaymentEvents(ctx context.Context, req *pb.PaymentEventListRequest) (*pb.PaymentEventListResponse, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	items, total, mismatches, err := service.ListPaymentEvents(ctx, req.GetOrderSn(), int(req.GetPage()), int(req.GetPageSize()), req.GetMismatchesOnly())
	if err != nil {
		return nil, err
	}
	resp := &pb.PaymentEventListResponse{Total: int32(total), MismatchCount: int32(mismatches), Data: make([]*pb.PaymentEventRecord, 0, len(items))}
	for _, item := range items {
		var completed int64
		if item.CompletedAt != nil {
			completed = item.CompletedAt.Unix()
		}
		resp.Data = append(resp.Data, &pb.PaymentEventRecord{Id: int64(item.ID), Provider: item.Provider, EventId: item.EventID, OrderSn: item.OrderSN, TradeNo: item.TradeNo, EventType: item.EventType, OrderAmountFen: item.OrderAmountFen, ProviderAmountFen: item.ProviderAmountFen, RefundAmountFen: item.RefundAmountFen, Status: item.Status, ErrorDetail: item.ErrorDetail, ReceivedAt: item.ReceivedAt.Unix(), CompletedAt: completed})
	}
	return resp, nil
}
