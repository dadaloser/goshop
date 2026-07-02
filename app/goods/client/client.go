package main

import (
	"context"
	"fmt"
	v1 "goshop/api/goods/v1"
	appclient "goshop/app/pkg/client"
	"goshop/gmicro/registry/consul"
	rpc "goshop/gmicro/server/rpcserver"
	_ "goshop/gmicro/server/rpcserver/resolver/direct"
	"goshop/gmicro/server/rpcserver/selector"
	"goshop/gmicro/server/rpcserver/selector/random"
	"time"

	"github.com/hashicorp/consul/api"
)

func main() {
	//设置全局的负载均衡策略
	selector.SetGlobalSelector(random.NewBuilder())
	rpc.InitBuilder()

	conf := api.DefaultConfig()
	conf.Address = "192.168.1.92:8500"
	conf.Scheme = "http"
	cli, err := api.NewClient(conf)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(true))

	conn, err := rpc.DialInsecure(context.Background(),
		rpc.WithBalancerName("selector"),
		rpc.WithDiscovery(r),
		rpc.WithClientTimeout(time.Second*5000),
		rpc.WithConnectProbe(true),
		rpc.WithEndpoint(appclient.ServiceEndpoint(appclient.ServiceGoods)),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	uc := v1.NewGoodsClient(conn)

	re, err := uc.GoodsList(context.Background(), &v1.GoodsFilterRequest{
		KeyWords: "猕猴桃",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(re)

}
