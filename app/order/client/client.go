package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc"

	v1 "goshop/api/order/v1"
	appclient "goshop/app/pkg/client"
	"goshop/app/pkg/options"
	"goshop/gmicro/registry/consul"
	rpc "goshop/gmicro/server/rpcserver"
	_ "goshop/gmicro/server/rpcserver/resolver/direct"
	"goshop/gmicro/server/rpcserver/selector"
	"goshop/gmicro/server/rpcserver/selector/random"
	"math/rand"
	"time"
)

func generateOrderSn(userId int32) string {
	//订单号的生成规则
	/*
		年月日时分秒+用户id+2位随机数
	*/
	now := time.Now()
	rand.New(rand.NewSource(time.Now().UnixNano()))
	rand.New(rand.NewSource(time.Now().UnixNano()))
	orderSn := fmt.Sprintf("%d%d%d%d%d%d%d%d",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Nanosecond(),
		userId, rand.Intn(90)+10,
	)
	return orderSn
}

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
	rpcSecurity := &options.RPCSecurityOptions{
		CertFile:   "configs/tls/dev/internal.crt",
		KeyFile:    "configs/tls/dev/internal.key",
		CAFile:     "configs/tls/dev/internal.crt",
		ServerName: "goshop.internal",
	}
	tlsConfig, err := rpcSecurity.LoadClientTLSConfig()
	if err != nil {
		panic(err)
	}

	conn, err := rpc.Dial(context.Background(),
		rpc.WithBalancerName("selector"),
		rpc.WithDiscovery(r),
		rpc.WithClientTLSConfig(tlsConfig),
		rpc.WithClientTimeout(time.Second*5000),
		rpc.WithConnectProbe(true),
		rpc.WithEndpoint(appclient.ServiceEndpoint(appclient.ServiceOrder)),
	)
	if err != nil {
		panic(err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}(conn)

	uc := v1.NewOrderClient(conn)

	_, err = uc.SubmitOrder(context.Background(), &v1.OrderRequest{
		UserId:  1,
		Address: "慕课网",
		OrderSn: generateOrderSn(1),
		Name:    "bobby",
		Post:    "尽快发货",
		Mobile:  "18787878787",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("订单创建成功")
}
