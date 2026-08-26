package smscode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"goshop/pkg/storage"

	"github.com/redis/go-redis/v9"
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
	Delete(ctx context.Context, key string) bool
	DeleteIfValue(ctx context.Context, key, value string) (bool, error)
	Consume(ctx context.Context, key, value string) error
}

type RedisStore struct {
	client *storage.RedisCluster
}

func NewRedisStore(client ...*storage.Client) *RedisStore {
	return &RedisStore{client: storage.NewRedisCluster(client...)}
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

func (s *RedisStore) Delete(ctx context.Context, key string) bool {
	return s.client.DeleteKey(ctx, key)
}

var deleteIfValueScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if current ~= ARGV[1] then return 0 end
redis.call('DEL', KEYS[1])
return 1
`)

func (s *RedisStore) DeleteIfValue(ctx context.Context, key, value string) (bool, error) {
	client := s.client.GetClient()
	if client == nil {
		return false, storage.ErrRedisIsDown
	}
	deleted, err := deleteIfValueScript.Run(ctx, client, []string{key}, value).Int()
	if err != nil {
		return false, err
	}
	return deleted == 1, nil
}

var consumeScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if not current then return 0 end
if current ~= ARGV[1] then return -1 end
redis.call('DEL', KEYS[1])
return 1
`)

func (s *RedisStore) Consume(ctx context.Context, key, value string) error {
	client := s.client.GetClient()
	if client == nil {
		return storage.ErrRedisIsDown
	}
	result, err := consumeScript.Run(ctx, client, []string{key}, value).Int()
	if err != nil {
		return err
	}
	switch result {
	case 1:
		return nil
	case -1:
		return ErrCodeMismatch
	default:
		return storage.ErrKeyNotFound
	}
}
