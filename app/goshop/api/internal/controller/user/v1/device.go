package user

import (
	"goshop/app/pkg/errorcatalog"
	"strconv"
	"strings"
	"time"

	"goshop/pkg/common/core"

	"github.com/gin-gonic/gin"
)

type deviceBlacklistForm struct {
	UserID   uint64 `json:"user_id" binding:"required,min=1"`
	DeviceID string `json:"device_id" binding:"required,max=128"`
}

type selfDeviceBlacklistForm struct {
	DeviceID string `json:"device_id" binding:"required,max=128"`
}

type userDeviceResponse struct {
	SessionID       string    `json:"session_id"`
	DeviceID        string    `json:"device_id"`
	Device          string    `json:"device"`
	IPAddress       string    `json:"ip_address"`
	Location        string    `json:"location,omitempty"`
	LastOperationAt time.Time `json:"last_operation_at"`
	Active          bool      `json:"active"`
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
	items := make([]userDeviceResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, userDeviceResponse{
			SessionID:       item.ID,
			DeviceID:        item.DeviceID,
			Device:          item.DeviceName,
			IPAddress:       item.ClientIP,
			Location:        item.Location,
			LastOperationAt: item.LastUsedAt,
			Active:          item.Active,
		})
	}
	core.WriteResponse(ctx, nil, gin.H{"total": result.TotalCount, "items": items})
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
	core.WriteResponse(ctx, nil, gin.H{"msg": "Device logged out successfully", "session_id": sessionID})
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
	result, err := service.ListDeviceBlacklist(ctx, 0, positiveQuery(ctx, "pn", 1), positiveQuery(ctx, "pSize", 20))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"total": result.TotalCount, "items": result.Items})
}

// ListOwnDeviceBlacklist returns only the authenticated user's blacklist.
func (us *userServer) ListOwnDeviceBlacklist(ctx *gin.Context) {
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
	result, err := service.ListDeviceBlacklist(ctx, userID, positiveQuery(ctx, "pn", 1), positiveQuery(ctx, "pSize", 20))
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	items := make([]gin.H, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, gin.H{"device_id": item.DeviceID, "created_at": item.CreatedAt})
	}
	core.WriteResponse(ctx, nil, gin.H{"total": result.TotalCount, "items": items})
}

// AddOwnDeviceBlacklist adds a device to the authenticated user's blacklist.
func (us *userServer) AddOwnDeviceBlacklist(ctx *gin.Context) {
	form := selfDeviceBlacklistForm{}
	if err := ctx.ShouldBindJSON(&form); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
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
	deviceID := strings.TrimSpace(form.DeviceID)
	if err := service.AddDeviceBlacklist(ctx, userID, deviceID); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"msg": true, "device_id": deviceID})
}

// DeleteOwnDeviceBlacklist removes a device from the authenticated user's blacklist.
func (us *userServer) DeleteOwnDeviceBlacklist(ctx *gin.Context) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	deviceID := strings.TrimSpace(ctx.Param("device_id"))
	if deviceID == "" {
		core.WriteResponse(ctx, errorcatalog.NewValidationError("device_id is required"), nil)
		return
	}
	service, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if err := service.DeleteDeviceBlacklist(ctx, userID, deviceID); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"msg": true, "device_id": deviceID})
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
	core.WriteResponse(ctx, nil, gin.H{"msg": true, "user_id": form.UserID, "device_id": strings.TrimSpace(form.DeviceID)})
}
func (us *userServer) DeleteDeviceBlacklist(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Query("user_id"), 10, 64)
	if err != nil || userID == 0 {
		core.WriteResponse(ctx, errorcatalog.NewValidationError("user_id is required"), nil)
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
	core.WriteResponse(ctx, nil, gin.H{"msg": true, "user_id": userID, "device_id": deviceID})
}
