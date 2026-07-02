package smscode

import (
	"context"
	"fmt"
	"time"

	"goshop/pkg/storage"
)

const (
	TypeRegister uint = 1

	DefaultTTL = 5 * time.Minute
)

type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) bool
}

type RedisStore struct {
	client *storage.RedisCluster
}

func NewRedisStore() *RedisStore {
	return &RedisStore{client: &storage.RedisCluster{}}
}

func RegisterKey(mobile string) string {
	return Key(mobile, TypeRegister)
}

func Key(mobile string, codeType uint) string {
	return fmt.Sprintf("sms:%d:%s", codeType, mobile)
}

func (s *RedisStore) Get(ctx context.Context, key string) (string, error) {
	return s.client.GetKey(ctx, key)
}

func (s *RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.client.SetKey(ctx, key, value, ttl)
}

func (s *RedisStore) Delete(ctx context.Context, key string) bool {
	return s.client.DeleteKey(ctx, key)
}
