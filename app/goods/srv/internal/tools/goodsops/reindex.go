package main

import (
	"context"

	datav1 "goshop/app/goods/srv/internal/data/v1"
	searchv1 "goshop/app/goods/srv/internal/data_search/v1"
	"goshop/app/goods/srv/internal/domain/do"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/log"
)

func reindexGoods(ctx context.Context, dataFactory datav1.DataFactory, searchFactory searchv1.SearchFactory) error {
	if dataFactory == nil || searchFactory == nil {
		return nil
	}

	page := 1
	pageSize := 100
	total := 0

	for {
		list, err := dataFactory.Goods().List(ctx, []string{"id asc"}, metav1.ListMeta{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return err
		}
		if list == nil || len(list.Items) == 0 {
			break
		}

		for _, goods := range list.Items {
			if goods == nil {
				continue
			}
			searchDoc := do.GoodsSearchDO{
				ID:             goods.ID,
				CategoryID:     goods.CategoryID,
				BrandsID:       goods.BrandsID,
				OnSale:         goods.OnSale,
				ShipFree:       goods.ShipFree,
				IsNew:          goods.IsNew,
				IsHot:          goods.IsHot,
				Name:           goods.Name,
				ClickNum:       goods.ClickNum,
				SoldNum:        goods.SoldNum,
				FavNum:         goods.FavNum,
				MarketPriceFen: goods.MarketPriceFen,
				GoodsBrief:     goods.GoodsBrief,
				ShopPriceFen:   goods.ShopPriceFen,
			}
			if err := searchFactory.Goods().Update(ctx, &searchDoc); err != nil {
				return err
			}
			total++
		}
		page++
	}

	log.Infof("reindexed %d goods documents", total)
	return nil
}
