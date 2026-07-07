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

func NewRedisStore() *RedisStore {
	return &RedisStore{
		client:   &storage.RedisCluster{},
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

	if !storage.Connected() {
		return false, storage.ErrRedisIsDown
	}
	client := s.client.GetClient()
	if client == nil {
		return false, storage.ErrRedisIsDown
	}

	exists, err := client.Exists(ctx, cooldownKey).Result()
	if err != nil {
		return false, err
	}
	if exists > 0 {
		return false, nil
	}

	count, err := client.Incr(ctx, windowKey).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err = client.Expire(ctx, windowKey, s.window).Err(); err != nil {
			return false, err
		}
	}
	if count > int64(s.maxSends) {
		return false, nil
	}

	if err = client.Set(ctx, cooldownKey, "1", s.cooldown).Err(); err != nil {
		return false, err
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
