package controller

import (
	"net/http"

	upbv1 "goshop/api/user/v1"

	"github.com/gin-gonic/gin"
)

type replaceResourceScopesRequest struct {
	Scopes []*upbv1.UserResourceScope `json:"scopes" binding:"required"`
}

func (us *userServer) ReplaceResourceScopes(ctx *gin.Context) {
	userID, ok := parseUserID(ctx)
	if !ok {
		return
	}
	var request replaceResourceScopesRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": "invalid resource scopes"})
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}
	resp, err := us.users.ReplaceUserResourceScopes(ctx, &upbv1.ReplaceUserResourceScopesRequest{UserId: int32(userID), Scopes: request.Scopes, Actor: actor})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"code": http.StatusBadGateway, "msg": "replace resource scopes failed"})
		return
	}
	if _, err = us.tokenVersions.Bump(ctx, userID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "msg": "resource scope token invalidation failed"})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
