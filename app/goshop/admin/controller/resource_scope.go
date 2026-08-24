package controller

import (
	upbv1 "goshop/api/user/v1"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
	core "goshop/pkg/transport/httperror"

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
		writePublicError(ctx, errcode.ErrValidation, apperrors.KindInvalidArgument, "invalid resource scopes")
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}
	resp, err := us.users.ReplaceUserResourceScopes(ctx, &upbv1.ReplaceUserResourceScopesRequest{UserId: int32(userID), Scopes: request.Scopes, Actor: actor})
	if err != nil {
		writeUserRPCError(ctx, err, "replace resource scopes failed")
		return
	}
	if _, err = us.tokenVersions.Bump(ctx, userID); err != nil {
		writePublicError(ctx, errcode.ErrUnknown, apperrors.KindInternal, "resource scope token invalidation failed")
		return
	}
	core.WriteResponse(ctx, nil, resp)
}
