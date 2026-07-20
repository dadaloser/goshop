package api

import (
	"context"
	"net/http"

	"goshop/app/goshop/api/config"
	"goshop/app/goshop/api/internal/controller/goods/v1"
	inventory "goshop/app/goshop/api/internal/controller/inventory/v1"
	orderv1 "goshop/app/goshop/api/internal/controller/order/v1"
	v12 "goshop/app/goshop/api/internal/controller/sms/v1"
	"goshop/app/goshop/api/internal/controller/user/v1"
	"goshop/app/goshop/api/internal/data/rpc"
	"goshop/app/goshop/api/internal/loginattempt"
	"goshop/app/goshop/api/internal/service"
	"goshop/app/goshop/api/internal/smsattempt"
	"goshop/app/goshop/api/internal/smscode"
	"goshop/app/goshop/api/internal/smslimit"
	"goshop/app/pkg/authsession/tokenrevocation"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver"

	"github.com/gin-gonic/gin"
)

// 初始化路由
func initRouter(ctx context.Context, g *restserver.Server, cfg *config.Config) error {
	if cfg != nil && cfg.Server != nil && cfg.Server.ManagementPort > 0 {
		registerBusinessLivez(g)
	}
	v1 := g.Group("/v1")
	uGroup := v1.Group("/user")

	data, err := rpc.GetDataFactoryOr(ctx, cfg.Registry, cfg.RPC, cfg.RPCClientResilience)
	if err != nil {
		return err
	}

	codeStore := smscode.NewRedisStore()
	loginAttempts := loginattempt.NewRedisStore()
	smsAttempts := smsattempt.NewRedisStore()
	smsLimiter := smslimit.NewRedisStore()
	revokedTokens := tokenrevocation.NewRedisStore()
	tokenVersions := tokenversion.NewRedisStore()
	//原来的过程其实很复杂
	serviceFactory := service.NewService(data, cfg.Sms, cfg.Jwt, codeStore, loginAttempts, smsAttempts, tokenVersions)
	uController := user.NewUserController(g.Translator(), serviceFactory, revokedTokens)
	{
		uGroup.POST("pwd_login", uController.Login)
		uGroup.POST("sms_login", uController.SmsLogin)
		uGroup.POST("register", uController.Register)

		jwtAuth, err := newJWTAuth(cfg.Jwt, revokedTokens, tokenVersions, data.Users())
		if err != nil {
			return err
		}
		uGroup.GET("detail", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionUserProfileReadSelf), uController.GetUserDetail)
		uGroup.PATCH("update", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionUserProfileUpdateSelf), uController.UpdateUser)
		uGroup.POST("logout", jwtAuth.AuthFunc(), uController.Logout)
		uGroup.POST("logout_all", jwtAuth.AuthFunc(), uController.LogoutAll)
		uGroup.DELETE("account", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionUserAccountDeleteSelf), uController.DeleteAccount)

		orderController := orderv1.NewOrderController(serviceFactory, g.Translator())
		uGroup.GET("cart_items", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionCartReadSelf), orderController.ListCartItems)
		uGroup.POST("cart_items", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionCartWriteSelf), orderController.CreateCartItem)
		uGroup.PATCH("cart_items", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionCartWriteSelf), orderController.UpdateCartItem)
		uGroup.DELETE("cart_items/:id", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionCartWriteSelf), orderController.DeleteCartItem)
		uGroup.POST("orders", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionOrderCreateSelf), orderController.SubmitOrder)
		uGroup.GET("orders", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionOrderReadSelf), orderController.OrderList)
		uGroup.GET("orders/:order_sn", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionOrderReadSelf), orderController.OrderDetail)
		uGroup.GET("orders/:order_sn/status_logs", jwtAuth.AuthFunc(), authz.RequirePermission(authz.PermissionOrderStatusLogReadSelf), orderController.OrderStatusLogs)
	}

	baseRouter := v1.Group("base")
	{
		smsCtl := v12.NewSmsController(serviceFactory, g.Translator(), codeStore, smsLimiter)
		baseRouter.POST("send_sms", smsCtl.SendSms)
		baseRouter.GET("captcha", user.GetCaptcha)
	}

	//商品相关的api
	goodsRouter := v1.Group("goods")
	{
		goodsController := goods.NewGoodsController(serviceFactory, g.Translator())
		goodsRouter.GET("", goodsController.List)
	}

	inventoryRouter := v1.Group("inventory")
	{
		inventoryController := inventory.NewInventoryController(serviceFactory, g.Translator())
		inventoryRouter.GET("/:goods_id", inventoryController.Detail)
	}

	return nil
}

func registerBusinessLivez(g *restserver.Server) {
	g.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
