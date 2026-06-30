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
