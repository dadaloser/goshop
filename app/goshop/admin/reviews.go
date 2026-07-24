package admin

import (
	"net/http"
	"strconv"
	"strings"

	rpb "goshop/api/review/v1"
	upb "goshop/api/user/v1"
	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/authsession/tokenrevocation"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver"

	"github.com/gin-gonic/gin"
)

func registerAdminReviewRoutes(server *restserver.Server, cfg *config.Config, users upb.UserClient, reviews rpb.ReviewClient) error {
	return registerAdminReviewRoutesWithStores(server, cfg, users, reviews, tokenrevocation.NewRedisStore(), tokenversion.NewRedisStore())
}

func registerAdminReviewRoutesWithStores(
	server *restserver.Server,
	cfg *config.Config,
	users upb.UserClient,
	reviews rpb.ReviewClient,
	revokedTokens tokenrevocation.Store,
	tokenVersions tokenversion.Store,
) error {
	auth, err := newStaffJWTAuth(cfg.Jwt, revokedTokens, tokenVersions, users)
	if err != nil {
		return err
	}
	group := server.Group("/v1/reviews", auth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff), requireRole(authz.StaffRoleReview, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainReview))
	h := &adminReviewHandler{client: reviews}
	group.GET("", authz.RequirePermission(authz.PermissionReviewModerateAny), h.list)
	group.POST(":id/moderate", authz.RequirePermission(authz.PermissionReviewModerateAny), h.moderate)
	group.POST(":id/reply", authz.RequirePermission(authz.PermissionReviewReplyAny), h.reply)
	group.POST("ratings/:goods_id/rebuild", authz.RequirePermission(authz.PermissionReviewAggregateRebuild), requireAdminConfirmation(cfg.AdminAuth), h.rebuild)
	return nil
}

type adminReviewHandler struct{ client rpb.ReviewClient }
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
	writeRPC(c, resp, err)
}
func (h *adminReviewHandler) moderate(c *gin.Context) {
	var f moderateForm
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid moderation"})
		return
	}
	id, ok := reviewID(c)
	if !ok {
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
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid reply"})
		return
	}
	id, ok := reviewID(c)
	if !ok {
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
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid goods id"})
		return
	}
	actor, _ := userIDFromClaims(c)
	resp, err := h.client.RebuildRating(c, &rpb.RebuildRatingRequest{GoodsId: int32(goods), ActorUserId: int32(actor), RequestId: requestID(c)})
	writeRPC(c, resp, err)
}
func reviewID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid review id"})
		return 0, false
	}
	return id, true
}
