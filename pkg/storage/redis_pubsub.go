package storage

import (
	"context"

	"goshop/pkg/errors"
	"goshop/pkg/log"

	"github.com/redis/go-redis/v9"
)

/**
发布与订阅
*/

// StartPubSubHandler listens for subscription messages until ctx is cancelled.
func (r *RedisCluster) StartPubSubHandler(ctx context.Context, channel string, callback func(interface{})) error {
	if err := r.up(); err != nil {
		return err
	}
	client := r.singleton()
	if client == nil {
		return errors.New("redis connection failed")
	}

	pubsub := client.Subscribe(ctx, channel)
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Errorf("Error trying to close pubsub: %s", err.Error())
		}
	}()

	if _, err := pubsub.Receive(ctx); err != nil {
		log.Errorf("Error while receiving pubsub message: %s", err.Error())
		return err
	}

	return handlePubSubMessages(ctx, pubsub.Channel(), callback)
}

func handlePubSubMessages(ctx context.Context, messages <-chan *redis.Message, callback func(interface{})) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-messages:
			if !ok {
				return nil
			}
			callback(msg)
		}
	}
}

// Publish publishes message to channel.
func (r *RedisCluster) Publish(ctx context.Context, channel, message string) error {
	if err := r.up(); err != nil {
		return err
	}
	if err := r.singleton().Publish(ctx, channel, message).Err(); err != nil {
		log.Errorf("Error trying to publish message: %s", err.Error())
		return err
	}
	return nil
}
