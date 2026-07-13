package resilience

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/spf13/pflag"
)

const (
	defaultTimeout          = 2 * time.Second
	defaultMaxConcurrency   = 100
	defaultErrorRatio       = 0.5
	defaultMinRequestAmount = 20
	defaultStatInterval     = 10 * time.Second
	defaultRecoveryTimeout  = 30 * time.Second
)

// Options configures timeout, isolation, and circuit breaking for an external dependency.
// Durations must fit Sentinel's millisecond rule fields and all thresholds must be positive.
type Options struct {
	Enabled          bool          `json:"enabled" mapstructure:"enabled"`
	Timeout          time.Duration `json:"timeout" mapstructure:"timeout"`
	MaxConcurrency   uint32        `json:"max-concurrency" mapstructure:"max-concurrency"`
	ErrorRatio       float64       `json:"error-ratio" mapstructure:"error-ratio"`
	MinRequestAmount uint64        `json:"min-request-amount" mapstructure:"min-request-amount"`
	StatInterval     time.Duration `json:"stat-interval" mapstructure:"stat-interval"`
	RecoveryTimeout  time.Duration `json:"recovery-timeout" mapstructure:"recovery-timeout"`
}

// NewOptions returns production-oriented defaults for dependency protection.
func NewOptions() *Options {
	return &Options{
		Enabled:          true,
		Timeout:          defaultTimeout,
		MaxConcurrency:   defaultMaxConcurrency,
		ErrorRatio:       defaultErrorRatio,
		MinRequestAmount: defaultMinRequestAmount,
		StatInterval:     defaultStatInterval,
		RecoveryTimeout:  defaultRecoveryTimeout,
	}
}

// Validate checks whether the options can be translated into Sentinel rules.
func (o *Options) Validate() []error {
	if o == nil {
		return []error{errors.New("resilience options are required")}
	}

	errs := []error{}
	if o.Timeout <= 0 {
		errs = append(errs, errors.New("resilience.timeout must be positive"))
	}
	if !o.Enabled {
		return errs
	}
	if o.MaxConcurrency == 0 {
		errs = append(errs, errors.New("resilience.max-concurrency must be positive"))
	}
	if o.ErrorRatio <= 0 || o.ErrorRatio > 1 {
		errs = append(errs, fmt.Errorf("resilience.error-ratio must be within (0, 1], got %v", o.ErrorRatio))
	}
	if o.MinRequestAmount == 0 {
		errs = append(errs, errors.New("resilience.min-request-amount must be positive"))
	}
	if err := validateSentinelDuration("stat-interval", o.StatInterval); err != nil {
		errs = append(errs, err)
	}
	if err := validateSentinelDuration("recovery-timeout", o.RecoveryTimeout); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// AddFlags adds dependency resilience flags using prefix, for example "redis.resilience".
func (o *Options) AddFlags(fs *pflag.FlagSet, prefix string) {
	fs.BoolVar(&o.Enabled, prefix+".enabled", o.Enabled, "Enable dependency circuit breaking and isolation.")
	fs.DurationVar(&o.Timeout, prefix+".timeout", o.Timeout, "Maximum duration of one dependency operation.")
	fs.Uint32Var(
		&o.MaxConcurrency,
		prefix+".max-concurrency",
		o.MaxConcurrency,
		"Maximum concurrent dependency operations.",
	)
	fs.Float64Var(&o.ErrorRatio, prefix+".error-ratio", o.ErrorRatio, "Error ratio that opens the dependency circuit.")
	fs.Uint64Var(
		&o.MinRequestAmount,
		prefix+".min-request-amount",
		o.MinRequestAmount,
		"Minimum requests in the statistic window before opening the circuit.",
	)
	fs.DurationVar(&o.StatInterval, prefix+".stat-interval", o.StatInterval, "Circuit breaker statistic window.")
	fs.DurationVar(
		&o.RecoveryTimeout,
		prefix+".recovery-timeout",
		o.RecoveryTimeout,
		"Open-circuit duration before a half-open recovery probe.",
	)
}

func validateSentinelDuration(name string, value time.Duration) error {
	if value < time.Millisecond {
		return fmt.Errorf("resilience.%s must be at least 1ms", name)
	}
	if value/time.Millisecond > math.MaxUint32 {
		return fmt.Errorf("resilience.%s exceeds Sentinel's maximum duration", name)
	}
	return nil
}
