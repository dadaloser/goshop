package v1

import (
	"context"
	"sort"
	"strings"

	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/goods/srv/internal/domain/dto"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"

	metav1 "goshop/pkg/common/meta/v1"
)

func (gs *goodsService) ListOutboxEvents(ctx context.Context, topic, status string, page, pageSize int) ([]*do.OutboxEventDO, int64, error) {
	if gs == nil || gs.data == nil || gs.data.Outbox() == nil {
		return nil, 0, errors.NewCode(errcode.ErrDatabase, "goods outbox store is not configured")
	}
	topic = normalizeGoodsOutboxTopic(topic)
	status = strings.ToUpper(strings.TrimSpace(status))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	total, err := gs.countOutboxEvents(ctx, topic, status)
	if err != nil {
		return nil, 0, err
	}
	limit := page * pageSize
	items, err := gs.listOutboxEvents(ctx, topic, status, limit)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if offset >= len(items) {
		return []*do.OutboxEventDO{}, total, nil
	}
	end := offset + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], total, nil
}

func (gs *goodsService) ReplayOutbox(ctx context.Context, ids []int32, status string, limit int) ([]int32, error) {
	if gs == nil || gs.data == nil || gs.data.Outbox() == nil {
		return nil, errors.NewCode(errcode.ErrDatabase, "goods outbox store is not configured")
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		status = do.OutboxStatusDead
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	filter := make(map[int32]struct{}, len(ids))
	for _, id := range ids {
		if id > 0 {
			filter[id] = struct{}{}
		}
	}
	var (
		events []*do.OutboxEventDO
		err    error
	)
	if len(filter) > 0 {
		selected := make([]int32, 0, len(filter))
		for id := range filter {
			selected = append(selected, id)
		}
		events, err = gs.data.Outbox().ListByIDs(ctx, selected)
	} else {
		events, err = gs.listOutboxEvents(ctx, do.OutboxTopicGoodsSync, status, limit)
	}
	if err != nil {
		return nil, err
	}
	replayed := make([]int32, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		if event.Topic != do.OutboxTopicGoodsSync {
			continue
		}
		if status != "" && !strings.EqualFold(event.Status, status) {
			continue
		}
		if err := gs.data.Outbox().MarkRetry(ctx, event.ID, event.RetryCount, 0, ""); err != nil {
			return nil, err
		}
		replayed = append(replayed, event.ID)
	}
	sort.Slice(replayed, func(i, j int) bool { return replayed[i] < replayed[j] })
	return replayed, nil
}

func (gs *goodsService) Reindex(ctx context.Context, ids []uint64, all bool) ([]uint64, error) {
	if gs == nil || gs.data == nil || gs.searchData == nil || gs.searchData.Goods() == nil {
		return nil, errors.NewCode(errcode.ErrDatabase, "goods search store is not configured")
	}
	if all {
		return gs.reindexAll(ctx)
	}
	uniq := make([]uint64, 0, len(ids))
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return nil, errors.NewCode(errcode.ErrValidation, "goods_ids or all=true is required")
	}
	reindexed := make([]uint64, 0, len(uniq))
	for _, id := range uniq {
		current, err := gs.data.Goods().Get(ctx, id)
		if err != nil {
			return nil, err
		}
		value := goodsSearchFromDTO(&dto.GoodsDTO{GoodsDO: *current})
		if err := gs.searchData.Goods().Update(ctx, &value); err != nil {
			return nil, err
		}
		reindexed = append(reindexed, id)
	}
	return reindexed, nil
}

func (gs *goodsService) reindexAll(ctx context.Context) ([]uint64, error) {
	reindexed := make([]uint64, 0)
	page := 1
	pageSize := 100
	for {
		list, err := gs.data.Goods().List(ctx, []string{"id asc"}, metav1.ListMeta{Page: page, PageSize: pageSize})
		if err != nil {
			return nil, err
		}
		if list == nil || len(list.Items) == 0 {
			break
		}
		for _, goods := range list.Items {
			if goods == nil {
				continue
			}
			searchDoc := goodsSearchFromDTO(&dto.GoodsDTO{GoodsDO: *goods})
			if err := gs.searchData.Goods().Update(ctx, &searchDoc); err != nil {
				return nil, err
			}
			reindexed = append(reindexed, uint64(goods.ID))
		}
		page++
	}
	return reindexed, nil
}

func (gs *goodsService) listOutboxEvents(ctx context.Context, topic, status string, limit int) ([]*do.OutboxEventDO, error) {
	return gs.data.Outbox().ListByStatus(ctx, topic, status, limit)
}

func (gs *goodsService) countOutboxEvents(ctx context.Context, topic, status string) (int64, error) {
	if status != "" {
		return gs.data.Outbox().CountByStatus(ctx, topic, status)
	}
	statuses := []string{do.OutboxStatusPending, do.OutboxStatusProcessing, do.OutboxStatusDone, do.OutboxStatusDead}
	var total int64
	for _, item := range statuses {
		count, err := gs.data.Outbox().CountByStatus(ctx, topic, item)
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func normalizeGoodsOutboxTopic(topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return do.OutboxTopicGoodsSync
	}
	return topic
}
