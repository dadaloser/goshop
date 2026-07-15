package main

import (
	"context"
	"fmt"
	"goshop/api/user/v1"
	appclient "goshop/app/pkg/client"
	"goshop/app/pkg/options"
	"goshop/gmicro/registry/consul"
	rpc "goshop/gmicro/server/rpcserver"
	_ "goshop/gmicro/server/rpcserver/resolver/direct"
	"goshop/gmicro/server/rpcserver/selector"
	"goshop/gmicro/server/rpcserver/selector/random"
	"time"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
)

func main() {
	//设置全局的负载均衡策略
	selector.SetGlobalSelector(random.NewBuilder())
	rpc.InitBuilder()

	conf := api.DefaultConfig()
	conf.Address = "192.168.1.92:8500"
	conf.Scheme = "http"
	client, err := api.NewClient(conf)
	if err != nil {
		panic(err)
	}
	r := consul.New(client, consul.WithHealthCheck(true))
	rpcSecurity := &options.RPCSecurityOptions{
		CertFile:   "configs/tls/dev/internal.crt",
		KeyFile:    "configs/tls/dev/internal.key",
		CAFile:     "configs/tls/dev/internal.crt",
		ServerName: "goshop.internal",
	}

	conn, err := rpc.DialDiscovery(context.Background(),
		rpc.WithBalancerName("selector"),
		rpc.WithDiscovery(r),
		rpc.WithClientSecurityPolicy(rpcSecurity),
		rpc.WithClientTimeout(time.Second*5000),
		rpc.WithEndpoint(appclient.ServiceEndpoint(appclient.ServiceUser)),
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

	uc := v1.NewUserClient(conn)

	for {
		res, err := uc.GetUserList(context.Background(), &v1.PageInfo{})
		if err != nil {
			panic(err)
		}

		fmt.Println(res)
		fmt.Println("success")
		time.Sleep(time.Millisecond * 2)
	}

}
