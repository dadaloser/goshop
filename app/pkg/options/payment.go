package options

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

type PaymentOptions struct {
	Enabled         bool          `json:"enabled" mapstructure:"enabled"`
	Provider        string        `json:"provider" mapstructure:"provider"`
	CheckoutBaseURL string        `json:"checkout-base-url" mapstructure:"checkout-base-url"`
	RefundURL       string        `json:"refund-url" mapstructure:"refund-url"`
	ReconcileURL    string        `json:"reconcile-url" mapstructure:"reconcile-url"`
	CallbackSecret  string        `json:"-" mapstructure:"callback-secret"`
	CallbackMaxSkew time.Duration `json:"callback-max-skew" mapstructure:"callback-max-skew"`
	RequestTimeout  time.Duration `json:"request-timeout" mapstructure:"request-timeout"`
	WorkerInterval  time.Duration `json:"worker-interval" mapstructure:"worker-interval"`
	WorkerBatchSize int           `json:"worker-batch-size" mapstructure:"worker-batch-size"`
	MaxAttempts     int           `json:"max-attempts" mapstructure:"max-attempts"`
}

func NewPaymentOptions() *PaymentOptions {
	return &PaymentOptions{Provider: "mock", CheckoutBaseURL: "https://payments.example.invalid/checkout", CallbackMaxSkew: 5 * time.Minute, RequestTimeout: 10 * time.Second, WorkerInterval: 5 * time.Second, WorkerBatchSize: 20, MaxAttempts: 8}
}
func (o *PaymentOptions) Validate() []error {
	errs := []error{}
	if o == nil || !o.Enabled {
		return errs
	}
	if o.Provider == "" || o.CheckoutBaseURL == "" || o.CallbackSecret == "" {
		errs = append(errs, errors.New("payment provider, checkout base URL, and callback secret are required"))
	}
	if o.CallbackMaxSkew <= 0 {
		errs = append(errs, errors.New("payment callback max skew must be positive"))
	}
	if o.RefundURL == "" || o.ReconcileURL == "" {
		errs = append(errs, errors.New("payment refund and reconciliation URLs are required"))
	}
	if o.RequestTimeout <= 0 || o.WorkerInterval <= 0 || o.WorkerBatchSize <= 0 || o.MaxAttempts <= 0 {
		errs = append(errs, errors.New("payment worker settings must be positive"))
	}
	return errs
}
func (o *PaymentOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.Enabled, "payment.enabled", o.Enabled, "enable payment provider")
	fs.StringVar(&o.Provider, "payment.provider", o.Provider, "payment provider")
	fs.StringVar(&o.CheckoutBaseURL, "payment.checkout-base-url", o.CheckoutBaseURL, "payment checkout base URL")
	fs.StringVar(&o.RefundURL, "payment.refund-url", o.RefundURL, "payment provider refund API URL")
	fs.StringVar(&o.ReconcileURL, "payment.reconcile-url", o.ReconcileURL, "payment provider reconciliation API URL")
	fs.StringVar(&o.CallbackSecret, "payment.callback-secret", o.CallbackSecret, "payment callback HMAC secret")
	fs.DurationVar(&o.CallbackMaxSkew, "payment.callback-max-skew", o.CallbackMaxSkew, "maximum signed callback clock skew")
	fs.DurationVar(&o.RequestTimeout, "payment.request-timeout", o.RequestTimeout, "payment provider request timeout")
	fs.DurationVar(&o.WorkerInterval, "payment.worker-interval", o.WorkerInterval, "payment background worker interval")
	fs.IntVar(&o.WorkerBatchSize, "payment.worker-batch-size", o.WorkerBatchSize, "payment background worker batch size")
	fs.IntVar(&o.MaxAttempts, "payment.max-attempts", o.MaxAttempts, "payment outbox maximum attempts")
}
