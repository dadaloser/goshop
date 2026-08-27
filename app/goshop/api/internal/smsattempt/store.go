package smsattempt

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"goshop/pkg/storage"
)

const (
	defaultMaxFailures = 5
	defaultWindow      = 5 * time.Minute
	defaultLockTTL     = 15 * time.Minute

	failKeyPrefix = "sms:verify:fail:"
	lockKeyPrefix = "sms:verify:lock:"
)

var ErrEmptyMobile = stderrors.New("mobile is empty")

type Store interface {
	IsLocked(ctx context.Context, mobile string, codeType uint) (bool, error)
	RecordFailure(ctx context.Context, mobile string, codeType uint) (bool, error)
	Reset(ctx context.Context, mobile string, codeType uint) error
}

type RedisStore struct {
	client      *storage.RedisCluster
	maxFailures int
	window      time.Duration
	lockTTL     time.Duration
}

func NewRedisStore(client *storage.Client) *RedisStore {
	return &RedisStore{
		client:      storage.NewRedisCluster(client),
		maxFailures: defaultMaxFailures,
		window:      defaultWindow,
		lockTTL:     defaultLockTTL,
	}
}

func (s *RedisStore) IsLocked(ctx context.Context, mobile string, codeType uint) (bool, error) {
	key, err := lockKey(mobile, codeType)
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

func (s *RedisStore) RecordFailure(ctx context.Context, mobile string, codeType uint) (bool, error) {
	if locked, err := s.IsLocked(ctx, mobile, codeType); err != nil || locked {
		return locked, err
	}

	key, err := failKey(mobile, codeType)
	if err != nil {
		return false, err
	}

	failures, err := s.client.IncrementWithExpiry(ctx, key, s.window)
	if err != nil {
		return false, err
	}

	if failures >= int64(s.maxFailures) {
		lock, err := lockKey(mobile, codeType)
		if err != nil {
			return false, err
		}
		if err = s.client.SetKey(ctx, lock, "1", s.lockTTL); err != nil {
			return false, err
		}
		if _, err := s.client.Delete(ctx, key); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *RedisStore) Reset(ctx context.Context, mobile string, codeType uint) error {
	key, err := failKey(mobile, codeType)
	if err != nil {
		return err
	}
	if _, err := s.client.Delete(ctx, key); err != nil {
		return err
	}

	key, err = lockKey(mobile, codeType)
	if err != nil {
		return err
	}
	_, err = s.client.Delete(ctx, key)
	return err
}

func failKey(mobile string, codeType uint) (string, error) {
	return keyedMobile(failKeyPrefix, mobile, codeType)
}

func lockKey(mobile string, codeType uint) (string, error) {
	return keyedMobile(lockKeyPrefix, mobile, codeType)
}

func keyedMobile(prefix, mobile string, codeType uint) (string, error) {
	mobile = strings.TrimSpace(mobile)
	if mobile == "" {
		return "", ErrEmptyMobile
	}

	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", codeType, mobile)))
	return prefix + base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

var _ Store = &RedisStore{}
