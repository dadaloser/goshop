package service

import "strings"

const accountDeletionBlockedReason = "存在未完成订单或退款，暂不能注销"

// AccountDeletionCanProceed is deliberately small and deterministic so the
// NATS consumer can make its decision inside one database transaction.
// This service currently has no after-sales domain table; when it is added, its
// active statuses must be included in the same transaction before confirming.
func AccountDeletionCanProceed(orderStatuses, refundStatuses []string) (bool, string) {
	for _, status := range orderStatuses {
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case OrderStatusTradeClosed, OrderStatusTradeFinished, OrderStatusRefunded, OrderStatusRefundFailed:
			continue
		default:
			return false, accountDeletionBlockedReason
		}
	}
	for _, status := range refundStatuses {
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "REFUNDED", "FAILED", "CLOSED", "CANCELLED":
			continue
		default:
			return false, accountDeletionBlockedReason
		}
	}
	return true, ""
}
