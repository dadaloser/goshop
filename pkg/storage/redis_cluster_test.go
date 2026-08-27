package storage

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"testing"
	"time"

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

func TestSortedSetOperationsReturnRedisDownWhenClientIsUnavailable(t *testing.T) {
	cluster := NewRedisCluster(nil)
	if _, _, err := cluster.GetSortedSetRange(context.Background(), "scores", "-inf", "+inf"); !errors.Is(err, ErrRedisIsDown) {
		t.Fatalf("GetSortedSetRange() error = %v, want ErrRedisIsDown", err)
	}
	if err := cluster.RemoveSortedSetRange(context.Background(), "scores", "-inf", "+inf"); !errors.Is(err, ErrRedisIsDown) {
		t.Fatalf("RemoveSortedSetRange() error = %v, want ErrRedisIsDown", err)
	}
}

func TestRedisClusterConnectReflectsReadiness(t *testing.T) {
	if NewRedisCluster(nil).Connect() {
		t.Fatal("Connect() = true, want false when Redis is disabled")
	}
}

func TestRedisConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{
			name:   "single node",
			config: &Config{Host: "192.168.1.119", Port: 6888},
		},
		{
			name:   "address list",
			config: &Config{Address: []string{"127.0.0.1:6379"}},
		},
		{
			name:    "missing address",
			config:  &Config{},
			wantErr: "host or address",
		},
		{
			name:    "cluster database",
			config:  &Config{Host: "127.0.0.1", Port: 6379, EnableCluster: true, Database: 1},
			wantErr: "database must be 0",
		},
		{
			name:    "invalid address",
			config:  &Config{Address: []string{"not-an-address"}},
			wantErr: "invalid redis address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestClientCloseClearsReadiness(t *testing.T) {
	client, err := NewClient(&Config{Host: "127.0.0.1", Port: 6379})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.pool = NewRedisClusterPool(false, &Config{Host: "127.0.0.1", Port: 6379})
	client.setUp(true)

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	if client.Connected() {
		t.Fatal("Connected() = true after Close(), want false")
	}
	if client.connect() {
		t.Fatal("connect() = true after Close(), want false")
	}
}

func TestClientStartReturnsWhenClosed(t *testing.T) {
	client, err := NewClient(&Config{Host: "127.0.0.1", Port: 6379})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := client.Start(context.Background()); !errors.Is(err, ErrRedisClientClosed) {
		t.Fatalf("Start() error = %v, want ErrRedisClientClosed", err)
	}
}

func TestWaitForRedisProbeReturnsWhenClientCloses(t *testing.T) {
	client, err := NewClient(&Config{Host: "127.0.0.1", Port: 6379})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result := make(chan error, 1)
	go func() {
		result <- waitForRedisProbe(context.Background(), client.done, time.Hour)
	}()
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, ErrRedisClientClosed) {
			t.Fatalf("waitForRedisProbe() error = %v, want ErrRedisClientClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("waitForRedisProbe() did not return after Client.Close()")
	}
}

func TestClientsOwnIndependentPools(t *testing.T) {
	first, err := NewClient(&Config{Host: "127.0.0.1", Port: 6379})
	if err != nil {
		t.Fatalf("NewClient(first) error = %v", err)
	}
	second, err := NewClient(&Config{Host: "127.0.0.2", Port: 6379})
	if err != nil {
		t.Fatalf("NewClient(second) error = %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })
	t.Cleanup(func() { _ = second.Close() })

	if !first.connect() || !second.connect() {
		t.Fatal("connect() = false, want independent pools to be created")
	}
	if first.singleton() == second.singleton() {
		t.Fatal("clients share a pool, want isolated pools")
	}
}

func TestRedisClusterCacheModeSharesPrimaryClient(t *testing.T) {
	client, err := NewClient(&Config{Host: "127.0.0.1", Port: 6379})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	pool := NewRedisClusterPool(false, &Config{Host: "127.0.0.1", Port: 6379})
	client.pool = pool
	client.setUp(true)

	if !NewRedisCluster(client).Connect() {
		t.Fatal("primary RedisCluster is unavailable, want shared client readiness")
	}
	if !NewRedisCluster(client).Connect() {
		t.Fatal("cache RedisCluster is unavailable, want shared client readiness")
	}
}
