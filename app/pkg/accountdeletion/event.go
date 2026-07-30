// Package accountdeletion defines the cross-service account-deletion contract.
package accountdeletion

import "time"

const (
	SubjectRequested = "goshop.account.deletion.requested"
	SubjectConfirmed = "goshop.account.deletion.confirmed"
	SubjectRejected  = "goshop.account.deletion.rejected"
)

// Requested is emitted after a customer account has entered deletion_pending.
// EventID is stable across Outbox retries and is the consumer idempotency key.
type Requested struct {
	EventID     string    `json:"event_id"`
	UserID      uint64    `json:"user_id"`
	RequestedAt time.Time `json:"requested_at"`
}

// Decision is emitted by the order service after it durably records an
// idempotent review of a Requested event.
type Decision struct {
	EventID   string    `json:"event_id"`
	RequestID string    `json:"request_id"`
	UserID    uint64    `json:"user_id"`
	Confirmed bool      `json:"confirmed"`
	Reason    string    `json:"reason,omitempty"`
	DecidedAt time.Time `json:"decided_at"`
}
