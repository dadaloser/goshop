package storage

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"goshop/pkg/log"
)

/**
有序集合操作
*/

// AddToSortedSet adds value with score to the sorted set identified by keyName.
func (r *RedisCluster) AddToSortedSet(ctx context.Context, keyName, value string, score float64) {
	fixedKey := r.fixKey(keyName)
	log.Debug("Pushing key to sorted set", log.String("keyHash", redactedRedisKey(keyName)), log.String("fixedKeyHash", redactedRedisKey(fixedKey)), log.String("value", redactedRedisValue(value)))
	if err := r.up(); err != nil {
		log.Debug(err.Error())
		return
	}
	if err := r.singleton().ZAdd(ctx, fixedKey, redis.Z{Score: score, Member: value}).Err(); err != nil {
		log.Error("ZADD command failed", log.String("keyHash", redactedRedisKey(keyName)), log.String("fixedKeyHash", redactedRedisKey(fixedKey)), log.String("error", err.Error()))
	}
}

// GetSortedSetRange gets sorted-set members and scores within the requested range.
func (r *RedisCluster) GetSortedSetRange(ctx context.Context, keyName, scoreFrom, scoreTo string) ([]string, []float64, error) {
	fixedKey := r.fixKey(keyName)
	args := redis.ZRangeBy{Min: scoreFrom, Max: scoreTo}
	values, err := r.singleton().ZRangeByScoreWithScores(ctx, fixedKey, &args).Result()
	if err != nil {
		log.Error("ZRANGEBYSCORE command failed", log.String("keyHash", redactedRedisKey(keyName)), log.String("fixedKeyHash", redactedRedisKey(fixedKey)), log.String("error", err.Error()))
		return nil, nil, err
	}
	if len(values) == 0 {
		return nil, nil, nil
	}
	elements := make([]string, len(values))
	scores := make([]float64, len(values))
	for i, value := range values {
		elements[i] = fmt.Sprint(value.Member)
		scores[i] = value.Score
	}
	return elements, scores, nil
}

// RemoveSortedSetRange removes sorted-set members within the requested range.
func (r *RedisCluster) RemoveSortedSetRange(ctx context.Context, keyName, scoreFrom, scoreTo string) error {
	fixedKey := r.fixKey(keyName)
	if err := r.singleton().ZRemRangeByScore(ctx, fixedKey, scoreFrom, scoreTo).Err(); err != nil {
		log.Debug("ZREMRANGEBYSCORE command failed", log.String("keyHash", redactedRedisKey(keyName)), log.String("fixedKeyHash", redactedRedisKey(fixedKey)), log.String("error", err.Error()))
		return err
	}
	return nil
}
