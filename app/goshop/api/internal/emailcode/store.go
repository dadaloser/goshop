package emailcode

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"goshop/pkg/storage"

	"github.com/redis/go-redis/v9"
)

var (
	ErrNotFound    = errors.New("email verification code not found")
	ErrInvalid     = errors.New("email verification code invalid")
	ErrRateLimited = errors.New("email verification code rate limited")
)

type Store interface {
	Issue(ctx context.Context, email, purpose, code string, ttl, interval time.Duration) error
	Consume(ctx context.Context, email, purpose, code string) error
}

type RedisStore struct{ client *storage.RedisCluster }

func NewRedisStore() *RedisStore { return &RedisStore{client: &storage.RedisCluster{}} }

func (s *RedisStore) Issue(ctx context.Context, email, purpose, code string, ttl, interval time.Duration) error {
	client := s.client.GetClient()
	if client == nil {
		return storage.ErrRedisIsDown
	}
	key := key(email, purpose)
	set, err := client.SetNX(ctx, key+":rate", "1", interval).Result()
	if err != nil {
		return fmt.Errorf("rate limit email code: %w", err)
	}
	if !set {
		return ErrRateLimited
	}
	if err := client.Set(ctx, key, codeHash(email, purpose, code), ttl).Err(); err != nil {
		client.Del(ctx, key+":rate")
		return fmt.Errorf("store email code: %w", err)
	}
	return nil
}

var consumeScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if not current then return 0 end
if current ~= ARGV[1] then return -1 end
redis.call('DEL', KEYS[1])
return 1
`)

func (s *RedisStore) Consume(ctx context.Context, email, purpose, code string) error {
	client := s.client.GetClient()
	if client == nil {
		return storage.ErrRedisIsDown
	}
	result, err := consumeScript.Run(ctx, client, []string{key(email, purpose)}, codeHash(email, purpose, code)).Int()
	if err != nil {
		return fmt.Errorf("consume email code: %w", err)
	}
	switch result {
	case 1:
		return nil
	case -1:
		return ErrInvalid
	default:
		return ErrNotFound
	}
}

func key(email, purpose string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "verify:email:" + purpose + ":" + base64.RawURLEncoding.EncodeToString(sum[:])
}

func codeHash(email, purpose, code string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email)) + "\x00" + purpose + "\x00" + code))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
