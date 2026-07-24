package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	opb "goshop/api/order/v1"
	"goshop/app/pkg/options"
)

type fakePaymentProvider struct {
	initiate     InitiateResponse
	initiateErr  error
	refund       RefundResponse
	refundErr    error
	transactions []Transaction
	listErr      error
}

func (f *fakePaymentProvider) Initiate(context.Context, InitiateRequest) (InitiateResponse, error) {
	return f.initiate, f.initiateErr
}
func (f *fakePaymentProvider) Refund(context.Context, RefundRequest) (RefundResponse, error) {
	return f.refund, f.refundErr
}
func (f *fakePaymentProvider) ListTransactions(context.Context, time.Time, time.Time) ([]Transaction, error) {
	return f.transactions, f.listErr
}

type fakeOrderClient struct {
	opb.OrderClient
	jobs         []*opb.RefundJob
	completed    []*opb.CompleteRefundJobRequest
	reconciled   *opb.ReconcilePaymentsRequest
	claimErr     error
	completeErr  error
	reconcileErr error
}

func (f *fakeOrderClient) ClaimRefundJobs(context.Context, *opb.ClaimRefundJobsRequest, ...grpc.CallOption) (*opb.ClaimRefundJobsResponse, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	return &opb.ClaimRefundJobsResponse{Jobs: f.jobs}, nil
}
func (f *fakeOrderClient) CompleteRefundJob(_ context.Context, request *opb.CompleteRefundJobRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.completeErr != nil {
		return nil, f.completeErr
	}
	f.completed = append(f.completed, request)
	return &emptypb.Empty{}, nil
}
func (f *fakeOrderClient) ReconcilePayments(_ context.Context, request *opb.ReconcilePaymentsRequest, _ ...grpc.CallOption) (*opb.ReconcilePaymentsResponse, error) {
	if f.reconcileErr != nil {
		return nil, f.reconcileErr
	}
	f.reconciled = request
	return &opb.ReconcilePaymentsResponse{}, nil
}

func TestWorkerCompletesRefundSuccessAndFailure(t *testing.T) {
	for _, tt := range []struct {
		name     string
		provider *fakePaymentProvider
		success  bool
	}{
		{"success", &fakePaymentProvider{refund: RefundResponse{ProviderRefundID: "provider-1", Status: "accepted"}}, true},
		{"failure", &fakePaymentProvider{refundErr: errors.New("timeout")}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			orders := &fakeOrderClient{jobs: []*opb.RefundJob{{Id: 1, CorrelationId: "request-1", OrderSn: "order-1", AmountFen: 100}}}
			worker := NewWorker(orders, tt.provider, &options.PaymentOptions{Provider: "mock", WorkerBatchSize: 10, MaxAttempts: 3, RequestTimeout: time.Second})
			if err := worker.processRefundBatch(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(orders.completed) != 1 || orders.completed[0].GetSuccess() != tt.success {
				t.Fatalf("completed=%+v", orders.completed)
			}
		})
	}
}

func TestWorkerPersistsProviderReconciliation(t *testing.T) {
	orders := &fakeOrderClient{}
	provider := &fakePaymentProvider{transactions: []Transaction{{EventID: "event-1", OrderSN: "order-1", AmountFen: 100, OccurredAt: time.Unix(100, 0)}}}
	worker := NewWorker(orders, provider, &options.PaymentOptions{Provider: "mock", RequestTimeout: time.Second})
	if err := worker.reconcileWindow(context.Background(), time.Unix(0, 0), time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if orders.reconciled == nil || len(orders.reconciled.GetTransactions()) != 1 || orders.reconciled.GetTransactions()[0].GetEventId() != "event-1" {
		t.Fatalf("reconciliation=%+v", orders.reconciled)
	}
}

func TestWorkerRunNoopsWithoutEnabledDependencies(t *testing.T) {
	tests := []*Worker{
		nil,
		NewWorker(nil, &fakePaymentProvider{}, &options.PaymentOptions{Enabled: true}),
		NewWorker(&fakeOrderClient{}, nil, &options.PaymentOptions{Enabled: true}),
		NewWorker(&fakeOrderClient{}, &fakePaymentProvider{}, &options.PaymentOptions{}),
	}
	for _, worker := range tests {
		if err := worker.Run(context.Background()); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}
}

func TestWorkerPropagatesRefundAndReconciliationErrors(t *testing.T) {
	worker := NewWorker(&fakeOrderClient{claimErr: errors.New("claim failed")}, &fakePaymentProvider{}, &options.PaymentOptions{Enabled: true, WorkerBatchSize: 10, MaxAttempts: 3, RequestTimeout: time.Second})
	if err := worker.processRefundBatch(context.Background()); err == nil {
		t.Fatal("processRefundBatch() error=nil")
	}

	worker = NewWorker(&fakeOrderClient{completeErr: errors.New("complete failed"), jobs: []*opb.RefundJob{{Id: 1, CorrelationId: "request-1", OrderSn: "order-1", AmountFen: 100}}}, &fakePaymentProvider{refund: RefundResponse{ProviderRefundID: "provider-1", Status: "accepted"}}, &options.PaymentOptions{Enabled: true, Provider: "mock", WorkerBatchSize: 10, MaxAttempts: 3, RequestTimeout: time.Second})
	if err := worker.processRefundBatch(context.Background()); err == nil {
		t.Fatal("processRefundBatch(complete error) error=nil")
	}

	worker = NewWorker(&fakeOrderClient{}, &fakePaymentProvider{listErr: errors.New("provider down")}, &options.PaymentOptions{Enabled: true, Provider: "mock", RequestTimeout: time.Second})
	if err := worker.reconcileWindow(context.Background(), time.Unix(0, 0), time.Unix(200, 0)); err == nil {
		t.Fatal("reconcileWindow() error=nil")
	}
}
