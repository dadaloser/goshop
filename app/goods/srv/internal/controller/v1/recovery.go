package v1

import (
	"context"
	"strings"

	proto "goshop/api/goods/v1"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"
)

func (gs *goodsServer) ListGoodsOutboxEvents(ctx context.Context, req *proto.ListGoodsOutboxEventsRequest) (*proto.ListGoodsOutboxEventsResponse, error) {
	if req == nil {
		return nil, errors.NewCode(errcode.ErrValidation, "goods outbox request is required")
	}
	items, total, err := gs.srv.Goods().ListOutboxEvents(ctx, req.GetTopic(), req.GetStatus(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	resp := &proto.ListGoodsOutboxEventsResponse{Total: int32(total), Data: make([]*proto.GoodsOutboxEventRecord, 0, len(items))}
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Data = append(resp.Data, &proto.GoodsOutboxEventRecord{
			Id:            item.ID,
			Topic:         item.Topic,
			AggregateType: item.AggregateType,
			AggregateId:   item.AggregateID,
			Action:        item.Action,
			Status:        item.Status,
			RetryCount:    item.RetryCount,
			MaxRetryCount: item.MaxRetryCount,
			LastError:     item.LastError,
			NextAttemptAt: item.NextAttemptAt,
			ClaimedAt:     item.ClaimedAt,
			AddTime:       item.CreatedAt.Unix(),
		})
	}
	return resp, nil
}

func (gs *goodsServer) ReplayGoodsOutbox(ctx context.Context, req *proto.ListGoodsOutboxReplayRequest) (*proto.ListGoodsOutboxReplayResponse, error) {
	if req == nil {
		return nil, errors.NewCode(errcode.ErrValidation, "goods outbox replay request is required")
	}
	replayed, err := gs.srv.Goods().ReplayOutbox(ctx, req.GetIds(), strings.TrimSpace(req.GetStatus()), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	return &proto.ListGoodsOutboxReplayResponse{Replayed: int32(len(replayed)), Ids: replayed}, nil
}

func (gs *goodsServer) ReindexGoods(ctx context.Context, req *proto.GoodsReindexRequest) (*proto.GoodsReindexResponse, error) {
	if req == nil {
		return nil, errors.NewCode(errcode.ErrValidation, "goods reindex request is required")
	}
	ids := make([]uint64, 0, len(req.GetGoodsIds()))
	for _, id := range req.GetGoodsIds() {
		if id > 0 {
			ids = append(ids, uint64(id))
		}
	}
	reindexed, err := gs.srv.Goods().Reindex(ctx, ids, req.GetAll())
	if err != nil {
		return nil, err
	}
	resp := &proto.GoodsReindexResponse{Reindexed: int32(len(reindexed)), GoodsIds: make([]int32, 0, len(reindexed))}
	for _, id := range reindexed {
		resp.GoodsIds = append(resp.GoodsIds, int32(id))
	}
	return resp, nil
}
