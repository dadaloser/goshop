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

func NewRedisStore(client *storage.Client) *RedisStore {
	return &RedisStore{client: storage.NewRedisCluster(client)}
}

func (s *RedisStore) Issue(ctx context.Context, email, purpose, code string, ttl, interval time.Duration) error {
	key := key(email, purpose)
	set, err := s.client.SetIfAbsent(ctx, key+":rate", "1", interval)
	if err != nil {
		return fmt.Errorf("rate limit email code: %w", err)
	}
	if !set {
		return ErrRateLimited
	}
	if err := s.client.SetKey(ctx, key, codeHash(email, purpose, code), ttl); err != nil {
		if _, cleanupErr := s.client.Delete(ctx, key+":rate"); cleanupErr != nil {
			return fmt.Errorf("store email code: %w", errors.Join(err, fmt.Errorf("clear email code rate limit: %w", cleanupErr)))
		}
		return fmt.Errorf("store email code: %w", err)
	}
	return nil
}

func (s *RedisStore) Consume(ctx context.Context, email, purpose, code string) error {
	err := s.client.ConsumeIfValue(ctx, key(email, purpose), codeHash(email, purpose, code))
	switch {
	case err == nil:
		return nil
	case errors.Is(err, storage.ErrKeyValueMismatch):
		return ErrInvalid
	case errors.Is(err, storage.ErrKeyNotFound):
		return ErrNotFound
	default:
		return fmt.Errorf("consume email code: %w", err)
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
