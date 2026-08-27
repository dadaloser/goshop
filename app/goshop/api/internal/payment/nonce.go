package payment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goshop/pkg/storage"
)

type NonceStore interface {
	Reserve(context.Context, string, time.Duration) (bool, error)
}

type RedisNonceStore struct{ client *storage.RedisCluster }

func NewRedisNonceStore(client *storage.Client) *RedisNonceStore {
	return &RedisNonceStore{client: storage.NewRedisCluster(client)}
}

func (s *RedisNonceStore) Reserve(ctx context.Context, nonce string, ttl time.Duration) (bool, error) {
	if s == nil || s.client == nil || strings.TrimSpace(nonce) == "" || ttl <= 0 {
		return false, fmt.Errorf("invalid payment callback nonce")
	}
	reserved, err := s.client.SetIfAbsent(ctx, "payment:callback:nonce:"+nonce, "1", ttl)
	if err != nil {
		return false, fmt.Errorf("reserve payment callback nonce: %w", err)
	}
	return reserved, nil
}

var _ NonceStore = (*RedisNonceStore)(nil)
