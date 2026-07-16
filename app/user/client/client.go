package main

import (
	"context"
	"fmt"
	"goshop/api/user/v1"
	appclient "goshop/app/pkg/client"
	"goshop/app/pkg/options"
	rpc "goshop/gmicro/server/rpcserver"
	"goshop/gmicro/server/rpcserver/selector"
	"goshop/gmicro/server/rpcserver/selector/random"
	"time"

	"google.golang.org/grpc"
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
		appclient.ServiceUser,
		rpc.WithBalancerName("selector"),
		rpc.WithClientTimeout(time.Second*5000),
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
