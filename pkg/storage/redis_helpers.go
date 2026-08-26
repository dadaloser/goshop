package storage

import (
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

/**
客户端获取、连接状态、键前缀、哈希与脱敏
*/

// Connect reports whether the storage backend can be used. Connections are
// established and checked by ConnectToRedis.
func (r *RedisCluster) Connect() bool {
	return r.up() == nil
}

// UnavailableReadinessCheck reports an explicit missing Redis dependency. It
// is used only when a caller has not been supplied an application Client.
func UnavailableReadinessCheck() func() error {
	return func() error { return ErrRedisIsDown }
}

func (r *RedisCluster) singleton() redis.UniversalClient {
	if r != nil && r.client != nil {
		return r.client.singleton()
	}
	return nil
}

// GetClient returns the Redis client associated with this storage instance.
func (r *RedisCluster) GetClient() redis.UniversalClient {
	return r.singleton()
}

func (r *RedisCluster) hashKey(in string) string {
	if !r.HashKeys {
		return in
	}

	return HashStr(in)
}

func (r *RedisCluster) fixKey(keyName string) string {
	return r.KeyPrefix + r.hashKey(keyName)
}

func redactedRedisValue(value string) string {
	if value == "" {
		return ""
	}

	return fmt.Sprintf("<redacted len=%d>", len(value))
}

func redactedRedisKey(key string) string {
	if key == "" {
		return ""
	}

	return HashStr(key)
}

func (r *RedisCluster) cleanKey(keyName string) string {
	return strings.TrimPrefix(keyName, r.KeyPrefix)
}

func (r *RedisCluster) up() error {
	if r == nil {
		return ErrRedisIsDown
	}
	if r.client == nil || !r.client.Connected() || r.singleton() == nil {
		return ErrRedisIsDown
	}

	return nil
}
