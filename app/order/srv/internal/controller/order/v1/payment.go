package order

import (
	"context"
	"time"

	pb "goshop/api/order/v1"
	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/order/srv/internal/domain/dto"
	"goshop/gmicro/errcode"
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
	ListPaymentReconciliationRuns(context.Context, string, *time.Time, *time.Time, int, int) ([]do.PaymentReconciliationRunDO, int64, error)
	ListPaymentReconciliationItems(context.Context, string, *time.Time, *time.Time, string, uint64, int, int) ([]do.PaymentReconciliationItemDO, int64, error)
	RetryDeadRefundJob(context.Context, uint64) (*do.RefundJob, error)
	GetOrderTrace(context.Context, do.OrderTraceLookup) (*do.OrderTrace, error)
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

func (os *orderServer) ListPaymentReconciliationRuns(ctx context.Context, req *pb.ListPaymentReconciliationRunsRequest) (*pb.ListPaymentReconciliationRunsResponse, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	runs, total, err := service.ListPaymentReconciliationRuns(ctx, req.GetProvider(), optionalUnixTime(req.GetWindowStart()), optionalUnixTime(req.GetWindowEnd()), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	resp := &pb.ListPaymentReconciliationRunsResponse{Total: int32(total), Data: make([]*pb.PaymentReconciliationRunRecord, 0, len(runs))}
	for _, run := range runs {
		record := &pb.PaymentReconciliationRunRecord{
			Id:            int64(run.ID),
			Provider:      run.Provider,
			WindowStart:   run.WindowStart.Unix(),
			WindowEnd:     run.WindowEnd.Unix(),
			StartedAt:     run.StartedAt.Unix(),
			CheckedCount:  int32(run.CheckedCount),
			MismatchCount: int32(run.MismatchCount),
			Status:        run.Status,
		}
		if run.FinishedAt != nil {
			record.FinishedAt = run.FinishedAt.Unix()
		}
		resp.Data = append(resp.Data, record)
	}
	return resp, nil
}

func (os *orderServer) ListPaymentReconciliationItems(ctx context.Context, req *pb.ListPaymentReconciliationItemsRequest) (*pb.ListPaymentReconciliationItemsResponse, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	items, total, err := service.ListPaymentReconciliationItems(ctx, req.GetProvider(), optionalUnixTime(req.GetWindowStart()), optionalUnixTime(req.GetWindowEnd()), req.GetResult(), uint64(req.GetRunId()), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	resp := &pb.ListPaymentReconciliationItemsResponse{Total: int32(total), Data: make([]*pb.PaymentReconciliationItemRecord, 0, len(items))}
	for _, item := range items {
		resp.Data = append(resp.Data, &pb.PaymentReconciliationItemRecord{
			Id:                int64(item.ID),
			RunId:             int64(item.RunID),
			ProviderEventId:   item.ProviderEventID,
			OrderSn:           item.OrderSN,
			TradeNo:           item.TradeNo,
			EventType:         item.EventType,
			ProviderAmountFen: item.ProviderAmountFen,
			LocalAmountFen:    item.LocalAmountFen,
			Result:            item.Result,
			Detail:            item.Detail,
			CreatedAt:         item.CreatedAt.Unix(),
		})
	}
	return resp, nil
}

func (os *orderServer) RetryDeadRefundJob(ctx context.Context, req *pb.RetryDeadRefundJobRequest) (*pb.RetryDeadRefundJobResponse, error) {
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	job, err := service.RetryDeadRefundJob(ctx, uint64(req.GetId()))
	if err != nil {
		return nil, err
	}
	return &pb.RetryDeadRefundJobResponse{Id: int64(job.OutboxID), Status: "retry", Attempts: int32(job.Attempts), CorrelationId: job.CorrelationID}, nil
}

func (os *orderServer) GetOrderTrace(ctx context.Context, req *pb.OrderTraceRequest) (*pb.OrderTraceResponse, error) {
	if req == nil {
		return nil, errors.NewCode(errcode.ErrValidation, "order trace request is required")
	}
	service, err := os.paymentService()
	if err != nil {
		return nil, err
	}
	trace, err := service.GetOrderTrace(ctx, do.OrderTraceLookup{
		OrderSN:       req.GetOrderSn(),
		TradeNo:       req.GetTradeNo(),
		CorrelationID: req.GetCorrelationId(),
	})
	if err != nil {
		return nil, err
	}
	resp := &pb.OrderTraceResponse{
		MatchedBy:    trace.MatchedBy,
		MatchedValue: trace.MatchedValue,
		Order:        orderToDetailResponse(trace.Order),
		StatusLogs:   statusLogsToResponse(trace.StatusLogs),
		PaymentEvents: &pb.PaymentEventListResponse{
			Total:         int32(len(trace.PaymentEvents)),
			Data:          make([]*pb.PaymentEventRecord, 0, len(trace.PaymentEvents)),
			MismatchCount: int32(countTraceMismatches(trace.PaymentEvents)),
		},
		Refunds: make([]*pb.RefundTraceRecord, 0, len(trace.Refunds)),
	}
	for _, item := range trace.PaymentEvents {
		var completed int64
		if item.CompletedAt != nil {
			completed = item.CompletedAt.Unix()
		}
		resp.PaymentEvents.Data = append(resp.PaymentEvents.Data, &pb.PaymentEventRecord{
			Id:                int64(item.ID),
			Provider:          item.Provider,
			EventId:           item.EventID,
			OrderSn:           item.OrderSN,
			TradeNo:           item.TradeNo,
			EventType:         item.EventType,
			OrderAmountFen:    item.OrderAmountFen,
			ProviderAmountFen: item.ProviderAmountFen,
			RefundAmountFen:   item.RefundAmountFen,
			Status:            item.Status,
			ErrorDetail:       item.ErrorDetail,
			ReceivedAt:        item.ReceivedAt.Unix(),
			CompletedAt:       completed,
		})
	}
	for _, item := range trace.Refunds {
		record := &pb.RefundTraceRecord{
			RefundRequestId:  int64(item.RefundRequest.ID),
			OrderSn:          item.RefundRequest.OrderSN,
			ActorUserId:      item.RefundRequest.ActorUserID,
			AmountFen:        item.RefundRequest.AmountFen,
			Reason:           item.RefundRequest.Reason,
			Status:           item.RefundRequest.Status,
			Provider:         item.RefundRequest.Provider,
			ProviderRefundId: item.RefundRequest.ProviderRefundID,
			FailureReason:    item.RefundRequest.FailureReason,
			CorrelationId:    item.RefundRequest.CorrelationID,
			CreatedAt:        item.RefundRequest.CreatedAt.Unix(),
			UpdatedAt:        item.RefundRequest.UpdatedAt.Unix(),
		}
		if item.RefundOutbox != nil {
			record.RefundJobId = int64(item.RefundOutbox.ID)
			record.RefundJobStatus = item.RefundOutbox.Status
			record.RefundJobAttempts = int32(item.RefundOutbox.Attempts)
			record.RefundJobAvailableAt = item.RefundOutbox.AvailableAt.Unix()
			if item.RefundOutbox.LockedAt != nil {
				record.RefundJobLockedAt = item.RefundOutbox.LockedAt.Unix()
			}
			record.RefundJobLastError = item.RefundOutbox.LastError
		}
		resp.Refunds = append(resp.Refunds, record)
	}
	return resp, nil
}

func (os *orderServer) paymentService() (paymentService, error) {
	service, ok := os.srv.Orders().(paymentService)
	if !ok {
		return nil, errors.NewCode(errcode.ErrDatabase, "payment service is not configured")
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

func optionalUnixTime(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	parsed := time.Unix(value, 0).UTC()
	return &parsed
}

func orderToDetailResponse(order *do.OrderInfoDO) *pb.OrderInfoDetailResponse {
	if order == nil {
		return nil
	}
	resp := &pb.OrderInfoDetailResponse{
		OrderInfo: orderToResponse(&dto.OrderDTO{OrderInfoDO: *order}),
		Goods:     orderGoodsToResponse(order.OrderGoods),
	}
	return resp
}

func statusLogsToResponse(entries []*do.OrderStatusLogDO) *pb.OrderStatusLogListResponse {
	resp := &pb.OrderStatusLogListResponse{
		Total: int32(len(entries)),
		Data:  make([]*pb.OrderStatusLogResponse, 0, len(entries)),
	}
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		resp.Data = append(resp.Data, &pb.OrderStatusLogResponse{
			Id:         entry.ID,
			OrderId:    entry.OrderID,
			OrderSn:    entry.OrderSn,
			FromStatus: entry.FromStatus,
			ToStatus:   entry.ToStatus,
			Reason:     entry.Reason,
			Source:     entry.Source,
			Operator:   entry.Operator,
			AddTime:    entry.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return resp
}

func countTraceMismatches(items []do.PaymentEventDO) int {
	total := 0
	for _, item := range items {
		if (item.EventType != "refund_succeeded" && item.ProviderAmountFen != item.OrderAmountFen) ||
			(item.EventType == "refund_succeeded" && (item.RefundAmountFen != item.ProviderAmountFen || item.RefundAmountFen > item.OrderAmountFen)) {
			total++
		}
	}
	return total
}
