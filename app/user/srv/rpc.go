package srv

import (
	"fmt"
	upb "goshop/api/user/v1"
	"goshop/app/pkg/options"
	"goshop/gmicro/core/trace"
	"goshop/gmicro/server/rpcserver"
	"goshop/pkg/log"

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
		Username:    opts.User,
		Password:    opts.Password,
	}

	log.Infof("creating nacos data source: host=%s port=%d namespace=%s group=%s dataid=%s",
		opts.Host, opts.Port, opts.Namespace, opts.Group, opts.DataId)
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

func NewUserRPCServer(telemetry *options.TelemetryOptions, serverOpts *options.ServerOptions, nacosOpts *options.NacosOptions, rpcSecurity *options.RPCSecurityOptions, uServer upb.UserServer) (*rpcserver.Server, error) {
	//初始化open-telemetry的exporter
	log.Infof("initializing telemetry: name=%s endpoint=%s batcher=%s", telemetry.Name, telemetry.Endpoint, telemetry.Batcher)
	if err := trace.InitAgent(trace.Options{
		Name:     telemetry.Name,
		Endpoint: telemetry.Endpoint,
		Sampler:  telemetry.Sampler,
		Batcher:  telemetry.Batcher,
	}); err != nil {
		return nil, err
	}

	rpcAddr := fmt.Sprintf("%s:%d", serverOpts.Host, serverOpts.Port)
	tlsConfig, err := rpcSecurity.LoadServerTLSConfig()
	if err != nil {
		return nil, err
	}

	var opts []rpcserver.ServerOption
	opts = append(opts, rpcserver.WithAddress(rpcAddr))
	opts = append(opts, rpcserver.WithServerTLSConfig(tlsConfig))
	if serverOpts.EnableLimit {
		log.Infof("initializing sentinel limit rules from nacos")
		dataNacos, err := NewNacosDataSource(nacosOpts)
		if err != nil {
			return nil, err
		}
		opts = append(opts, rpcserver.WithUnaryInterceptor(grpc.NewUnaryServerInterceptor())) //添加限流拦截器
		if err := dataNacos.Initialize(); err != nil {
			return nil, err
		}
		log.Infof("sentinel limit rules initialized from nacos")
	} else {
		log.Infof("sentinel limit disabled; skip nacos data source initialization")
	}
	log.Infof("creating user rpc server: address=%s", rpcAddr)
	uRpcServer, err := rpcserver.NewServerE(opts...)
	if err != nil {
		return nil, err
	}

	upb.RegisterUserServer(uRpcServer.Server, uServer)

	//r := gin.Default()
	//upb.RegisterUserServerHTTPServer(uServer, r)
	//r.Run(":8075")
	return uRpcServer, nil
}
