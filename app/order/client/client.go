package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	v1 "goshop/api/order/v1"
	appclient "goshop/app/pkg/client"
	"goshop/app/pkg/options"
	rpc "goshop/gmicro/server/rpcserver"
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
		appclient.ServiceOrder,
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
