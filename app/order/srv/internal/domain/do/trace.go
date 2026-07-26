package do

type OrderTraceLookup struct {
	OrderSN       string
	TradeNo       string
	CorrelationID string
}

type RefundTraceRecord struct {
	RefundRequest RefundRequestDO
	RefundOutbox  *RefundOutboxDO
}

type OrderTrace struct {
	MatchedBy     string
	MatchedValue  string
	Order         *OrderInfoDO
	StatusLogs    []*OrderStatusLogDO
	PaymentEvents []PaymentEventDO
	Refunds       []RefundTraceRecord
}
