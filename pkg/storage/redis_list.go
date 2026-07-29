package storage

import (
	"context"

	"goshop/pkg/log"
)

// RemoveFromList removes all matching values from the list identified by keyName.
func (r *RedisCluster) RemoveFromList(ctx context.Context, keyName, value string) error {
	if err := r.up(); err != nil {
		return err
	}
	fixedKey := r.fixKey(keyName)
	if err := r.singleton().LRem(ctx, fixedKey, 0, value).Err(); err != nil {
		log.Error("LREM command failed", log.String("keyHash", redactedRedisKey(keyName)), log.String("fixedKeyHash", redactedRedisKey(fixedKey)), log.String("value", redactedRedisValue(value)), log.String("error", err.Error()))
		return err
	}
	return nil
}

// GetListRange gets the requested element range from the list identified by keyName.
func (r *RedisCluster) GetListRange(ctx context.Context, keyName string, from, to int64) ([]string, error) {
	if err := r.up(); err != nil {
		return nil, err
	}
	fixedKey := r.fixKey(keyName)
	elements, err := r.singleton().LRange(ctx, fixedKey, from, to).Result()
	if err != nil {
		log.Error("LRANGE command failed", log.String("keyHash", redactedRedisKey(keyName)), log.String("fixedKeyHash", redactedRedisKey(fixedKey)), log.Int64("from", from), log.Int64("to", to), log.String("error", err.Error()))
		return nil, err
	}
	return elements, nil
}
