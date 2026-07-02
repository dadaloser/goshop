package options

import (
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/spf13/pflag"
)

type DtmOptions struct {
	GrpcServer string `mapstructure:"grpc" json:"grpc,omitempty"`
	HttpServer string `mapstructure:"http" json:"http,omitempty"`
}

func NewDtmOptions() *DtmOptions {
	return &DtmOptions{
		HttpServer: "http://127.0.0.1:36789/api/dtmsvr",
		GrpcServer: "127.0.0.1:36790",
	}
}

func (o *DtmOptions) Validate() []error {
	var errs []error
	if o.GrpcServer == "" {
		errs = append(errs, errors.New("dtm.grpc is required"))
	} else if _, _, err := net.SplitHostPort(o.GrpcServer); err != nil {
		errs = append(errs, fmt.Errorf("dtm.grpc must be host:port, got %q: %w", o.GrpcServer, err))
	}
	if o.HttpServer == "" {
		errs = append(errs, errors.New("dtm.http is required"))
	} else if parsed, err := url.Parse(o.HttpServer); err != nil {
		errs = append(errs, fmt.Errorf("dtm.http must be a valid URL: %w", err))
	} else if parsed.Scheme != "http" && parsed.Scheme != "https" {
		errs = append(errs, fmt.Errorf("dtm.http must use http or https, got %q", parsed.Scheme))
	} else if parsed.Host == "" {
		errs = append(errs, errors.New("dtm.http must include host"))
	}
	return errs
}

func (o *DtmOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.GrpcServer, "dtm.grpc", o.GrpcServer, "")
	fs.StringVar(&o.HttpServer, "dtm.http", o.HttpServer, "")
}
