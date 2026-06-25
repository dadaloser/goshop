package srv

import (
	"fmt"
	upb "goshop/api/user/v1"
	"goshop/app/pkg/options"
	"goshop/gmicro/core/trace"
	"goshop/gmicro/server/rpcserver"

	"github.com/alibaba/sentinel-golang/ext/datasource"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"

	"github.com/alibaba/sentinel-golang/pkg/adapters/grpc"
	"github.com/alibaba/sentinel-golang/pkg/datasource/nacos"
)

func NewNacosDataSource(opts *options.NacosOptions) (*nacos.NacosDataSource, error) {
	//nacos server地址
	sc := []constant.ServerConfig{
		{
			ContextPath: "/nacos",
			Port:        opts.Port,
			IpAddr:      opts.Host,
		},
	}

	//nacos client 相关参数配置,具体配置可参考github.com/nacos-group/nacos-sdk-go
	cc := constant.ClientConfig{
		NamespaceId: opts.Namespace,
		TimeoutMs:   5000,
	}

	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		return nil, err
	}

	//注册流控规则Handler
	h := datasource.NewFlowRulesHandler(datasource.FlowRuleJsonArrayParser)
	//创建NacosDataSource数据源
	nds, err := nacos.NewNacosDataSource(client, opts.Group, opts.DataId, h)
	if err != nil {
		return nil, err
	}
	return nds, nil
}

func NewUserRPCServer(telemetry *options.TelemetryOptions, serverOpts *options.ServerOptions, uServer upb.UserServer, dataNacos *nacos.NacosDataSource) (*rpcserver.Server, error) {
	//初始化open-telemetry的exporter
	trace.InitAgent(trace.Options{
		Name:     telemetry.Name,
		Endpoint: telemetry.Endpoint,
		Sampler:  telemetry.Sampler,
		Batcher:  telemetry.Batcher,
	})

	rpcAddr := fmt.Sprintf("%s:%d", serverOpts.Host, serverOpts.Port)

	var opts []rpcserver.ServerOption
	opts = append(opts, rpcserver.WithAddress(rpcAddr))
	if serverOpts.EnableLimit {
		opts = append(opts, rpcserver.WithUnaryInterceptor(grpc.NewUnaryServerInterceptor())) //添加限流拦截器
		//我去初始化nacos
		err := dataNacos.Initialize()
		if err != nil {
			return nil, err
		}
	}
	uRpcServer := rpcserver.NewServer(opts...)

	upb.RegisterUserServer(uRpcServer.Server, uServer)

	//r := gin.Default()
	//upb.RegisterUserServerHTTPServer(uServer, r)
	//r.Run(":8075")
	return uRpcServer, nil
}
