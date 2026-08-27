package tokenversion

import (
	"context"
	stderrors "errors"
	"fmt"
	"strconv"
	"strings"

	"goshop/pkg/storage"
)

const keyPrefix = "jwt:version:"

var ErrInvalidUserID = stderrors.New("user id is invalid")

type Store interface {
	CurrentVersion(ctx context.Context, userID uint64) (uint64, error)
	Bump(ctx context.Context, userID uint64) (uint64, error)
}

type RedisStore struct {
	client *storage.RedisCluster
}

func NewRedisStore(client *storage.Client) *RedisStore {
	return &RedisStore{client: storage.NewRedisCluster(client)}
}

func (s *RedisStore) CurrentVersion(ctx context.Context, userID uint64) (uint64, error) {
	key, err := versionKey(userID)
	if err != nil {
		return 0, err
	}

	value, err := s.client.GetKey(ctx, key)
	if err != nil {
		if stderrors.Is(err, storage.ErrKeyNotFound) {
			return 0, nil
		}
		return 0, err
	}

	version, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse token version: %w", err)
	}
	return version, nil
}

func (s *RedisStore) Bump(ctx context.Context, userID uint64) (uint64, error) {
	key, err := versionKey(userID)
	if err != nil {
		return 0, err
	}

	version, err := s.client.Increment(ctx, key)
	if err != nil {
		return 0, err
	}
	return uint64(version), nil
}

func versionKey(userID uint64) (string, error) {
	if userID == 0 {
		return "", ErrInvalidUserID
	}
	return fmt.Sprintf("%s%d", keyPrefix, userID), nil
}

var _ Store = &RedisStore{}
