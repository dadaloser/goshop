package api

import (
	"context"

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
	"goshop/app/goshop/api/internal/tokenrevocation"
	"goshop/app/goshop/api/internal/tokenversion"
	"goshop/gmicro/server/restserver"
)

// 初始化路由
func initRouter(ctx context.Context, g *restserver.Server, cfg *config.Config) error {
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

		jwtAuth, err := newJWTAuth(cfg.Jwt, revokedTokens, tokenVersions)
		if err != nil {
			return err
		}
		uGroup.GET("detail", jwtAuth.AuthFunc(), uController.GetUserDetail)
		uGroup.PATCH("update", jwtAuth.AuthFunc(), uController.UpdateUser)
		uGroup.POST("logout", jwtAuth.AuthFunc(), uController.Logout)
		uGroup.POST("logout_all", jwtAuth.AuthFunc(), uController.LogoutAll)
		uGroup.DELETE("account", jwtAuth.AuthFunc(), uController.DeleteAccount)

		orderController := orderv1.NewOrderController(serviceFactory, g.Translator())
		uGroup.GET("cart_items", jwtAuth.AuthFunc(), orderController.ListCartItems)
		uGroup.POST("cart_items", jwtAuth.AuthFunc(), orderController.CreateCartItem)
		uGroup.PATCH("cart_items", jwtAuth.AuthFunc(), orderController.UpdateCartItem)
		uGroup.DELETE("cart_items/:id", jwtAuth.AuthFunc(), orderController.DeleteCartItem)
		uGroup.POST("orders", jwtAuth.AuthFunc(), orderController.SubmitOrder)
		uGroup.GET("orders", jwtAuth.AuthFunc(), orderController.OrderList)
		uGroup.GET("orders/:order_sn", jwtAuth.AuthFunc(), orderController.OrderDetail)
		uGroup.GET("orders/:order_sn/status_logs", jwtAuth.AuthFunc(), orderController.OrderStatusLogs)
		uGroup.POST("orders/pay/callback", jwtAuth.AuthFunc(), orderController.SimulatePayCallback)
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
		inventoryRouter.GET("/orders/:order_sn", inventoryController.OrderDetail)
		inventoryRouter.GET("/:goods_id", inventoryController.Detail)
	}

	return nil
}
