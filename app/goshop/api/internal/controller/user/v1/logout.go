package user

import (
	"encoding/json"
	"net/http"
	"time"

	"goshop/gmicro/errcode"
	gauth "goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

/*
注销当前会话
*/

func (us *userServer) Logout(ctx *gin.Context) {
	userID, userErr := userIDFromContext(ctx)
	claims := gauth.ExtractClaims(ctx)
	sessionID, _ := claims["session_id"].(string)
	if userErr == nil && sessionID != "" {
		if userSrv, err := us.usersService(); err == nil {
			if err = userSrv.Logout(ctx, userID, sessionID); err != nil {
				core.WriteResponse(ctx, err, nil)
				return
			}
		}
	}
	if us.revokedTokens != nil {
		token, err := gauth.GetToken(ctx)
		if err != nil {
			core.WriteResponse(ctx, errors.NewSpec(errcode.TokenInvalidSpec, "token not found"), nil)
			return
		}

		expiresAt, err := jwtExpiresAt(ctx)
		if err != nil {
			core.WriteResponse(ctx, err, nil)
			return
		}
		if err = us.revokedTokens.Revoke(ctx.Request.Context(), token, expiresAt); err != nil {
			core.WriteResponse(ctx, errors.NewSpec(errcode.UnknownSpec, "revoke current token"), nil)
			return
		}
	}

	core.WriteResponse(ctx, nil, gin.H{"code": http.StatusOK, "msg": "退出成功!"})
}

/*
注销该用户的全部会话
*/

func (us *userServer) LogoutAll(ctx *gin.Context) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userSrv, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if err = userSrv.LogoutAll(ctx, userID); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"code": http.StatusOK, "msg": "退出成功!"})
}

func jwtExpiresAt(ctx *gin.Context) (time.Time, error) {
	exp, ok := gauth.ExtractClaims(ctx)["exp"]
	if !ok {
		return time.Time{}, errors.NewSpec(errcode.TokenInvalidSpec, "token missing exp")
	}

	var unix int64
	switch value := exp.(type) {
	case float64:
		unix = int64(value)
	case json.Number:
		v, err := value.Int64()
		if err != nil {
			return time.Time{}, errors.NewSpec(errcode.TokenInvalidSpec, "token exp invalid")
		}
		unix = v
	case int64:
		unix = value
	case int:
		unix = int64(value)
	default:
		return time.Time{}, errors.NewSpec(errcode.TokenInvalidSpec, "token exp invalid")
	}
	if unix <= 0 {
		return time.Time{}, errors.NewSpec(errcode.TokenInvalidSpec, "token exp invalid")
	}
	return time.Unix(unix, 0), nil
}
