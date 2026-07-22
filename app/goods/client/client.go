package main

import (
	"context"
	"fmt"
	v1 "goshop/api/goods/v1"
	appclient "goshop/app/pkg/client"
	"goshop/app/pkg/options"
	rpc "goshop/gmicro/server/rpcserver"
	"goshop/gmicro/server/rpcserver/selector"
	"goshop/gmicro/server/rpcserver/selector/random"
	"time"
)

func main() {
	//设置全局的负载均衡策略
	selector.SetGlobalSelector(random.NewBuilder())
	registry := &options.RegistryOptions{
		Address: "192.168.1.92:8500",
		Scheme:  "http",
	}
	rpcSecurity := &options.RPCSecurityOptions{
		CertFile:   "configs/tls/dev/internal.crt",
		KeyFile:    "configs/tls/dev/internal.key",
		CAFile:     "configs/tls/dev/internal.crt",
		ServerName: "goshop.internal",
	}

	conn, err := appclient.DialService(
		context.Background(),
		registry,
		rpcSecurity,
		appclient.ServiceGoods,
		rpc.WithBalancerName("selector"),
		rpc.WithClientTimeout(time.Second*5000),
	)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = conn.Close()
	}()

	uc := v1.NewGoodsClient(conn)

	re, err := uc.GoodsList(context.Background(), &v1.GoodsFilterRequest{
		KeyWords: "猕猴桃",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(re)

}
