package smscode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"goshop/pkg/storage"
)

/**
验证码存储
*/

var ErrCodeMismatch = errors.New("sms verification code mismatch")

const (
	TypeRegister uint = 1
	TypeLogin    uint = 2

	DefaultTTL = 5 * time.Minute
)

type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) (bool, error)
	DeleteIfValue(ctx context.Context, key, value string) (bool, error)
	Consume(ctx context.Context, key, value string) error
}

type RedisStore struct {
	client *storage.RedisCluster
}

func NewRedisStore(client *storage.Client) *RedisStore {
	return &RedisStore{client: storage.NewRedisCluster(client)}
}

func RegisterKey(mobile string) string {
	return Key(mobile, TypeRegister)
}

func LoginKey(mobile string) string {
	return Key(mobile, TypeLogin)
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

func (s *RedisStore) Delete(ctx context.Context, key string) (bool, error) {
	return s.client.Delete(ctx, key)
}

func (s *RedisStore) DeleteIfValue(ctx context.Context, key, value string) (bool, error) {
	return s.client.DeleteIfValue(ctx, key, value)
}

func (s *RedisStore) Consume(ctx context.Context, key, value string) error {
	err := s.client.ConsumeIfValue(ctx, key, value)
	if errors.Is(err, storage.ErrKeyValueMismatch) {
		return ErrCodeMismatch
	}
	return err
}
