package v1

import (
	"goshop/app/goshop/api/internal/domain/request"
	"goshop/app/goshop/api/internal/service"
	"goshop/app/pkg/code"
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type inventoryController struct {
	trans ut.Translator
	sf    service.ServiceFactory
}

func NewInventoryController(sf service.ServiceFactory, trans ut.Translator) *inventoryController {
	return &inventoryController{sf: sf, trans: trans}
}

func (ic *inventoryController) Detail(ctx *gin.Context) {
	if ic == nil || ic.sf == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "inventory service is not initialized"), nil)
		return
	}

	var uri request.InventoryDetailURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		gin2.HandleValidatorError(ctx, err, ic.trans)
		return
	}

	inventorySrv := ic.sf.Inventory()
	if inventorySrv == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "inventory service is not initialized"), nil)
		return
	}

	inv, err := inventorySrv.Detail(ctx, uint64(uri.GoodsID))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if inv == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "inventory service response is empty"), nil)
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
		"goods_id":  inv.GetGoodsId(),
		"num":       inv.GetNum(),
		"total":     inv.GetTotal(),
		"available": inv.GetAvailable(),
		"locked":    inv.GetLocked(),
		"sold":      inv.GetSold(),
	})
}
