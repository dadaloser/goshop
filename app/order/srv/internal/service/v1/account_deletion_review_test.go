package service

import "testing"

func TestAccountDeletionCanProceed(t *testing.T) {
	tests := []struct {
		name    string
		orders  []string
		refunds []string
		want    bool
	}{
		{name: "terminal orders and refunds", orders: []string{OrderStatusTradeClosed, OrderStatusRefunded}, refunds: []string{"REFUNDED", "FAILED"}, want: true},
		{name: "pending order blocks", orders: []string{OrderStatusWaitBuyerPay}, want: false},
		{name: "pending refund blocks", orders: []string{OrderStatusTradeFinished}, refunds: []string{"PENDING"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := AccountDeletionCanProceed(tt.orders, tt.refunds)
			if got != tt.want {
				t.Fatalf("AccountDeletionCanProceed() = %v, want %v", got, tt.want)
			}
		})
	}
}
