package es

import (
	"context"
	"encoding/json"
	proto "goshop/api/goods/v1"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
	"strconv"

	v1 "goshop/app/goods/srv/internal/data_search/v1"
	"goshop/app/goods/srv/internal/domain/do"

	"github.com/olivere/elastic/v7"
)

type goods struct {
	esClient *elastic.Client
}

func newGoods(ds *esDataSearch) *goods {
	return &goods{esClient: ds.esClient}
}

func NewGoods(esClient *elastic.Client) *goods {
	return &goods{esClient: esClient}
}

func (g *goods) Create(ctx context.Context, goods *do.GoodsSearchDO) error {
	if goods == nil || goods.ID <= 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "goods is required")
	}

	_, err := g.esClient.Index().
		Index(goods.GetIndexName()).
		Id(strconv.Itoa(int(goods.ID))).
		BodyJson(&goods).
		Do(ctx)
	return err
}

func (g *goods) Delete(ctx context.Context, ID uint64) error {
	if ID == 0 {
		return errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}

	_, err := g.esClient.Delete().Index(do.GoodsSearchDO{}.GetIndexName()).Id(strconv.Itoa(int(ID))).Refresh("true").Do(ctx)
	if elastic.IsNotFound(err) {
		return nil
	}
	return err
}

func (g *goods) Update(ctx context.Context, goods *do.GoodsSearchDO) error {
	if goods == nil || goods.ID <= 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "goods is required")
	}

	return g.Create(ctx, goods)
}

func (g *goods) Search(ctx context.Context, req *v1.GoodsFilterRequest) (*do.GoodsSearchDOList, error) {
	if req == nil {
		req = &v1.GoodsFilterRequest{}
	}
	if req.GoodsFilterRequest == nil {
		req.GoodsFilterRequest = &proto.GoodsFilterRequest{}
	}

	q := buildGoodsSearchQuery(req)

	//分页
	if req.Pages == 0 {
		req.Pages = 1
	}

	switch {
	case req.PagePerNums > 100:
		req.PagePerNums = 100
	case req.PagePerNums <= 0:
		req.PagePerNums = 10
	}

	res, err := g.esClient.Search().Index(do.GoodsSearchDO{}.GetIndexName()).
		Query(q).
		From(int(req.Pages-1) * int(req.PagePerNums)).
		Size(int(req.PagePerNums)).Do(ctx)

	if err != nil {
		return nil, errors.WithCode(code.ErrGoodsNotFound, err.Error())
	}

	var ret do.GoodsSearchDOList
	ret.TotalCount = res.Hits.TotalHits.Value
	for _, value := range res.Hits.Hits {
		goods := do.GoodsSearchDO{}
		err := json.Unmarshal(value.Source, &goods)
		if err != nil {
			return nil, errors.WithCode(code.ErrEsUnmarshal, err.Error())
		}
		ret.Items = append(ret.Items, &goods)
	}
	return &ret, err
}

func buildGoodsSearchQuery(req *v1.GoodsFilterRequest) *elastic.BoolQuery {
	q := elastic.NewBoolQuery()
	if !req.IncludeOffSale {
		q = q.Filter(elastic.NewTermQuery("on_sale", true))
	}
	if req.SpuCode != "" {
		q = q.Filter(elastic.NewTermQuery("spu_code.keyword", req.SpuCode))
	}
	if req.SkuCode != "" {
		q = q.Filter(elastic.NewTermQuery("sku_code.keyword", req.SkuCode))
	}
	if req.KeyWords != "" {
		q = q.Must(elastic.NewMultiMatchQuery(req.KeyWords, "name", "goods_brief"))
	}
	if req.IsHot {
		q = q.Filter(elastic.NewTermQuery("is_hot", req.IsHot))
	}
	if req.IsNew {
		q = q.Filter(elastic.NewTermQuery("is_new", req.IsNew))
	}

	if req.PriceMinFen > 0 {
		q = q.Filter(elastic.NewRangeQuery("shop_price_fen").Gte(req.PriceMinFen))
	}
	if req.PriceMaxFen > 0 {
		q = q.Filter(elastic.NewRangeQuery("shop_price_fen").Lte(req.PriceMaxFen))
	}

	if req.Brand > 0 {
		q = q.Filter(elastic.NewTermQuery("brands_id", req.Brand))
	}

	if req.TopCategory > 0 {
		q = q.Filter(elastic.NewTermsQuery("category_id", req.CategoryIDs...))
	}

	return q
}

var _ v1.GoodsStore = &goods{}
