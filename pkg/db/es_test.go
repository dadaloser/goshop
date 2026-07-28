package db

import (
	"testing"
	"time"
)

func TestNewEsClientWithProductionOptions(t *testing.T) {
	client, err := NewEsClient(&EsOptions{
		Host:                  "127.0.0.1",
		Port:                  "9200",
		Scheme:                "https",
		Username:              "user",
		Password:              "password",
		Timeout:               time.Second,
		UseSSL:                true,
		SSLInsecureSkipVerify: false,
		DisableHealthcheck:    true,
	})
	if err != nil {
		t.Fatalf("NewEsClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewEsClient() returned nil client")
	}
}

func TestNewEsClientRejectsNilOptions(t *testing.T) {
	client, err := NewEsClient(nil)
	if err == nil {
		t.Fatal("NewEsClient(nil) error = nil, want error")
	}
	if client != nil {
		t.Fatalf("NewEsClient(nil) client = %v, want nil", client)
	}
}

func TestBuildESURL(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		host   string
		port   string
		want   string
	}{
		{name: "hostname", scheme: "http", host: "es.internal", port: "9200", want: "http://es.internal:9200/"},
		{name: "ipv6", scheme: "https", host: "2001:db8::1", port: "9200", want: "https://[2001:db8::1]:9200/"},
		{name: "ipv6 without port", scheme: "https", host: "2001:db8::1", want: "https://[2001:db8::1]/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildESURL(tt.scheme, tt.host, tt.port); got != tt.want {
				t.Fatalf("buildESURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
