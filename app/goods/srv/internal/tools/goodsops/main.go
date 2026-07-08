package main

import (
	"context"
	"fmt"
	"os"

	"goshop/app/goods/srv/config"
	dbv1 "goshop/app/goods/srv/internal/data/v1/db"
	searches "goshop/app/goods/srv/internal/data_search/v1/es"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/pkg/app"
	"goshop/pkg/log"

	"github.com/spf13/viper"
)

func main() {
	cfg := config.New()
	appl := app.NewApp(
		"goodsops",
		"goshop",
		app.WithOptions(cfg),
		app.WithNoVersion(),
		app.WithRunFunc(func(ctx context.Context, basename string) error {
			log.Init(cfg.Log)
			defer log.Flush()

			if len(os.Args) < 2 {
				return fmt.Errorf("missing action, want reindex-goods or replay-dead-outbox")
			}

			dataFactory, err := dbv1.GetDBFactoryOr(cfg.MySQLOptions)
			if err != nil {
				return err
			}
			searchFactory, err := searches.GetSearchFactoryOr(cfg.EsOptions)
			if err != nil {
				return err
			}

			switch os.Args[1] {
			case "reindex-goods":
				return reindexGoods(ctx, dataFactory, searchFactory)
			case "replay-dead-outbox":
				return replayDeadOutbox(ctx, dataFactory.Outbox(), viper.GetInt("limit"))
			default:
				return fmt.Errorf("unsupported action %q", os.Args[1])
			}
		}),
	)

	appl.Run()
}

func replayDeadOutbox(ctx context.Context, outboxStore interface {
	ListByStatus(context.Context, string, string, int) ([]*do.OutboxEventDO, error)
	MarkRetry(context.Context, int32, int32, int64, string) error
}, limit int) error {
	if limit <= 0 {
		limit = 100
	}
	events, err := outboxStore.ListByStatus(ctx, do.OutboxTopicGoodsSync, do.OutboxStatusDead, limit)
	if err != nil {
		return err
	}
	for _, event := range events {
		if event == nil {
			continue
		}
		if err := outboxStore.MarkRetry(ctx, event.ID, event.RetryCount, 0, ""); err != nil {
			return err
		}
	}
	log.Infof("replayed %d dead outbox events", len(events))
	return nil
}
