package storage

import (
	"context"
	"errors"
	"net"
	"strings"

	"goshop/gmicro/resilience"

	"github.com/redis/go-redis/v9"
)

type redisResilienceHook struct {
	guard *resilience.Guard
}

func newRedisResilienceHook(options *resilience.Options) (*redisResilienceHook, error) {
	guard, err := resilience.NewGuard("redis", options, isRedisDependencyError)
	if err != nil {
		return nil, err
	}
	return &redisResilienceHook{guard: guard}, nil
}

func (h *redisResilienceHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *redisResilienceHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		resource := strings.ToLower(cmd.Name())
		return h.guard.Do(ctx, resource, func(callCtx context.Context) error {
			return next(callCtx, cmd)
		})
	}
}

func (h *redisResilienceHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, commands []redis.Cmder) error {
		return h.guard.Do(ctx, "pipeline", func(callCtx context.Context) error {
			return next(callCtx, commands)
		})
	}
}

func isRedisDependencyError(err error) bool {
	return err != nil && !errors.Is(err, redis.Nil) && !errors.Is(err, context.Canceled)
}
