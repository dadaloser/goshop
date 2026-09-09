package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// AccountDeletionEventOptions controls asynchronous deletion-event delivery.
// Leaving URL blank disables only the dispatcher, never the user RPC service.
type AccountDeletionEventOptions struct {
	URL            string        `json:"url" mapstructure:"url"`
	PollInterval   time.Duration `json:"poll-interval" mapstructure:"poll-interval"`
	SweepTimeout   time.Duration `json:"sweep-timeout" mapstructure:"sweep-timeout"`
	PublishTimeout time.Duration `json:"publish-timeout" mapstructure:"publish-timeout"`
	BatchSize      int           `json:"batch-size" mapstructure:"batch-size"`
	MaxRetries     int           `json:"max-retries" mapstructure:"max-retries"`
}

func NewAccountDeletionEventOptions() *AccountDeletionEventOptions {
	return &AccountDeletionEventOptions{PollInterval: 2 * time.Second, SweepTimeout: 2 * time.Minute, PublishTimeout: 5 * time.Second, BatchSize: 50, MaxRetries: 20}
}

func (o *AccountDeletionEventOptions) Enabled() bool {
	return o != nil && strings.TrimSpace(o.URL) != ""
}

func (o *AccountDeletionEventOptions) Validate() []error {
	if o == nil || !o.Enabled() {
		return nil
	}
	var errs []error
	if o.PollInterval <= 0 {
		errs = append(errs, fmt.Errorf("account-deletion-events.poll-interval must be positive"))
	}
	if o.SweepTimeout <= 0 {
		errs = append(errs, fmt.Errorf("account-deletion-events.sweep-timeout must be positive"))
	}
	if o.PublishTimeout <= 0 {
		errs = append(errs, fmt.Errorf("account-deletion-events.publish-timeout must be positive"))
	}
	if o.BatchSize <= 0 {
		errs = append(errs, fmt.Errorf("account-deletion-events.batch-size must be positive"))
	}
	if o.MaxRetries <= 0 {
		errs = append(errs, fmt.Errorf("account-deletion-events.max-retries must be positive"))
	}
	return errs
}

func (o *AccountDeletionEventOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.URL, "account-deletion-events.url", o.URL, "NATS JetStream URL; empty disables the account deletion event dispatcher")
	fs.DurationVar(&o.PollInterval, "account-deletion-events.poll-interval", o.PollInterval, "account deletion outbox polling interval")
	fs.DurationVar(&o.SweepTimeout, "account-deletion-events.sweep-timeout", o.SweepTimeout, "maximum duration of one account deletion outbox sweep")
	fs.DurationVar(&o.PublishTimeout, "account-deletion-events.publish-timeout", o.PublishTimeout, "NATS JetStream publish timeout")
	fs.IntVar(&o.BatchSize, "account-deletion-events.batch-size", o.BatchSize, "maximum account deletion events per sweep")
	fs.IntVar(&o.MaxRetries, "account-deletion-events.max-retries", o.MaxRetries, "maximum account deletion event delivery retries")
}
