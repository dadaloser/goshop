package goods

import (
	proto "goshop/api/goods/v1"
	"goshop/app/goshop/api/internal/domain/request"
	"goshop/app/goshop/api/internal/service"
	"goshop/app/pkg/code"
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type goodsController struct {
	trans ut.Translator
	srv   service.ServiceFactory
}

func NewGoodsController(srv service.ServiceFactory, trans ut.Translator) *goodsController {
	return &goodsController{
		srv:   srv,
		trans: trans,
	}
}

func (gc *goodsController) List(ctx *gin.Context) {
	log.Info("goods list function called ...")

	if gc == nil || gc.srv == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "goods service is not initialized"), nil)
		return
	}

	var r request.GoodsFilter

	if err := ctx.ShouldBindQuery(&r); err != nil {
		gin2.HandleValidatorError(ctx, err, gc.trans)
		return
	}

	gfr := proto.GoodsFilterRequest{
		IsNew:       r.IsNew,
		IsHot:       r.IsHot,
		PriceMax:    r.PriceMax,
		PriceMin:    r.PriceMin,
		TopCategory: r.TopCategory,
		Brand:       r.Brand,
		KeyWords:    r.KeyWords,
		Pages:       r.Pages,
		PagePerNums: r.PagePerNums,
	}

	goodsSrv := gc.srv.Goods()
	if goodsSrv == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "goods service is not initialized"), nil)
		return
	}

	goodsDTOList, err := goodsSrv.List(ctx, &gfr)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if goodsDTOList == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "goods service response is empty"), nil)
		return
	}

	reMap := map[string]interface{}{
		"total": goodsDTOList.GetTotal(),
	}
	goodsList := make([]interface{}, 0)
	for _, value := range goodsDTOList.GetData() {
		if value == nil {
			continue
		}
		category := value.GetCategory()
		brand := value.GetBrand()
		goodsList = append(goodsList, map[string]interface{}{
			"id":             value.GetId(),
			"name":           value.GetName(),
			"goods_brief":    value.GetGoodsBrief(),
			"desc":           value.GetGoodsDesc(),
			"ship_free":      value.GetShipFree(),
			"images":         value.GetImages(),
			"desc_images":    value.GetDescImages(),
			"front_image":    value.GetGoodsFrontImage(),
			"shop_price":     value.GetShopPrice(),
			"shop_price_fen": value.GetShopPriceFen(),
			"category": map[string]interface{}{
				"id":   category.GetId(),
				"name": category.GetName(),
			},
			"brand": map[string]interface{}{
				"id":   brand.GetId(),
				"name": brand.GetName(),
				"logo": brand.GetLogo(),
			},
			"is_hot":  value.GetIsHot(),
			"is_new":  value.GetIsNew(),
			"on_sale": value.GetOnSale(),
		})
	}
	reMap["data"] = goodsList

	core.WriteResponse(ctx, nil, reMap)
}
