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

func TestRedisUsernameIsPropagatedToAllClientOptions(t *testing.T) {
	const username = "app-user"

	client := NewRedisClusterPool(false, &Config{
		Host:     "127.0.0.1",
		Port:     6379,
		Username: username,
	})
	defer client.Close()

	redisClient, ok := client.(*redis.Client)
	if !ok {
		t.Fatalf("client type = %T, want *redis.Client", client)
	}
	if got := redisClient.Options().Username; got != username {
		t.Fatalf("simple client username = %q, want %q", got, username)
	}

	opts := &RedisOpts{Username: username}
	if got := opts.cluster().Username; got != username {
		t.Fatalf("cluster client username = %q, want %q", got, username)
	}
	if got := opts.failover().Username; got != username {
		t.Fatalf("failover client username = %q, want %q", got, username)
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

func TestRedisClusterCleanKey(t *testing.T) {
	cluster := RedisCluster{KeyPrefix: "goshop:"}
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "removes leading prefix", key: "goshop:session", want: "session"},
		{name: "preserves embedded prefix", key: "session:goshop:1", want: "session:goshop:1"},
		{name: "preserves key without prefix", key: "session", want: "session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cluster.cleanKey(tt.key); got != tt.want {
				t.Errorf("cleanKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}
