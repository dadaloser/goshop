package user

import (
	"goshop/pkg/common/core"

	"github.com/gin-gonic/gin"
)

func (us *userServer) Logout(ctx *gin.Context) {
	core.WriteResponse(ctx, nil, gin.H{"ok": true})
}
