package request

type GoodsFilter struct {
	PriceMinFen int64  `form:"pmin_fen"`
	PriceMaxFen int64  `form:"pmax_fen"`
	IsHot       bool   `form:"ih"`
	IsNew       bool   `form:"in"`
	IsTab       bool   `form:"it"`
	TopCategory int32  `form:"c"`
	Pages       int32  `form:"p"`
	PagePerNums int32  `form:"pnum"`
	KeyWords    string `form:"q"`
	Brand       int32  `form:"b"`
}
