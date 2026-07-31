package user

import (
	"strconv"
	"strings"

	"goshop/gmicro/errcode"
	"goshop/pkg/common/core"

	"github.com/gin-gonic/gin"
)

type deviceBlacklistForm struct {
	UserID   uint64 `json:"user_id" binding:"required,min=1"`
	DeviceID string `json:"device_id" binding:"required,max=128"`
}

func (us *userServer) ListDevices(ctx *gin.Context) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	service, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	page, pageSize := positiveQuery(ctx, "pn", 1), positiveQuery(ctx, "pSize", 20)
	result, err := service.ListDevices(ctx, userID, page, pageSize)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"total": result.TotalCount, "items": result.Items})
}

func (us *userServer) LogoutDevice(ctx *gin.Context) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	service, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	sessionID := strings.TrimSpace(ctx.Param("session_id"))
	if err := service.LogoutDevice(ctx, userID, sessionID); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"ok": true, "session_id": sessionID})
}

func positiveQuery(ctx *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(ctx.Query(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func (us *userServer) ListDeviceBlacklist(ctx *gin.Context) {
	service, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	result, err := service.ListDeviceBlacklist(ctx, positiveQuery(ctx, "pn", 1), positiveQuery(ctx, "pSize", 20))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"total": result.TotalCount, "items": result.Items})
}
func (us *userServer) AddDeviceBlacklist(ctx *gin.Context) {
	form := deviceBlacklistForm{}
	if err := ctx.ShouldBindJSON(&form); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	service, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if err := service.AddDeviceBlacklist(ctx, form.UserID, strings.TrimSpace(form.DeviceID)); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"ok": true, "user_id": form.UserID, "device_id": strings.TrimSpace(form.DeviceID)})
}
func (us *userServer) DeleteDeviceBlacklist(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Query("user_id"), 10, 64)
	if err != nil || userID == 0 {
		core.WriteResponse(ctx, errcode.NewValidationError("user_id is required"), nil)
		return
	}
	deviceID := strings.TrimSpace(ctx.Param("device_id"))
	service, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if err := service.DeleteDeviceBlacklist(ctx, userID, deviceID); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"ok": true, "user_id": userID, "device_id": deviceID})
}
