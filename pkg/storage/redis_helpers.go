package storage

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

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

// SetIfAbsent atomically sets keyName only when it does not already exist.
func (r *RedisCluster) SetIfAbsent(ctx context.Context, keyName, value string, ttl time.Duration) (bool, error) {
	if err := r.up(); err != nil {
		return false, err
	}
	set, err := r.singleton().SetNX(ctx, r.fixKey(keyName), value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("set redis key if absent: %w", err)
	}
	return set, nil
}

// Increment increments keyName and returns the resulting value.
func (r *RedisCluster) Increment(ctx context.Context, keyName string) (int64, error) {
	if err := r.up(); err != nil {
		return 0, err
	}
	value, err := r.singleton().Incr(ctx, r.fixKey(keyName)).Result()
	if err != nil {
		return 0, fmt.Errorf("increment redis key: %w", err)
	}
	return value, nil
}

// IncrementWithExpiry increments keyName and sets ttl only for its first value.
func (r *RedisCluster) IncrementWithExpiry(ctx context.Context, keyName string, ttl time.Duration) (int64, error) {
	if err := r.up(); err != nil {
		return 0, err
	}
	value, err := incrementWithExpireScript.Run(ctx, r.singleton(), []string{r.fixKey(keyName)}, int64(ttl.Seconds())).Int64()
	if err != nil {
		return 0, fmt.Errorf("increment redis key with expiry: %w", err)
	}
	return value, nil
}

// ConsumeIfValue atomically removes keyName only if its value matches. A
// missing key returns ErrKeyNotFound and a non-matching value returns
// ErrKeyValueMismatch.
func (r *RedisCluster) ConsumeIfValue(ctx context.Context, keyName, value string) error {
	if err := r.up(); err != nil {
		return err
	}
	result, err := compareAndDeleteScript.Run(ctx, r.singleton(), []string{r.fixKey(keyName)}, value).Int()
	if err != nil {
		return fmt.Errorf("consume redis key: %w", err)
	}
	switch result {
	case 1:
		return nil
	case -1:
		return ErrKeyValueMismatch
	default:
		return ErrKeyNotFound
	}
}

// DeleteIfValue atomically removes keyName if its value matches. It returns
// false when the key is missing or holds a different value.
func (r *RedisCluster) DeleteIfValue(ctx context.Context, keyName, value string) (bool, error) {
	err := r.ConsumeIfValue(ctx, keyName, value)
	if err == nil {
		return true, nil
	}
	if stderrors.Is(err, ErrKeyNotFound) || stderrors.Is(err, ErrKeyValueMismatch) {
		return false, nil
	}
	return false, err
}

// Delete removes keyName and reports whether Redis removed an existing key.
func (r *RedisCluster) Delete(ctx context.Context, keyName string) (bool, error) {
	if err := r.up(); err != nil {
		return false, err
	}
	deleted, err := r.singleton().Del(ctx, r.fixKey(keyName)).Result()
	if err != nil {
		return false, fmt.Errorf("delete redis key: %w", err)
	}
	return deleted > 0, nil
}
