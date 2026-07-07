package loginattempt

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	stderrors "errors"

	"goshop/pkg/storage"
)

const (
	defaultMaxFailures = 5
	defaultWindow      = 15 * time.Minute
	defaultLockTTL     = 15 * time.Minute

	failKeyPrefix = "login:fail:"
	lockKeyPrefix = "login:lock:"
)

var ErrEmptyIdentifier = stderrors.New("identifier is empty")

type Store interface {
	IsLocked(ctx context.Context, identifier string) (bool, error)
	RecordFailure(ctx context.Context, identifier string) (bool, error)
	Reset(ctx context.Context, identifier string) error
}

type RedisStore struct {
	client      *storage.RedisCluster
	maxFailures int
	window      time.Duration
	lockTTL     time.Duration
}

func NewRedisStore() *RedisStore {
	return &RedisStore{
		client:      &storage.RedisCluster{},
		maxFailures: defaultMaxFailures,
		window:      defaultWindow,
		lockTTL:     defaultLockTTL,
	}
}

func (s *RedisStore) IsLocked(ctx context.Context, identifier string) (bool, error) {
	key, err := lockKey(identifier)
	if err != nil {
		return false, err
	}

	if _, err = s.client.GetKey(ctx, key); err != nil {
		if stderrors.Is(err, storage.ErrKeyNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *RedisStore) RecordFailure(ctx context.Context, identifier string) (bool, error) {
	if locked, err := s.IsLocked(ctx, identifier); err != nil || locked {
		return locked, err
	}

	key, err := failKey(identifier)
	if err != nil {
		return false, err
	}

	if !storage.Connected() {
		return false, storage.ErrRedisIsDown
	}
	client := s.client.GetClient()
	if client == nil {
		return false, storage.ErrRedisIsDown
	}

	failures, err := client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if failures == 1 {
		if err = client.Expire(ctx, key, s.window).Err(); err != nil {
			return false, err
		}
	}

	if failures >= int64(s.maxFailures) {
		lock, err := lockKey(identifier)
		if err != nil {
			return false, err
		}
		if err = s.client.SetKey(ctx, lock, "1", s.lockTTL); err != nil {
			return false, err
		}
		s.client.DeleteKey(ctx, key)
		return true, nil
	}
	return false, nil
}

func (s *RedisStore) Reset(ctx context.Context, identifier string) error {
	key, err := failKey(identifier)
	if err != nil {
		return err
	}
	s.client.DeleteKey(ctx, key)

	key, err = lockKey(identifier)
	if err != nil {
		return err
	}
	s.client.DeleteKey(ctx, key)
	return nil
}

func failKey(identifier string) (string, error) {
	return keyedIdentifier(failKeyPrefix, identifier)
}

func lockKey(identifier string) (string, error) {
	return keyedIdentifier(lockKeyPrefix, identifier)
}

func keyedIdentifier(prefix, identifier string) (string, error) {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	if identifier == "" {
		return "", ErrEmptyIdentifier
	}

	sum := sha256.Sum256([]byte(identifier))
	return prefix + base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

var _ Store = &RedisStore{}
