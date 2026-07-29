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
	return true
}

func (r *RedisCluster) singleton() redis.UniversalClient {
	return singleton(r.IsCache)
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
	if !Connected() || r.singleton() == nil {
		return ErrRedisIsDown
	}

	return nil
}
