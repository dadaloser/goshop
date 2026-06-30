package storage

import (
	"crypto/tls"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisTLSConfigUsesMinimumTLS12(t *testing.T) {
	client := NewRedisClusterPool(false, &Config{
		Host:                  "127.0.0.1",
		Port:                  6379,
		UseSSL:                true,
		SSLInsecureSkipVerify: true,
	})
	defer client.Close()

	redisClient, ok := client.(*redis.Client)
	if !ok {
		t.Fatalf("client type = %T, want *redis.Client", client)
	}
	opts := redisClient.Options()
	if opts.TLSConfig == nil {
		t.Fatal("TLSConfig = nil, want configured TLS")
	}
	if opts.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("TLSConfig.MinVersion = %v, want TLS 1.2", opts.TLSConfig.MinVersion)
	}
	if !opts.TLSConfig.InsecureSkipVerify {
		t.Fatal("TLSConfig.InsecureSkipVerify = false, want true from config")
	}
}

func TestRedisKeyRedaction(t *testing.T) {
	raw := "mobile:13800138000"

	got := redactedRedisKey(raw)

	if got == "" {
		t.Fatal("redactedRedisKey() returned empty hash")
	}
	if got == raw {
		t.Fatal("redactedRedisKey() returned raw key")
	}
}

func TestRedisValueRedaction(t *testing.T) {
	if got := redactedRedisValue("secret-value"); got != "<redacted len=12>" {
		t.Fatalf("redactedRedisValue() = %q, want length marker", got)
	}
	if got := redactedRedisValue(""); got != "" {
		t.Fatalf("redactedRedisValue(empty) = %q, want empty", got)
	}
}
