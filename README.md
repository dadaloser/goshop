| 目录                        | 职责                                                        
| [app/user/srv]             | 用户微服务，负责用户创建、查询、手机号查询、密码校验、更新等 gRPC 接口        
| [app/goods/srv]            | 商品微服务，负责商品、品牌、分类、Banner、分类品牌关系；同时有 ES 搜索层     
| [app/inventory/srv]        | 库存微服务，负责库存查询、扣减、回滚等库存能力                       
| [app/order/srv]            | 订单微服务，负责订单、购物车，并调用 goods/inventory；有事务/边界层迹象  
| [app/goshop/api]           | 面向用户侧的 HTTP API 网关，聚合后端 gRPC 服务               
| [app/goshop/admin]         | 后台管理 HTTP 服务                                  
| [app/pkg]                  | 应用层共享包，如统一 gRPC client、业务错误码、Gorm options、翻译器 




架构逻辑
三层架构，api 层，app 层，gmicro 层,cmd 层，configs 层，pkg 层，tools 层,

1. api 存放与外部交互的接口及 proto 文件，metadata 引用 kratos 发现所有 gRPC 服务接口，user 存放 user 模块的接口定义
2. app 具体服务相关的实现，shop 举例 shop http 服务 同级的还可以创建其他文件夹，admin shop 下的 admin 侧 http 服务
   同级的还可以创建其他文件夹，config 配置文件，controller 表示层，pkg 服务共通的包，code 服务的错误码，options
   服务的选项，translator 翻译器，gin 相关的翻译器，user 举例 user gRPC 服务，可以同时建立 http 服务，参考 shop，client 用于本地测试
   user rpc 服务的客户端，srv 服务相关的实现，config 服务的配置项，Log error 等子服务的相关逻辑全部注册到配置中，controller
   表示层，user user 模块的表示层，data 数据访问层：数据库，缓存，消息队列等，无业务逻辑，v1 数据库相关的操作，mock 数据库的
   mock 用于测试，service 业务逻辑层，v1 业务逻辑层的实现，app.go user 服务生成逻辑，rpc.go user 模块的 rpc 服务 初始化逻辑

5. gmicro 微服务相关包，app 服务启动相关的结构体，app.go 这个 app 是 GRPC，服务名称，注册中心等的集合，code 有一些公用的错误码，core
   底层共通核心的包，metric 服务监控相关的逻辑 使用了 prometheus，trace 链路追踪，采用 opentemlemetry，registry
   服务注册中心相关逻辑，consul 基于 consul 的服务注册中心相关逻辑，server rpc 服务的初始化配置，restserver http
   服务的初始化配置，中间件 http 服务的中间件，auth 认证相关中间件，包括 jwt，cache，basic 这里是校验认证的中间件，jwt.go
   生成token的中间件，tracing.go 只是向gin中封装了 jeager 的 span 的相关信息，待上报的时候，才能入库，pprof http 服务的
   pprof 相关逻辑，validation http 服务的参数校验，rpcserver rpc 服务的初始化配置，client.go rpc 客户端的初始化配置，server.go
   rpc 服务端的初始化配置，clientinterceptors 客户端的拦截器：超时连接器，监控 prometheus，resolver 服务发现相关的逻辑 解析器
   direct 直连，加权轮询的时候使用 discovery 服务发现，负载均衡 builder.go 服务发现的构建器 resolver.go
   服务发现的解析器,负载均衡的逻辑在这里实现 UpdateState 核心 selector 重写 grpc 接口，具体服务相关的实现 node gRPC 服务节点
   direct 直连节点 ewma ewma算法节点，用于实现 p2c 负载均衡策略 p2c 负载均衡 [Power of Two Random Choices] 算法 random
   负载均衡随机算法 wrr 负载均衡 加权轮询 [Weighted Round Robin] 算法
6. serverinterceptors 服务端的拦截器：超时，crash恢复，监控

cmd启动添加配置文件路径:






准备
nacos
consul
mysql
redis
es


运行启动流程:
cmd开始启动服务
    1.读取configs下的配置文件,在Edit Configurations 中的 Program arguments 写入 --config=./configs/user/srv.yaml




需要设置环境变量来控制生产还是开发环境



#nacos配置sentinel配置项,可以查看sentinel的流控规则
[
{
"resource":"/User/GetUserList", #控制接口
"tokenCalculateStrategy":0,     #计算方式，0表示直接使用请求数进行流控，1表示使用平均响应时间进行流控
"controlBehavior":0,            #控制策略,0表示直接拒绝，1表示匀速排队等待
"threshold":20,                 #阈值:可通过的请求数
"statIntervalInMs":1000         #统计时间窗口，单位毫秒
}
]