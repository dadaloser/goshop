package api

import (
	"goshop/app/goshop/api/config"
	"goshop/app/goshop/api/internal/controller/goods/v1"
	v12 "goshop/app/goshop/api/internal/controller/sms/v1"
	"goshop/app/goshop/api/internal/controller/user/v1"
	"goshop/app/goshop/api/internal/data/rpc"
	"goshop/app/goshop/api/internal/service"
	"goshop/gmicro/server/restserver"
)

// 初始化路由
func initRouter(g *restserver.Server, cfg *config.Config) error {
	v1 := g.Group("/v1")
	uGroup := v1.Group("/user")

	data, err := rpc.GetDataFactoryOr(cfg.Registry)
	if err != nil {
		return err
	}

	//原来的过程其实很复杂
	serviceFactory := service.NewService(data, cfg.Sms, cfg.Jwt)
	uController := user.NewUserController(g.Translator(), serviceFactory)
	{
		uGroup.POST("pwd_login", uController.Login)
		uGroup.POST("register", uController.Register)

		jwtAuth := newJWTAuth(cfg.Jwt)
		uGroup.GET("detail", jwtAuth.AuthFunc(), uController.GetUserDetail)
		uGroup.PATCH("update", jwtAuth.AuthFunc(), uController.GetUserDetail)
	}

	baseRouter := v1.Group("base")
	{
		smsCtl := v12.NewSmsController(serviceFactory, g.Translator())
		baseRouter.POST("send_sms", smsCtl.SendSms)
		baseRouter.GET("captcha", user.GetCaptcha)
	}

	//商品相关的api
	goodsRouter := v1.Group("goods")
	{
		goodsController := goods.NewGoodsController(serviceFactory, g.Translator())
		goodsRouter.GET("", goodsController.List)
	}

	return nil
}
