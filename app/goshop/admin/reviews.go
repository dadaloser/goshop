package admin

import (
	"strconv"
	"strings"

	goodspb "goshop/api/goods/v1"
	rpb "goshop/api/review/v1"
	upb "goshop/api/user/v1"
	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/authsession/tokenrevocation"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver"
	gauth "goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
	core "goshop/pkg/transport/httperror"

	"github.com/gin-gonic/gin"
)

func registerAdminReviewRoutes(server *restserver.Server, cfg *config.Config, users upb.UserClient, goods goodspb.GoodsClient, reviews rpb.ReviewClient) error {
	return registerAdminReviewRoutesWithStores(server, cfg, users, goods, reviews, tokenrevocation.NewRedisStore(), tokenversion.NewRedisStore())
}

func registerAdminReviewRoutesWithStores(
	server *restserver.Server,
	cfg *config.Config,
	users upb.UserClient,
	goods goodspb.GoodsClient,
	reviews rpb.ReviewClient,
	revokedTokens tokenrevocation.Store,
	tokenVersions tokenversion.Store,
) error {
	auth, err := newStaffJWTAuth(cfg.Jwt, revokedTokens, tokenVersions, users)
	if err != nil {
		return err
	}
	group := server.Group("/v1/reviews", auth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff), requireRole(authz.StaffRoleReview, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainReview))
	h := &adminReviewHandler{client: reviews, goods: goods}
	group.GET("", authz.RequirePermission(authz.PermissionReviewModerateAny), h.list)
	group.POST(":id/moderate", authz.RequirePermission(authz.PermissionReviewModerateAny), h.moderate)
	group.POST(":id/reply", authz.RequirePermission(authz.PermissionReviewReplyAny), h.reply)
	group.POST("ratings/:goods_id/rebuild", authz.RequirePermission(authz.PermissionReviewAggregateRebuild), requireAdminConfirmation(cfg.AdminAuth), h.rebuild)
	return nil
}

type adminReviewHandler struct {
	client rpb.ReviewClient
	goods  goodspb.GoodsClient
}
type moderateForm struct {
	Decision string `json:"decision" binding:"required"`
	Reason   string `json:"reason"`
}
type replyForm struct {
	Content string `json:"content" binding:"required,max=2000"`
}

func (h *adminReviewHandler) list(c *gin.Context) {
	goods, _ := strconv.ParseInt(c.Query("goods_id"), 10, 32)
	resp, err := h.client.ListReviews(c, &rpb.ListReviewsRequest{GoodsId: int32(goods), Status: strings.ToUpper(strings.TrimSpace(c.Query("status"))), Page: 1, PageSize: 100})
	if err == nil && resp != nil && h.goods != nil {
		filtered := make([]*rpb.ReviewResponse, 0, len(resp.GetData()))
		for _, item := range resp.GetData() {
			goodsResp, goodsErr := h.goods.GetGoodsDetail(c, &goodspb.GoodInfoRequest{Id: item.GetGoodsId()})
			if goodsErr != nil {
				err = goodsErr
				break
			}
			if claimsAllowScope(gauth.ExtractClaims(c), authz.ResourceScope{Domain: string(authz.BusinessDomainReview), StoreID: strings.TrimSpace(goodsResp.GetStoreId()), ResourceType: "review", ResourceID: strconv.FormatInt(item.GetId(), 10)}) {
				filtered = append(filtered, item)
			}
		}
		if err == nil {
			resp.Data = filtered
			resp.Total = int32(len(filtered))
		}
	}
	writeRPC(c, resp, err)
}
func (h *adminReviewHandler) moderate(c *gin.Context) {
	var f moderateForm
	if err := c.ShouldBindJSON(&f); err != nil {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid moderation"))
		return
	}
	id, ok := reviewID(c)
	if !ok {
		return
	}
	review, err := h.client.GetReview(c, &rpb.GetReviewRequest{ReviewId: id})
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	goodsResp, err := h.goods.GetGoodsDetail(c, &goodspb.GoodInfoRequest{Id: review.GetGoodsId()})
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	if !claimsAllowScope(gauth.ExtractClaims(c), authz.ResourceScope{Domain: string(authz.BusinessDomainReview), StoreID: strings.TrimSpace(goodsResp.GetStoreId()), ResourceType: "review", ResourceID: strconv.FormatInt(review.GetId(), 10)}) {
		denyScope(c)
		return
	}
	actor, err := userIDFromClaims(c)
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	resp, err := h.client.ModerateReview(c, &rpb.ModerateReviewRequest{ReviewId: id, Decision: f.Decision, ActorUserId: int32(actor), RequestId: requestID(c), Reason: f.Reason})
	writeRPC(c, resp, err)
}
func (h *adminReviewHandler) reply(c *gin.Context) {
	var f replyForm
	if err := c.ShouldBindJSON(&f); err != nil {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid reply"))
		return
	}
	id, ok := reviewID(c)
	if !ok {
		return
	}
	review, err := h.client.GetReview(c, &rpb.GetReviewRequest{ReviewId: id})
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	goodsResp, err := h.goods.GetGoodsDetail(c, &goodspb.GoodInfoRequest{Id: review.GetGoodsId()})
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	if !claimsAllowScope(gauth.ExtractClaims(c), authz.ResourceScope{Domain: string(authz.BusinessDomainReview), StoreID: strings.TrimSpace(goodsResp.GetStoreId()), ResourceType: "review", ResourceID: strconv.FormatInt(review.GetId(), 10)}) {
		denyScope(c)
		return
	}
	actor, err := userIDFromClaims(c)
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	resp, err := h.client.ReplyReview(c, &rpb.ReplyReviewRequest{ReviewId: id, ActorUserId: int32(actor), Content: f.Content, RequestId: requestID(c)})
	writeRPC(c, resp, err)
}
func (h *adminReviewHandler) rebuild(c *gin.Context) {
	goods, err := strconv.ParseInt(c.Param("goods_id"), 10, 32)
	if err != nil || goods <= 0 {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid goods id"))
		return
	}
	if h.goods != nil {
		goodsResp, goodsErr := h.goods.GetGoodsDetail(c, &goodspb.GoodInfoRequest{Id: int32(goods)})
		if goodsErr != nil {
			writeRPC(c, nil, goodsErr)
			return
		}
		if !claimsAllowScope(gauth.ExtractClaims(c), authz.ResourceScope{Domain: string(authz.BusinessDomainReview), StoreID: strings.TrimSpace(goodsResp.GetStoreId()), ResourceType: "goods", ResourceID: strconv.Itoa(int(goods))}) {
			denyScope(c)
			return
		}
	}
	actor, _ := userIDFromClaims(c)
	resp, err := h.client.RebuildRating(c, &rpb.RebuildRatingRequest{GoodsId: int32(goods), ActorUserId: int32(actor), RequestId: requestID(c)})
	writeRPC(c, resp, err)
}
func reviewID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid review id"))
		return 0, false
	}
	return id, true
}
