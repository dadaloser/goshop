package review

import (
	"net/http"
	"strconv"
	"strings"

	rpb "goshop/api/review/v1"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/common/core"

	"github.com/gin-gonic/gin"
)

type Controller struct{ client rpb.ReviewClient }

func New(client rpb.ReviewClient) *Controller { return &Controller{client: client} }

type createForm struct {
	OrderSN string `json:"order_sn" binding:"required"`
	GoodsID int32  `json:"goods_id" binding:"required,gt=0"`
	Rating  int32  `json:"rating" binding:"required,min=1,max=5"`
	Content string `json:"content" binding:"required,max=2000"`
}
type appendForm struct {
	Content string `json:"content" binding:"required,max=2000"`
}

func (c *Controller) Create(ctx *gin.Context) {
	var f createForm
	if err := ctx.ShouldBindJSON(&f); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "invalid review"})
		return
	}
	id, ok := userID(ctx)
	if !ok {
		return
	}
	resp, err := c.client.CreateReview(ctx, &rpb.CreateReviewRequest{UserId: id, OrderSn: strings.TrimSpace(f.OrderSN), GoodsId: f.GoodsID, Rating: f.Rating, Content: strings.TrimSpace(f.Content)})
	core.WriteResponse(ctx, err, resp)
}
func (c *Controller) Append(ctx *gin.Context) {
	var f appendForm
	if err := ctx.ShouldBindJSON(&f); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "invalid append"})
		return
	}
	id, ok := userID(ctx)
	if !ok {
		return
	}
	reviewID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "invalid review id"})
		return
	}
	resp, err := c.client.AppendReview(ctx, &rpb.AppendReviewRequest{UserId: id, ReviewId: reviewID, Content: strings.TrimSpace(f.Content)})
	core.WriteResponse(ctx, err, resp)
}
func (c *Controller) List(ctx *gin.Context) {
	goods, _ := strconv.ParseInt(ctx.Param("goods_id"), 10, 32)
	resp, err := c.client.ListReviews(ctx, &rpb.ListReviewsRequest{GoodsId: int32(goods), Status: "APPROVED", Page: 1, PageSize: 100})
	core.WriteResponse(ctx, err, resp)
}
func (c *Controller) Rating(ctx *gin.Context) {
	goods, _ := strconv.ParseInt(ctx.Param("goods_id"), 10, 32)
	resp, err := c.client.GetRating(ctx, &rpb.GetRatingRequest{GoodsId: int32(goods)})
	core.WriteResponse(ctx, err, resp)
}
func userID(ctx *gin.Context) (int32, bool) {
	raw, ok := ctx.Get(middlewares.KeyUserID)
	value, valid := raw.(float64)
	if !ok || !valid || value <= 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "user id missing"})
		return 0, false
	}
	return int32(value), true
}
