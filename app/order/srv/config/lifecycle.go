package config

import (
	"fmt"
	service "goshop/app/order/srv/internal/service/v1"
	"time"

	"github.com/spf13/pflag"
)

type LifecycleOptions struct {
	PollInterval       time.Duration `json:"poll-interval" mapstructure:"poll-interval"`
	TimeoutCloseAfter  time.Duration `json:"timeout-close-after" mapstructure:"timeout-close-after"`
	FinishAfterPayment time.Duration `json:"finish-after-payment" mapstructure:"finish-after-payment"`
	BatchSize          int           `json:"batch-size" mapstructure:"batch-size"`
}

func NewLifecycleOptions() *LifecycleOptions {
	return &LifecycleOptions{
		PollInterval:       5 * time.Second,
		TimeoutCloseAfter:  30 * time.Minute,
		FinishAfterPayment: 7 * 24 * time.Hour,
		BatchSize:          20,
	}
}

func (o *LifecycleOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil || o == nil {
		return
	}

	fs.DurationVar(&o.PollInterval, "lifecycle.poll-interval", o.PollInterval, "Order lifecycle worker poll interval.")
	fs.DurationVar(&o.TimeoutCloseAfter, "lifecycle.timeout-close-after", o.TimeoutCloseAfter, "Auto-close unpaid orders older than this duration.")
	fs.DurationVar(&o.FinishAfterPayment, "lifecycle.finish-after-payment", o.FinishAfterPayment, "Auto-finish paid orders older than this duration.")
	fs.IntVar(&o.BatchSize, "lifecycle.batch-size", o.BatchSize, "Maximum orders processed per lifecycle sweep.")
}

func (o *LifecycleOptions) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	if o.PollInterval <= 0 {
		errs = append(errs, fmt.Errorf("lifecycle.poll-interval must be positive"))
	}
	if o.TimeoutCloseAfter <= 0 {
		errs = append(errs, fmt.Errorf("lifecycle.timeout-close-after must be positive"))
	}
	if o.FinishAfterPayment <= 0 {
		errs = append(errs, fmt.Errorf("lifecycle.finish-after-payment must be positive"))
	}
	if o.BatchSize <= 0 {
		errs = append(errs, fmt.Errorf("lifecycle.batch-size must be positive"))
	}
	return errs
}

func (o *LifecycleOptions) ToServiceConfig() service.LifecycleConfig {
	if o == nil {
		o = NewLifecycleOptions()
	}
	return service.LifecycleConfig{
		PollInterval:       o.PollInterval,
		TimeoutCloseAfter:  o.TimeoutCloseAfter,
		FinishAfterPayment: o.FinishAfterPayment,
		BatchSize:          o.BatchSize,
	}
}
