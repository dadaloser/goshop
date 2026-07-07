package tokenrevocation

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	stderrors "errors"

	"goshop/pkg/storage"
)

const keyPrefix = "jwt:revoked:"

var ErrEmptyToken = stderrors.New("token is empty")

type Store interface {
	Revoke(ctx context.Context, token string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, token string) (bool, error)
}

type RedisStore struct {
	client *storage.RedisCluster
}

func NewRedisStore() *RedisStore {
	return &RedisStore{client: &storage.RedisCluster{}}
}

func (s *RedisStore) Revoke(ctx context.Context, token string, expiresAt time.Time) error {
	key, err := key(token)
	if err != nil {
		return err
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Second
	}
	return s.client.SetKey(ctx, key, "1", ttl)
}

func (s *RedisStore) IsRevoked(ctx context.Context, token string) (bool, error) {
	key, err := key(token)
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

func key(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrEmptyToken
	}

	sum := sha256.Sum256([]byte(token))
	return keyPrefix + base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

var _ Store = &RedisStore{}
