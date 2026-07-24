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
	refund       RefundResponse
	refundErr    error
	transactions []Transaction
}

func (f *fakePaymentProvider) Initiate(context.Context, InitiateRequest) (InitiateResponse, error) {
	return InitiateResponse{}, nil
}
func (f *fakePaymentProvider) Refund(context.Context, RefundRequest) (RefundResponse, error) {
	return f.refund, f.refundErr
}
func (f *fakePaymentProvider) ListTransactions(context.Context, time.Time, time.Time) ([]Transaction, error) {
	return f.transactions, nil
}

type fakeOrderClient struct {
	opb.OrderClient
	jobs       []*opb.RefundJob
	completed  []*opb.CompleteRefundJobRequest
	reconciled *opb.ReconcilePaymentsRequest
}

func (f *fakeOrderClient) ClaimRefundJobs(context.Context, *opb.ClaimRefundJobsRequest, ...grpc.CallOption) (*opb.ClaimRefundJobsResponse, error) {
	return &opb.ClaimRefundJobsResponse{Jobs: f.jobs}, nil
}
func (f *fakeOrderClient) CompleteRefundJob(_ context.Context, request *opb.CompleteRefundJobRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	f.completed = append(f.completed, request)
	return &emptypb.Empty{}, nil
}
func (f *fakeOrderClient) ReconcilePayments(_ context.Context, request *opb.ReconcilePaymentsRequest, _ ...grpc.CallOption) (*opb.ReconcilePaymentsResponse, error) {
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
