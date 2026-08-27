package smslimit

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	stderrors "errors"

	"goshop/pkg/storage"
)

const (
	defaultCooldown = time.Minute
	defaultWindow   = 24 * time.Hour
	defaultMaxSends = 10

	cooldownKeyPrefix = "sms:cooldown:"
	windowKeyPrefix   = "sms:window:"
)

var ErrEmptyMobile = stderrors.New("mobile is empty")

type Store interface {
	Take(ctx context.Context, mobile string, codeType uint) (bool, error)
}

type RedisStore struct {
	client   *storage.RedisCluster
	cooldown time.Duration
	window   time.Duration
	maxSends int
}

func NewRedisStore(client *storage.Client) *RedisStore {
	return &RedisStore{
		client:   storage.NewRedisCluster(client),
		cooldown: defaultCooldown,
		window:   defaultWindow,
		maxSends: defaultMaxSends,
	}
}

func (s *RedisStore) Take(ctx context.Context, mobile string, codeType uint) (bool, error) {
	cooldownKey, err := cooldownKey(mobile, codeType)
	if err != nil {
		return false, err
	}
	windowKey, err := windowKey(mobile, codeType)
	if err != nil {
		return false, err
	}

	reserved, err := s.client.SetIfAbsent(ctx, cooldownKey, "1", s.cooldown)
	if err != nil {
		return false, err
	}
	if !reserved {
		return false, nil
	}

	count, err := s.client.IncrementWithExpiry(ctx, windowKey, s.window)
	if err != nil {
		if _, cleanupErr := s.client.DeleteIfValue(ctx, cooldownKey, "1"); cleanupErr != nil {
			return false, stderrors.Join(err, fmt.Errorf("clear sms cooldown after increment failure: %w", cleanupErr))
		}
		return false, err
	}
	if count > int64(s.maxSends) {
		if _, err := s.client.DeleteIfValue(ctx, cooldownKey, "1"); err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func cooldownKey(mobile string, codeType uint) (string, error) {
	return keyedMobile(cooldownKeyPrefix, mobile, codeType)
}

func windowKey(mobile string, codeType uint) (string, error) {
	return keyedMobile(windowKeyPrefix, mobile, codeType)
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
