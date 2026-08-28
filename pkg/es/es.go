package es

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/olivere/elastic/v7"
)

type EsOptions struct {
	Host                  string
	Port                  string
	Scheme                string
	Username              string
	Password              string
	Timeout               time.Duration
	UseSSL                bool
	SSLInsecureSkipVerify bool
	DisableHealthcheck    bool
}

func NewEsClient(opts *EsOptions) (*elastic.Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("elasticsearch options are required")
	}
	options := *opts
	opts = &options
	if opts.Scheme == "" {
		opts.Scheme = "http"
	}
	if opts.UseSSL {
		opts.Scheme = "https"
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if opts.UseSSL {
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: opts.SSLInsecureSkipVerify,
		}
	}

	client := &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
	}

	clientOptions := []elastic.ClientOptionFunc{
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
		elastic.SetSniff(false),
		elastic.SetHttpClient(client),
		elastic.SetURL(buildESURL(opts.Scheme, opts.Host, opts.Port)),
	}
	if opts.DisableHealthcheck {
		clientOptions = append(clientOptions, elastic.SetHealthcheck(false))
	}
	if opts.Username != "" || opts.Password != "" {
		clientOptions = append(clientOptions, elastic.SetBasicAuth(opts.Username, opts.Password))
	}

	esClient, err := elastic.NewClient(
		clientOptions...,
	)
	if err != nil {
		return nil, err
	}
	return esClient, nil
}

func buildESURL(scheme, host, port string) string {
	address := host
	if port != "" {
		address = net.JoinHostPort(host, port)
	} else if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		address = "[" + host + "]"
	}

	return (&url.URL{Scheme: scheme, Host: address, Path: "/"}).String()
}
