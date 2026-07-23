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
	CallbackSecret  string        `json:"-" mapstructure:"callback-secret"`
	CallbackMaxSkew time.Duration `json:"callback-max-skew" mapstructure:"callback-max-skew"`
}

func NewPaymentOptions() *PaymentOptions {
	return &PaymentOptions{Provider: "mock", CheckoutBaseURL: "https://payments.example.invalid/checkout", CallbackMaxSkew: 5 * time.Minute}
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
	return errs
}
func (o *PaymentOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.Enabled, "payment.enabled", o.Enabled, "enable payment provider")
	fs.StringVar(&o.Provider, "payment.provider", o.Provider, "payment provider")
	fs.StringVar(&o.CheckoutBaseURL, "payment.checkout-base-url", o.CheckoutBaseURL, "payment checkout base URL")
	fs.StringVar(&o.CallbackSecret, "payment.callback-secret", o.CallbackSecret, "payment callback HMAC secret")
	fs.DurationVar(&o.CallbackMaxSkew, "payment.callback-max-skew", o.CallbackMaxSkew, "maximum signed callback clock skew")
}
