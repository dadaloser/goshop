package config

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

type OutboxOptions struct {
	PollInterval time.Duration `json:"poll-interval" mapstructure:"poll-interval"`
	SweepTimeout time.Duration `json:"sweep-timeout" mapstructure:"sweep-timeout"`
	BatchSize    int           `json:"batch-size" mapstructure:"batch-size"`
}

func NewOutboxOptions() *OutboxOptions {
	return &OutboxOptions{
		PollInterval: 2 * time.Second,
		SweepTimeout: 2 * time.Minute,
		BatchSize:    50,
	}
}

func (o *OutboxOptions) AddFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&o.PollInterval, "outbox.poll-interval", o.PollInterval, "review rating outbox polling interval")
	fs.DurationVar(&o.SweepTimeout, "outbox.sweep-timeout", o.SweepTimeout, "maximum duration of one review rating outbox sweep")
	fs.IntVar(&o.BatchSize, "outbox.batch-size", o.BatchSize, "maximum review rating outbox events processed per sweep")
}

func (o *OutboxOptions) Validate() []error {
	var errs []error
	if o.PollInterval <= 0 {
		errs = append(errs, fmt.Errorf("outbox.poll-interval must be positive"))
	}
	if o.SweepTimeout <= 0 {
		errs = append(errs, fmt.Errorf("outbox.sweep-timeout must be positive"))
	}
	if o.BatchSize <= 0 {
		errs = append(errs, fmt.Errorf("outbox.batch-size must be positive"))
	}
	return errs
}

func (o *OutboxOptions) ToWorkerConfig() WorkerConfig {
	cfg := WorkerConfig{
		PollInterval: o.PollInterval,
		SweepTimeout: o.SweepTimeout,
		BatchSize:    o.BatchSize,
	}
	return cfg.normalize()
}

type WorkerConfig struct {
	PollInterval time.Duration
	SweepTimeout time.Duration
	BatchSize    int
}

func (c WorkerConfig) normalize() WorkerConfig {
	if c.PollInterval <= 0 {
		c.PollInterval = 2 * time.Second
	}
	if c.SweepTimeout <= 0 {
		c.SweepTimeout = 2 * time.Minute
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	return c
}
